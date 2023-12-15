package ConnQuic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"net"
	"os"
	"sync"
	"time"

	. "aoanima.ru/Logger"
	quic "github.com/quic-go/quic-go"
)

// type КартаСессий struct {
//    sync.RWMutex
//		СессииСервисов  quic.Connection       // кладём соовтетсвие сессий и потоков
//		ОчередьПотоков *ОчередьПотоков // все потоки всех сессий кладём в одну очередь
//	}
/*
Создаём новый Серверв
сервер := &СхемаСервера{
			Имя:         "SynQuic",
			Адрес:       "localhost:4242",
			КартаСессий: КартаСессий{},
		}

Вызываем мтед Соединиться? в него передаёт сообщение для регистрации клиента на сервере , с перечнем маршрутов
Клиент.Соединиться(Адрес string, сообщениеРегистрации Сообщение)

После установки соединения открываем 1 поток, и отправляем в него сообщение, сервер регистрирует и отвечает что всё ок.
Этот поток не кладём в очередь потоков

Дальше сервер Открывает поток,


*/

// где string это адрес или имя сервиса.. лучше наверное адрес
type ИмяСервера string
type СхемаСервера struct {
	Имя          ИмяСервера
	Адрес        string
	ДанныеСессии ДанныеСессии
}

type ДанныеСессии struct {
	Блок           *sync.RWMutex
	Сессия         quic.Connection
	Потоки         []quic.Stream // массив потому что клиенту не нужна очередь, тут просто хранятся все принятые потоки от SynQuic
	СистемныйПоток quic.Stream   // сохраним первый поток как сервисный, ля отправки каких то уведомлений... проверки загруженности или ещё что то
}

// Массив подключений, вдруг понадобится открыть несоклько подключений к одному серверу
type Клиент map[ИмяСервера][]*СхемаСервера

// func (к Клиент) Соединиться(сервер СхемаСервера, обработчикСообщений func(поток quic.Stream, сообщение Сообщение)) {
// Содеинится, это для сервисов, которые сами не инициируют потоки, а принимают входящие потоки от SynQuic

func (клиент Клиент) Соединиться(
	сервер *СхемаСервера,
	сообщениеРегистрации Сообщение,
	ОбработчикОтветаРегистрации func(сообщение Сообщение),
	ОбработчикЗапросовСервера func(поток quic.Stream, сообщение Сообщение)) {

	конфигТлс, err := КлиентскийТлсКонфиг()
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	// Адрес = "localhost:4242"
	if сервер.Адрес == "" {
		сервер := &СхемаСервера{
			Имя:          "SynQuic",
			Адрес:        "localhost:4242",
			ДанныеСессии: ДанныеСессии{},
		}
		клиент["SynQuic"] = append(клиент["SynQuic"], сервер)
	}

	Конифгурация := &quic.Config{
		KeepAlivePeriod: 30 * time.Second,
		MaxIdleTimeout:  360 * time.Second,
	}
	// Tracer := func(ctx context.Context, p logging.Perspective, connID quic.ConnectionID) *logging.ConnectionTracer {
	// 	filename := fmt.Sprintf("server_%s.qlog", connID)
	// 	f, err := os.Create(filename)
	// 	if err != nil {
	// 		Ошибка(" %+v \n", err)
	// 	}
	// 	Инфо("Creating qlog file %s.\n", filename)
	// 	return qlog.NewConnectionTracer(NewBufferedWriteCloser(bufio.NewWriter(f), f), p, connID)
	// }
	// Конифгурация.Tracer = Tracer
	var ошибкаСоединения error
	var сессия quic.Connection
	Инфо(" соединяемся с SynQuic %+v \n", сессия)
	for сессия == nil {
		сессия, ошибкаСоединения = quic.DialAddr(context.Background(), сервер.Адрес, конфигТлс, Конифгурация)
		if err != nil {
			Ошибка(" не удаётся покдлючиться к серверу  %+v \n", ошибкаСоединения, сессия)
			time.Sleep(10 * time.Second)
		}
	}
	// Добавляем сессию соединения с сервером в карту сессий
	сервер.ДанныеСессии.Сессия = сессия
	// слушаем запрос на открытиые потока от сервера
	go сервер.ДанныеСессии.ОжиданиеВходящегоПотока(ОбработчикЗапросовСервера)

	// этот поток не добавляем в очередь потоков, в него мы писать ничгео не будем в адльнейшем, отпраавляем сейчас только регистрационное сообщение с маршрутами котоые обрабатывает сервис
	системныйПоток, err := сессия.OpenStream()
	// системныйПоток, err := сессия.OpenStreamSync(context.Background())
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		Ошибка(" не удаётся открыть системный поток %+v \n", err)
		return
	}
	// сохраним первый поток как сервисный, ля отправки каких то уведомлений... проверки загруженности или ещё что то
	сервер.ДанныеСессии.СистемныйПоток = системныйПоток
	сообщ, _ := Кодировать(сообщениеРегистрации)
	системныйПоток.Write(сообщ)                     // пишем сообщение в поток
	ответ := клиент.ЧитатьСообщение(системныйПоток) // ожидаем ответа
	ОбработчикОтветаРегистрации(ответ)              // обрабатываем ответ
}

// func инициализацияКлиента() {
// 	клиент := make(Клиент)
// 	сервер := &СхемаСервера{
// 		Имя:          "SynQuic",
// 		Адрес:        "localhost:4242",
// 		ДанныеСессии: ДанныеСессии{},
// 	}
// 	// каналСообщений := make(chan Сообщение, 5)
// 	// сообщениеРегистрации := Сообщение{}
// 	клиент.Соединиться(сервер, каналСообщений)

// 	// перед отправкой регистрационного сообщения нужно запустить ожидание входящих потоков, потому что при регистрации сервер сразу открывает исходящий поток
// 	go сервер.ДанныеСессии.ОжиданиеВходящегоПотока(ОбработчикЗапросовСервера)

// 	// системныйПоток.Write(сообщениеРегистрации)      // пишем сообщение в поток
// 	// ответ := клиент.ЧитатьСообщение(системныйПоток) // ожидаем ответа
// 	// статусРегистрации(ответ)                        // обрабатываем ответ

// }

//  Ожидает сообщения , декодирует и отдаёт в канал

// читает ожно сообщение, декодирует и возвращает его , и завершает свою работу.
func (клиент Клиент) ЧитатьСообщение(поток quic.Stream) Сообщение {
	длинаСообщения := make([]byte, 4)
	var прочитаноБайт int
	var err error

	// for {
	прочитаноБайт, err = поток.Read(длинаСообщения)
	Инфо(" длинаСообщения %+v , прочитаноБайт %+v \n", длинаСообщения, прочитаноБайт)

	if err != nil {
		Ошибка(" прочитаноБайт %+v  err %+v \n", прочитаноБайт, err)
		return Сообщение{}
	}

	// получаем число байткоторое нужно прочитать
	длинаДанных := binary.LittleEndian.Uint32(длинаСообщения)

	Инфо(" длинаДанных  %+v \n", длинаДанных)
	Инфо(" длинаСообщения %+v ,  \n прочитаноБайт %+v ,  \n длинаДанных %+v \n", длинаСообщения,
		прочитаноБайт, длинаДанных)

	//читаем количество байт = длинаСообщения
	// var запросКлиента ЗапросКлиента
	сообщениеБинарное := make([]byte, длинаДанных)
	прочитаноБайт, err = поток.Read(сообщениеБинарное)
	if err != nil {
		Ошибка("Ошибка при десериализации структуры: %+v ", err)
	}

	if длинаДанных != uint32(прочитаноБайт) {
		Ошибка("Количество прочитаных байт не ранво длине данных :\n длинаДанных %+v  <> прочитаноБайт %+v ", длинаДанных, прочитаноБайт)
	} else {

		сообщение, err := ДекодироватьПакет(сообщениеБинарное)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		return сообщение

		// break
	}
	// каналПолученияСообщений <- пакетОтвета
	return Сообщение{}
	// }
}

func (данныеСессий ДанныеСессии) ЧитатьСообщения(поток quic.Stream, обработчикСообщений func(поток quic.Stream, сообщение Сообщение)) {

	длинаСообщения := make([]byte, 4)
	var прочитаноБайт int
	var err error

	for {
		прочитаноБайт, err = поток.Read(длинаСообщения)
		// Инфо(" длинаСообщения %+v , прочитаноБайт %+v \n", длинаСообщения, прочитаноБайт)

		if err != nil {
			Ошибка(" прочитаноБайт %+v  err %+v \n", прочитаноБайт, err)
			break
		}

		// получаем число байткоторое нужно прочитать
		длинаДанных := binary.LittleEndian.Uint32(длинаСообщения)

		Инфо(" длинаДанных  %+v \n", длинаДанных)
		Инфо(" длинаСообщения %+v = длинаДанных %+v \n прочитаноБайт %+v ,  \n ", длинаСообщения,
			длинаДанных, прочитаноБайт)

		//читаем количество байт = длинаСообщения
		// var запросКлиента ЗапросКлиента
		сообщениеБинарное := make([]byte, длинаДанных)
		прочитаноБайт, err = поток.Read(сообщениеБинарное)
		if err != nil {
			Ошибка("Ошибка при десериализации структуры: %+v ", err)
		}

		if длинаДанных != uint32(прочитаноБайт) {
			Ошибка("Количество прочитаных байт не ранво длине данных :\n длинаДанных %+v  <> прочитаноБайт %+v сообщениеБинарное %+s", длинаДанных, прочитаноБайт, сообщениеБинарное)
		} else {

			сообщение, err := ДекодироватьПакет(сообщениеБинарное)
			if err != nil {
				Ошибка("  %+v \n", err)
			}
			go обработчикСообщений(поток, сообщение)
		}

	}

}

func (данныеСессий ДанныеСессии) ОжиданиеВходящегоПотока(ОбработчикЗапросовСервера func(поток quic.Stream, сообщение Сообщение)) {
	for {
		Инфо(" ОжиданиеВходящегоПотока  \n")
		// принимаем запрос на открытиые потока, и добавляем его в очередь потоков
		поток, err := данныеСессий.Сессия.AcceptStream(context.Background())
		if err != nil {
			Ошибка("  %+v \n", err)
		}

		// Инфо(" Запрос на открытие нового потока id нового потока %+v \n", поток.StreamID())
		//! если это клиент, то хачем мне хранить потоки на клиенте в очереди, клиент не будет сам отправлять серверу сообщения...
		//! можно не закрывать первый поток, пометив его например как системный, тоесть через него клиент будет отправлет какието уведомления серверу.
		//!
		данныеСессий.Потоки = append(данныеСессий.Потоки, поток)
		go данныеСессий.ЧитатьСообщения(поток, ОбработчикЗапросовСервера)

	}
}

func СтатусРегистрации(сообщение Сообщение) {

	Инфо(" статусРегистрации %+v \n", сообщение)

	if len(сообщение.Ответ) > 0 {
		for сервис, данные := range сообщение.Ответ {
			Инфо("сервис  %+v данные  %+v \n", сервис, данные)
		}
	}

}
func КлиентскийТлсКонфиг() (*tls.Config, error) {

	caCert, err := os.ReadFile(ДирректорияЗапуска + "/cert/ca.crt")
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	// Инфо("Корневой сертфикат создан?  %v ", ok)

	cert, err := tls.LoadX509KeyPair(ДирректорияЗапуска+"/cert/server.crt", ДирректорияЗапуска+"/cert/server.key")
	if err != nil {
		Ошибка(" %s", err)
	}

	return &tls.Config{
		// InsecureSkipVerify: true,
		RootCAs: caCertPool,

		Certificates: []tls.Certificate{cert},
		// NextProtos:   []string{"h3", "quic", "websocket"},
		NextProtos: []string{"h3", "quic", "websocket"},
	}, nil
}
