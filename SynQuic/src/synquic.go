package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	_ "net/http/pprof"

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/Logger"
	. "aoanima.ru/QErrors"
	"github.com/quic-go/quic-go"
)

var (
	ВходящийПорт  = ":81"
	ИсходящийПорт = ":82"
	// каналОтправкиОтветов     = make(chan ОтветКлиенту, 10)
	// КаналыИсходящихСообщений = map[string]chan ОтветКлиенту{}
)

// type ОтветКлиенту struct {
// 	Сервис    []byte
// 	Ответ     []byte
// 	ИдКлиента []byte
// }

//	type ЗапросКлиента struct {
//		Сервис       []byte
//		Запрос       *ЗапросОтКлиента
//		ИдКлиента    uuid.UUID
//		ТокенКлиента []byte // JWT сериализованный
//	}
//
//	type ЗапросОтКлиента struct {
//		СтрокаЗапроса string
//		Форма         map[string][]string
//		Файл          string
//	}
type Конфигурация struct{}

// var каталогСтатичныхФайлов string
var Конфиг = &Конфигурация{}

func init() {
	Инфо(" проверяем какие аргументы переданы при запуске, если пусто то читаем конфиг, если конфига нет то устанавливаем значения по умолчанию %+v \n")

	// каталогСтатичныхФайлов = "../../HTML/static/"
	ЧитатьКонфиг(Конфиг)
}
func main() {
	go func() {
		http.ListenAndServe("localhost:6061", nil)
	}()
	// Вероятно нужно откуда то получить список Сервисов с которомы предстоит общаться
	//  Или !!!! ОбработчикВходящихСообщений
	//обработчикСистемныхСообщений - функция которая обрабатывает сигналы от сервисов.
	ЗапуститьSynQuicСервер("localhost:4242", обработчикСообщенийHTTPсервера, обработчикСистемныхСообщений)

	Инфо(" %s", "запустили сервер")
	// ЗапуститьСерверИсходящихСообщений()
}

// var Адрес = "localhost:4242"
// Запускаем сервер который слушает на адресе,
// принимает соединиеие, и отправляет его в обработчик Сессии
// обработчикСообщенийHTTPсервера - функция в которую передаётся сообщение из HTTP сервера, от клиента, реализцется непосредственно в саомо приложени в сервисе выступабщим в качестве менеджера сообщений, в данном случае SynQuic

// Запускаем SynQuic сервер
func ЗапуститьSynQuicСервер(Адрес string,
	обработчикСообщенийHTTPсервера func(сообщение Сообщение) (Сообщение, error),
	ОбработчикСистемныхСообщений func(поток quic.Stream, сообщение Сообщение)) {
	кофигТлс, err := СерверныйТлсКонфиг()
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	Конифгурация := &quic.Config{
		KeepAlivePeriod: 30 * time.Second,
		MaxIdleTimeout:  360 * time.Second,
	}

	listener, err := quic.ListenAddr(Адрес, кофигТлс, Конифгурация)
	if err != nil {
		Ошибка(" %+v ", err)
	}
	Инфо("  Запустил SynQuic сервер %+v \n", Адрес)

	for {
		сессия, err := listener.Accept(context.Background())

		if err != nil {
			Ошибка(" %+v ", err)
		}
		// go ЧитатьСистемныйПоток(сессия, ОбработчикСистемныхСообщений)

		каналСообщений := make(chan Сообщение, 5)
		системныйПоток, err := сессия.AcceptStream(context.Background()) // условно назвал Системный поток, через него сервисы обмениваются системными сообщениями, будто то запрос на переподключение и ли обмен метриками или ещё что то, тот же пинг
		if err != nil {
			Ошибка("  %+v \n", err)
		}

		go ЧитатьСообщения(системныйПоток, каналСообщений)

		go func() {
			for сообщение := range каналСообщений {
				if сообщение.Регистрация {
					if сообщение.Сервис == "КлиентСервер" {
						РегистрацияHTTPсервера(сессия, системныйПоток, &сообщение, обработчикСообщенийHTTPсервера)
					} else {
						РегистрацияСервиса(сессия, системныйПоток, &сообщение)
					}
				} else {
					// Инфо(" вызываем ОбработчикСистемныхСообщений %+v \n", сообщение)
					ОбработчикСистемныхСообщений(системныйПоток, сообщение)
				}
			}
		}()

		// НУЖНО СДЕЛАТЬ обработчик для входящих стримов и обработки регистрации сервисов, после чего открывать исходящий поток.

		// go ОбработчикСессии(сессия, обработчикСообщений)
	}
}
func обработчикСистемныхСообщений(системныйПоток quic.Stream, сообщение Сообщение) {
	if сообщение.Пинг {
		понг := сообщение
		понг.Пинг = false
		понг.Понг = true
		отправить, err := Кодировать(понг)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		// Инфо(" отправляем понг  %+v \n", понг  )
		системныйПоток.Write(отправить)
	}
	// Инфо(" обработчикСистемныхСообщений : пришёл сигнал в системный поток от сервиса, нужно придумать какие сигналы и как будут обрабатываться%+v \n", сообщение)
}

func РегистрацияHTTPсервера(сессия quic.Connection, системныйПоток quic.Stream, сообщение *Сообщение, обработчикСообщенийHTTPсервера func(сообщение Сообщение) (Сообщение, error)) {
	// TODO добавить обработчик для регистрации HTTP сервера? т.к. сервер инциирует открытиые новых потоков то он регистриаруется чуть по другом, чтобы не загромоэдать функцию ергнситарции сервисов, реализуем логику тут:

	// очередьПотоков := НоваяОчередьПотоков()
	// новаяСессия := КартаСессий{
	// 	Соединение:     сессия,
	// 	ОчередьПотоков: очередьПотоков,
	// 	СистемныйПоток: поток,
	// }

	Инфо(" РегистрацияHTTPсервера %+v \n", сообщение)

	номерСессии := НомерСессии(len(АктивныеHTTPСесии))
	новаяСессия := HTTPСессии{
		Блок:       &sync.RWMutex{},
		Соединение: сессия,
		// Потоки:     []quic.Stream{поток},
		Потоки: []quic.Stream{},
	}
	АктивныеHTTPСесии[номерСессии] = новаяСессия

	данныеОТвета := Ответ{
		"КлиентСервер": ОтветСервиса{
			ЗапросОбработан: true,
		},
	}
	сообщениеОтвет := Сообщение{
		Регистрация:  true,
		Сервис:       "SynQuic",
		ИдКлиента:    сообщение.ИдКлиента,
		УИДСообщения: сообщение.УИДСообщения,
		Ответ:        данныеОТвета,
	}
	ОтправитьСообщение(системныйПоток, сообщениеОтвет)
	// ответ, err := Кодировать(сообщениеОтвет)
	// if err != nil {
	// 	Ошибка("  %+v \n", err)
	// }
	// Инфо("отправляем ответ  регистрации  %+v \n", string(ответ) )
	// поток.Write(ответ)
	go ОжиданиеВходящихПотоковHTTP(&новаяСессия, обработчикСообщенийHTTPсервера)
	ПульсСессии()
}

func ОжиданиеВходящихПотоковHTTP(сессия *HTTPСессии, обработчикСообщенийHTTPсервера func(сообщение Сообщение) (Сообщение, error)) {
	for {
		// принимаем поток от http сервера
		поток, err := сессия.Соединение.AcceptStream(context.Background())
		if err != nil {
			Ошибка("Удалить сессию из АктивныеHTTPСесии %+v сессия.Соединение %+v \n", err, сессия.Соединение.ConnectionState())

			break
		} else {
			Инфо(" пришёл новый запрос на открытие поток  \n")
			сессия.Блок.RLock()
			сессия.Потоки = append(сессия.Потоки, поток)
			сессия.Блок.RUnlock()

			go ЧитатьHTTPПоток(поток, обработчикСообщенийHTTPсервера)
		}

		// читаем сообщения из поток

	}
}

func ЧитатьHTTPПоток(поток quic.Stream, обработчикСообщенийHTTPсервера func(сообщение Сообщение) (Сообщение, error)) {
	// сообщение := ЧитатьСообщение(поток)

	каналСообщений := make(chan Сообщение, 5)

	Инфо("ЧитатьHTTPПоток читаю поток  %+v \n", поток.StreamID())

	go ЧитатьСообщения(поток, каналСообщений)

	for сообщение := range каналСообщений {
		Инфо("ЧитатьHTTPПоток пришёл запрос от клиентаы  %+v \n", сообщение)
		ответ, err := обработчикСообщенийHTTPсервера(сообщение)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		Инфо(" отправляем ответ в HTTP поток,  %+v \n", поток.StreamID())
		статус := ОтправитьСообщение(поток, ответ)
		if статус.Код != Ок {
			Ошибка("  %+v \n", статус)
		}

	}

}
