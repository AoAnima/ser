package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"sync"

	. "aoanima.ru/logger"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

// клиент это какойто сервис который установил содинение , для каждого ответа должен быть свой канал, в который будет писаться сообщение
func обработчикИсходящихСоединений(клиент net.Conn) { //, данныеДляОтвета chan []byte

	go РукопожатиеИсходящегоКанала(клиент)
	// for данныеДляОтвета := range КаналыИсходящихСообщений {
	// 	Инфо(" %+v \n", данныеДляОтвета)

	// 	КаналыИсходящихСообщений[string(данныеДляОтвета.Сервис)]<- данныеДляОтвета

	// }

}

type СтруктураДанных struct {
	ОбъектДанных interface{}
}
type Отпечаток struct {
	Сервис      string
	КаналОтвета chan interface{}
	Маршруты    map[string]map[string]interface{}
}

var mutex sync.Mutex
var Маршрутизатор = make(map[string]*Отпечаток)

// Маршрутизатор = map[string]*Отпечаток{
// 	"ОтветКлиенту": Отпечаток{
// 		Сервис: "КлиентСервер",
// 		КаналОтвета: make(chan interface{}),
// 		Маршруты: map[string]map[string]interface{}{
// 			"ОтветКлиенту": map[string]interface{}{
// 				"HTML": "string",
// 				"JSON": "string",
// 			},
// 		},
// 	},
// }

func РегистрацияСервиса(отпечатокСервиса Отпечаток) {
	отпечатокСервиса.КаналОтвета = make(chan interface{}, 10)
	for маршрут, _ := range отпечатокСервиса.Маршруты {
		mutex.Lock()
		Маршрутизатор[маршрут] = &отпечатокСервиса
		mutex.Unlock()
	}
}

// исключительно для рукопожатия и сохранения в пул Сервисов/ когда сервис присылает запрос на рукопожатие, он присылает маршруты которые он обрабатывает !!!
// напрмиер сервис каталогов обрабатывает запросы  начинающиеся на /catalog
func РукопожатиеИсходящегоКанала(клиент net.Conn) {
	длинаСообщения := make([]byte, 4)
	// рукопожатие := [4]byte{}
	var прочитаноБайт int
	var err error
	for {
		// получаем длину сообщения рукопожатия
		прочитаноБайт, err = клиент.Read(длинаСообщения)
		Инфо("  %+v \n", прочитаноБайт)
		// читаем всё остальное сообщение
		// создадим буфер куда поместим сообщение
		// сообщениеРукопожатия := make([]byte, binary.LittleEndian.Uint32(длинаСообщения))

		// copy(рукопожатие[0:], длинаСообщения[:4])

		// if длинаСообщенияФикс == [4]byte{240, 159, 164, 157} { //"🤝"
		// 	Инфо(" %+v \n", string(длинаСообщения))
		сообщениеОтСервиса := make([]byte, binary.LittleEndian.Uint32(длинаСообщения))
		_, err = клиент.Read(сообщениеОтСервиса)
		if err != nil {
			Ошибка("  %+v \n", err)
		}

		ОтпечатокСервиса := Отпечаток{}

		err := jsoniter.Unmarshal(сообщениеОтСервиса, &ОтпечатокСервиса)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		РегистрацияСервиса(ОтпечатокСервиса)
		// КаналыИсходящихСообщений[ОтпечатокСервиса.Сервис] = make(chan ОтветКлиенту, 10)

		// Инфо(" ЗапросОтКлиента %+s \n", ЗапросОтКлиента)

		// _, err = клиент.Read(длинаСообщения)
		// 	if err != nil {
		// 		Ошибка(" прочитаноБайт %+v  err %+v \n", прочитаноБайт, err)
		// 	}
		// 	ИмяСервиса := make([]byte, binary.LittleEndian.Uint32(длинаСообщения))
		// 	_, err = клиент.Read(ИмяСервиса)
		// 	if err != nil {
		// 		Ошибка(" прочитаноБайт %+v  err %+v \n", прочитаноБайт, err)
		// 	}
		// 	Инфо("ИмяСервиса %+v \n", string(ИмяСервиса))
		// 	strings.Split(string(ИмяСервиса), ".")
		// 	continue
		// }
	}
}

// Обрабатываем запрос, отправялем в базу данных, и в маршрутизатор который отправит запрос в соответствующий сервис
func ОбработатьПакет(пакет []byte) {
	Инфо(" ОбработатьПакет пакет %+v \n", пакет)

}

func МаршрутизаторЗапросов() {

}

func ДекодироватьПакет(пакет []byte) {
	Инфо(" ДекодироватьПакет пакет %+s \n", пакет)

	// ⁝  [3]byte{226, 129, 157, 0} - разделить между сообщениями за которым может следовать новое сообщение, первые 4 байта которого определяет длину
	// почму запрос клиента, тут могут прийти данные от каких то сервисов с ответом, результатом обработка запроса.....
	// Значит нужно определять от какого из сервисов пришёл запрос для начала, а потом уже декодировать данные в соответсвующую структуру ?????

	var ЗапросОтКлиента = ЗапросКлиента{
		Сервис:    []byte{},
		Запрос:    ЗапросОтКлиента{},
		ИдКлиента: uuid.UUID{},
	}

	err := jsoniter.Unmarshal(пакет, &ЗапросОтКлиента)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	Инфо(" ЗапросОтКлиента %+s \n", ЗапросОтКлиента)

	go СохранитьЗапрос(ЗапросОтКлиента)

	АнализЗапроса(ЗапросОтКлиента)

}

var СписокСервисов = map[string]string{}

func АнализЗапроса(СтрокаЗапроса ЗапросКлиента) {

	// СтрокаЗапроса.Запрос = string(СтрокаЗапроса.Запрос)

	параметрыЗапроса, err := url.ParseQuery(string(СтрокаЗапроса.Запрос.Строка))
	Инфо(" %+v \n", параметрыЗапроса)
	if err != nil {
		fmt.Println("Ошибка при парсинге СтрокаЗапроса запроса:", err)
		return
	}
	// Анап

	Инфо("СтрокаЗапроса.Запрос.Форма %+v \n", СтрокаЗапроса.Запрос.Форма)

	// Анализируем в какой сервис отправить запрос
	// например присутствует строка category - знаичт отправляем в сервис отвечающий за категории

	ОтправитЗапросВСервис(параметрыЗапроса)

}

func ОтправитЗапросВСервис(параметрыЗапроса url.Values) {

}

func СохранитьЗапрос(запрос ЗапросКлиента) {
	sql := fmt.Sprintf("INSERT INTO querys (id, query, service) VALUES (%s, %s, %s)", запрос.ИдКлиента.String(), запрос.Запрос, запрос.Сервис)
	Инфо(" Пишем в бд >> %+v \n", sql)

}

// Обрабатывает только запросы полученный от сервисов, в ответ ничего не отправляет
func обработчикВходящихСообщений(клиент net.Conn) {

	длинаСообщения := make([]byte, 4)
	var прочитаноБайт int
	var err error
	for {
		прочитаноБайт, err = клиент.Read(длинаСообщения)
		Инфо(" длинаСообщения %+v , прочитаноБайт %+v \n", длинаСообщения, прочитаноБайт)

		if err != nil {
			Ошибка(" прочитаноБайт %+v  err %+v \n", прочитаноБайт, err)
			break
		}

		// получаем число байткоторое нужно прочитать
		длинаДанных := binary.LittleEndian.Uint32(длинаСообщения)

		Инфо(" длинаДанных  %+v \n", длинаДанных)
		Инфо(" длинаСообщения %+v ,  \n прочитаноБайт %+v ,  \n длинаДанных %+v \n", длинаСообщения,
			прочитаноБайт, длинаДанных)

		//читаем количество байт = длинаСообщения
		// var запросКлиента ЗапросКлиента
		пакетЗапроса := make([]byte, длинаДанных)
		прочитаноБайт, err = клиент.Read(пакетЗапроса)
		if err != nil {
			Ошибка("Ошибка при десериализации структуры: %+v ", err)
		}
		if длинаДанных != uint32(прочитаноБайт) {
			Ошибка("Количество прочитаных байт не ранво длине данных :\n длинаДанных %+v  <> прочитаноБайт %+v ", длинаДанных, прочитаноБайт)
		}

		// Запускаем для пакета отдельную горутину, т.к. в ожном соединении будет приходить множество запросов от разных клиентов, и обработчик будт всегда один

		go ДекодироватьПакет(пакетЗапроса)

	}

}

// func ТестВХодящихСообщенийСнизкойСкоростью(клиент net.Conn) {
// 	длинаСообщения := make([]byte, 100)
// 	var прочитаноБайт int
// 	var err error
// 	ВсегоПрочитано := 0
// 	socket := клиент
// 	fd, err := socket.File()

// 	sock := syscall.Handle(fd.Fd())
// 	level := syscall.SOL_SOCKET
// 	name := syscall.SO_RCVBUF

// 	in, err := syscall.GetsockoptInt(sock, level, name)
// 	if err != nil {
// 		fmt.Println("Ошибка при получении значения опции сокета:", err)
// 		return
// 	}
// 	Инфо(" in %+v \n", in)
// 	for {
// 		прочитаноБайт, err = клиент.Read(длинаСообщения)
// 		if err != nil {
// 			Ошибка("  %+v \n", err.Error())
// 		}

// 		ВсегоПрочитано = ВсегоПрочитано + прочитаноБайт
// 		Инфо(" ВсегоПрочитано %+v , прочитаноБайт %+v \n", ВсегоПрочитано, прочитаноБайт)
// 		time.Sleep(1 * time.Second)
// 	}
// }
