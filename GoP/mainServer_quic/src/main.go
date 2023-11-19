package main

import (
	"context"
	"errors"
	"net/http"
	"sync"

	_ "net/http/pprof"

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/logger"
	quic "github.com/quic-go/quic-go"
)

var КартаSynQuic = make(HTTPКлиент)

type HTTPКлиент map[ИмяСервер]struct {
	*sync.RWMutex
	Сессии         map[НомерСессии]*СхемаСервераHTTP
	НеПолныеСессии map[НомерСессии]int
}

type СхемаСервераHTTP struct {
	Имя   ИмяСервер
	Адрес string
	*sync.RWMutex
	Соединение     quic.Connection
	СистемныйПоток quic.Stream
	ОчередьПотоков *ОчередьПотоков
}

func main() {
	/* каналЗапросовОтКлиентов - передаём этот канал в в функци  ЗапуститьСерверТЛС , когда прийдёт сообщение из браузера, функция обработчик запишет данные в этот канал
	 */
	каналЗапросовОтКлиентов := make(chan http.Request, 10)
	/*
	   Запускаем сервер передаём в него канал, в который запишем обработанный запрос из браузера
	*/
	go ЗапуститьСерверТЛС(каналЗапросовОтКлиентов)
	go SynQuicСоединение(каналЗапросовОтКлиентов)
	Инфо(" %s", "запустили сервер")
	/* Инициализирум сервисы коннектора передадим в них канал, из которого Коннектор будет читать сообщение, и отправлять его в synqTCP  */

	ЗапуститьWebСервер()

}

func SynQuicСоединение(каналЗапросов chan http.Request) {
	сервер := &СхемаСервераHTTP{
		Имя:            "SynQuic",
		Адрес:          "localhost:4242",
		RWMutex:        &sync.RWMutex{},
		ОчередьПотоков: &ОчередьПотоков{},
	}
	сообщениеРегистрации := Сообщение{
		Сервис:      "КлиентСервер",
		Регистрация: true,
		Маршруты:    []Маршрут{},
	}

	конфигТлс, err := КлиентскийТлсКонфиг("root.crt")
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	// Адрес = "localhost:4242"

	сессия, err := quic.DialAddr(context.Background(), сервер.Адрес, конфигТлс, &quic.Config{})
	if err != nil {
		Ошибка(" не удаётся покдлючиться к серверу  %+v \n", err)
		return
	}
	поток, err := сессия.OpenStream()
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	сервер.Соединение = сессия
	сервер.СистемныйПоток = поток // первый поток помечаем как системный, потому что synquic кладёт первые потоки в системные
	ДобавитьСессию(сервер)
	err = ОтправитьСообщение(поток, сообщениеРегистрации)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	// каналОтвета chan Сообщение
	// for сообщениеОтКлиента := range каналЗапросов {

	// 	go ОтправитьЗапросВОбработку(сообщениеОтКлиента,  )

	// }

}

func ДобавитьСессию(сервер *СхемаСервераHTTP) {
	КартаSynQuic[сервер.Имя] =  НУЖНО ДОДЕЛАТЬ ЭТУ ШТУКУ
	// КартаSynQuic[сервер.Имя] = make(map[НомерСессии]*СхемаСервераHTTP)
	// КартаSynQuic[сервер.Имя][0] = сервер

}

func ПолучитьSynQuicПотокДляОтправки() (ПотокСессии, error) {

	for _, схема := range КартаSynQuic {
		// надём в любой сессии поток и вернём его
		for номерСессии, схемаСессии := range схема {
			if поток := схемаСессии.ОчередьПотоков.Взять(); поток != nil {
				ПотокСессии := ПотокСессии{
					НомерСессии: номерСессии,
					Поток:       поток,
				}
				return ПотокСессии, nil
			}
		}
	}
	// если поток не найден , то попытаемся создать в любой не полной сессиия

	return ПотокСессии{}, errors.New("не найден поток")
}

func ОбработчикОтветаРегистрации(сообщение Сообщение) {
	Инфо("  ОбработчикОтветаРегистрации %+v \n", сообщение)
}

// обработчик сообщений от synqTCP
// func ОбработатьСообщение(поток quic.Stream, ВходящееСообщение Сообщение) {
// // СООБЩЕНИЕ ФОРМИРУЕТСЯ ТУТ

// 	Инфо(" ОбработатьСообщение %+v \n", ВходящееСообщение)

// 	Сообщение, err := Кодировать(ВходящееСообщение)
// 	if err != nil {
// 		Ошибка("  %+v \n", err)
// 	}
// 	// TODO: Реализум логику обработки запроса от клиента, и генерацию ответа

// 	каналОтправкиСообщений <- Сообщение
// }

func ЗапуститьСерверТЛС(каналЗапросов chan<- http.Request) {

	err := http.ListenAndServeTLS(":443",
		"cert/server.crt",
		"cert/server.key",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				обработчикЗапроса(w, r, каналЗапросов)
			}))

	if err != nil {
		Ошибка(" %s ", err)
	}

}
func обработчикЗапроса(w http.ResponseWriter, req *http.Request, отправитьЗапросВОбработку chan<- http.Request) {

	Инфо(" %s \n", *req)

	каналОтвета := make(chan Сообщение, 10)

	// /*  Тут мы читаем из канала  каналОтвета кторый храниться в карте клиенты , данные пишутся в канал  в функции ОтправитьОтветКлиенту */
	// Отправляем сырой запрос в функцию ОтправитьЗапросВОбработку
	отправитьЗапросВОбработку <- *req
	ответ := ОтправитьЗапросВОбработку(req)
	for данныеДляОтвета := range каналОтвета {
		Инфо("  %+v \n", данныеДляОтвета)
		// if данныеДляОтвета.Ответ != "" {
		// 	Инфо(" данныеДляОтвета.Ответ %+v \n", данныеДляОтвета.Ответ)

		// 	if f, ok := w.(http.Flusher); ok {
		// 		i, err := w.Write([]byte(данныеДляОтвета.Ответ))
		// 		Инфо("  %+v \n", i)
		// 		if err != nil {
		// 			Ошибка(" %s ", err)
		// 		}
		// 		f.Flush()
		// 		break
		// 	}
		// }
	}

}

// func ОбработчикОтветов(w http.ResponseWriter, каналОтветов <-chan Ответ) {

// 	Ответ := <-каналОтветов
// 	if Ответ.Сообщение != nil {
// 		w.Write([]byte(Ответ.Сообщение.(string)))
// 	}

// }

func ЗапуститьWebСервер() {
	err := http.ListenAndServe(":80", http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			// 	Инфо(" %s  %s \n", w, req)
			http.Redirect(w, req, "https://localhost:443"+req.RequestURI, http.StatusMovedPermanently)
		},
	))
	// err := http.ListenAndServe(":6060", nil)
	if err != nil {
		Ошибка(" %s ", err)
	}
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
}
