package main

<<<<<<< HEAD
=======
import (
	"net"
	"sync"
	// . "aoanima.ru/ConnQuic"
	// . "aoanima.ru/logger"
)

>>>>>>> 749006ec09c54c1e21404de823aefea1a35f2753
// клиент это какойто сервис который установил содинение , для каждого ответа должен быть свой канал, в который будет писаться сообщение

// type СтруктураДанных struct {
// 	ОбъектДанных interface{}
// }
// type ОтпечатокСервиса struct {
// 	Сервис         string
// 	КаналСообщения chan interface{}
// 	Маршруты       map[string]map[string]interface{}
// 	КлиентМьютекс  sync.Mutex
// 	// Клиент        []net.Conn
// 	Клиент ПулСоединений
// }
// type ПулСоединений struct {
// 	пулл chan net.Conn
// }

// var МаршрутизаторМьютекс sync.Mutex
// var Маршрутизатор = make(map[string]*ОтпечатокСервиса)

// func РегистрацияСервиса(отпечатокСервиса *ОтпечатокСервиса, Клиент net.Conn) {

// 	for маршрут, _ := range отпечатокСервиса.Маршруты {
// 		// проверим вдруг это сервис хъотчет открыть ещё одно соединение , если в маршрутизаторе уже есть такой маршрут от сервиса, то добавим соединение в маршрут
// 		if ЗарегистрированныйСервис, есть := Маршрутизатор[маршрут]; есть {
// 			ЗарегистрированныйСервис.КлиентМьютекс.Lock()
// 			ЗарегистрированныйСервис.Клиент.пулл <- Клиент
// 			// ЗарегистрированныйСервис.Клиент=append(ЗарегистрированныйСервис.Клиент, Клиент)
// 			ЗарегистрированныйСервис.КлиентМьютекс.Unlock()
// 		} else {

// 			отпечатокСервиса.Клиент.пулл = make(chan net.Conn, 10)
// 			// отпечатокСервиса.Клиент = append(отпечатокСервиса.Клиент, Клиент)
// 			отпечатокСервиса.КаналСообщения = make(chan interface{}, 10)
// 			отпечатокСервиса.Клиент.пулл <- Клиент

// 			МаршрутизаторМьютекс.Lock()
// 			Маршрутизатор[маршрут] = отпечатокСервиса
// 			МаршрутизаторМьютекс.Unlock()
// 			go отпечатокСервиса.ЧитатьКаналСообщений()
// 		}

// 	}
// 	Инфо(" Маршрутизатор %+v \n", Маршрутизатор)
// }

// func Кодировать(данныеДляКодирования interface{}) ([]byte, error) {

// 	b, err := jsoniter.Marshal(&данныеДляКодирования)
// 	if err != nil {
// 		Ошибка("  %+v \n", err)
// 		return nil, err
// 	}
// 	данные := make([]byte, len(b)+4)
// 	binary.LittleEndian.PutUint32(данные, uint32(len(b)))
// 	copy(данные[4:], b)
// 	return данные, nil

// }
// func ДекодироватьПакет(пакет []byte) Сообщение {
// 	Инфо(" ДекодироватьПакет пакет %+s \n", пакет)

// 	// var запросОтКлиента = ЗапросКлиента{
// 	// 	Сервис:    []byte{},
// 	// 	Запрос:    &ЗапросОтКлиента{},
// 	// 	ИдКлиента: uuid.UUID{},
// 	// }
// 	var Сообщение Сообщение

// 	// TODO тут лишний парсинг, нужно получить только URL patch чтобы определить сервис, которому принадлежит запрос, потому nxj дальше весь запрос опять сериализуйется

// 	err := jsoniter.Unmarshal(пакет, &Сообщение)
// 	if err != nil {
// 		Ошибка("  %+v \n", err)
// 	}
// 	Инфо(" Сообщение входящее %+s \n", Сообщение)

// 	return Сообщение
// }

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
