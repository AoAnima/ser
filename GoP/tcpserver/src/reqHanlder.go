package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	. "aoanima.ru/logger"
	"github.com/google/uuid"
)

var клиенты = make(map[[16]byte]Запрос)
var мьютекс = sync.Mutex{}

// каналЗапросов - исползуется для получения запросов от клиента, в запросе от клиента передаётся канал в который нужно отправить ответ клиенту
func ПодключитсяКМенеджеруЗапросов(каналЗапросов chan Запрос) {
	go ПодключитьсяКСерверуДляОтправкиСообщений(каналЗапросов)
	ПодключитсяКСерверуДляПолученияСообщений()
}

func ПодключитсяКСерверуДляПолученияСообщений() {
	caCert, err := os.ReadFile("cert/ca.crt")

	if err != nil {
		Ошибка(" %s ", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	Инфо("Корневой сертфикат создан?  %v ", ok)

	cert, err := tls.LoadX509KeyPair("cert/client.crt", "cert/client.key")
	if err != nil {
		Ошибка(" %s", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	// Подключение к TCP-серверу с TLS на localhost:8080
	количествоПопыток := 500
	задержка := 1 * time.Second
	var сервер *tls.Conn
	var errDial error
	for попытка := 1; попытка <= количествоПопыток; попытка++ {
		сервер, errDial = tls.Dial("tcp", "localhost:82", tlsConfig)
		if errDial != nil {
			Ошибка("  %+v \n", err)
			time.Sleep(задержка)
		} else {
			break
		}
	}
	go ЧитатьСообщения(сервер)
	Рукопожатие(сервер)
}

func ЧитатьСообщения(сервер *tls.Conn) {
	for {
		сообщение := make([]byte, 1024)
		_, err := сервер.Read(сообщение)
		Инфо("сообщение  %+s \n", сообщение)
		if err != nil {
			Ошибка(" %s", err)
		}
		
	}
}

func ПодключитьсяКСерверуДляОтправкиСообщений(каналЗапросов chan Запрос) {
	caCert, err := os.ReadFile("cert/ca.crt")

	if err != nil {
		Ошибка(" %s ", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	Инфо("Корневой сертфикат создан?  %v ", ok)

	cert, err := tls.LoadX509KeyPair("cert/client.crt", "cert/client.key")
	if err != nil {
		Ошибка(" %s", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	// Подключение к TCP-серверу с TLS на localhost:8080
	количествоПопыток := 500
	задержка := 1 * time.Second
	var сервер *tls.Conn
	var errDial error
	for попытка := 1; попытка <= количествоПопыток; попытка++ {
		сервер, errDial = tls.Dial("tcp", "localhost:81", tlsConfig)
		if errDial != nil {
			Ошибка("  %+v \n", err)
			time.Sleep(задержка)
		} else {
			break
		}
	}

	//По идее рукопожатие нужно при регистрации на серевере котоый отправляет сообщения, поэтому при регаистрации на сервере которому будем мы слать сообщение рукопожатие не нужно.
	// Рукопожатие(сервер)
	// defer сервер.Close()
	// каналЗапросов - исползуется для получения запросов от клиента, в запросе от клиента передаётся канал в который нужно отправить ответ клиенту

	go ОтправитьЗапросВОбработку(сервер, каналЗапросов)
	// входящий потому что на стороне менеджера сообщений это соединение будет для входяих запросов
	baseURL := "http://example.com/catalo/lost"
	params := url.Values{}
	params.Add("value", "1")
	params.Add("value2", "2")

	u, _ := url.ParseRequestURI(baseURL)
	u.RawQuery = params.Encode()
	Инфо("  %+v %+v \n", u, params)
	каналЗапросов <- Запрос{
		Сервис: []byte("КлиентСервер"),
		Запрос: ЗапросОтКлиента{
			Строка: u.String(),
			Форма:  nil,
			Файл:   "",
		},
		ИдКлиента: Уид(),
	}
}

func Рукопожатие(сервер *tls.Conn) {
	// буфер := new(bytes.Buffer)
	// Запрос{
	// 	Сервис:    []byte("КлиентСервер"),
	// 	Запрос:    "🤝",
	// 	ИдКлиента: Уид(),
	// }

	// Инфо("  %+v %+v \n", "🤝", []byte("🤝"), len([]byte("🤝")))
	// binary.Write(буфер, binary.LittleEndian, [4]byte{240, 159, 164, 157}) // [4]byte{240, 159, 164, 157} = "🤝"

	// Будет описаывать какие данные в каком виде нужно присылать в запросах для конкретного маршрута для данного сервиса
	//например сервис КлиентСервер , имеет обработчик ОтветКЛиенту : ДляЭтого метода ему нужен ИдКлиента, и ответ в виде HTML строки или json
	type СтруктураДанных struct {
		ОбъектДанных interface{}
	}
	type Отпечаток struct {
		Сервис   string
		Маршруты map[string]map[string]interface{}
	}

	КлиентСервер := Отпечаток{
		Сервис: "КлиентСервер",
		Маршруты: map[string]map[string]interface{}{
			"ОтветКлиенту": {
				"HTML": "string",
				"JSON": "string",
			},
			"catalog": {
				"HTML": "string",
				"JSON": "string",
			},
		},
	}
	// КлиентСервер := Отпечаток{
	// 	Сервис: "КаталогСервис",
	// 	Маршруты: map[string]map[string]interface{}{
	// 		"catalog": map[string]interface{}{
	// 			"Запрос": "string",
	// 		}
	//
	// 	},
	// }

	данныеВОтправку, err := Кодировать(КлиентСервер)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	// binary.Write(буфер, binary.LittleEndian, int32(len([]byte("КлиентСервер"))))
	// binary.Write(буфер, binary.LittleEndian, []byte("КлиентСервер"))
	сервер.Write(данныеВОтправку)

}

type ЗапросВОбработку struct {
	Сервис    []byte
	ИдКлиента uuid.UUID
	Запрос    ЗапросОтКлиента
}

func ПингПонг(сервер *tls.Conn) {
	for {
		err := сервер.Handshake()
		if err != nil {
			Инфо("Соединение разорвано!  %+v", err)
		} else {
			Инфо("Соединение установлено успешно! %+v", err)
			i, err := сервер.Write([]byte("ping"))
			if err != nil {
				Ошибка(" i %+v err %+v\n", i, err)
				сервер.Close()

				break
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func ОтправитьЗапросВОбработку(сервер *tls.Conn, каналЗапросов chan Запрос) {
	for ЗапросОтКлиента := range каналЗапросов {
		// Отправка сообщений серверу
		Инфо(" ЗапросОтКлиента %+v \n", ЗапросОтКлиента)
		мьютекс.Lock()
		клиенты[ЗапросОтКлиента.ИдКлиента] = ЗапросОтКлиента
		мьютекс.Unlock()

		// буфер := new(bytes.Buffer)
		// запросВОбработку := ЗапросВОбработку{
		// 	Сервис:    ЗапросОтКлиента.Сервис,
		// 	ИдКлиента: ЗапросОтКлиента.ИдКлиента,
		// 	Запрос:    ЗапросОтКлиента.Запрос,
		// }
		// Инфо(" ЗапросВОбработку %+v \n", ЗапросВОбработку)

		БинарныйЗапрос, err := Кодировать(ЗапросВОбработку{
			Сервис:    ЗапросОтКлиента.Сервис,
			ИдКлиента: ЗапросОтКлиента.ИдКлиента,
			Запрос:    ЗапросОтКлиента.Запрос,
		})

		if err != nil {
			Ошибка("  %+v \n", err)
		}
		Инфо(" БинарныйЗапрос %+s \n", БинарныйЗапрос)
		// err = binary.Write(буфер, binary.LittleEndian, БинарныйЗапрос)
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }

		int, err := сервер.Write(БинарныйЗапрос)
		if err != nil {
			Ошибка("  %+v %+v \n", int, err)
		}
		Инфо(" отправленно  %+v \n", int)

	}
}

// func (з ЗапросВОбработку) Кодировать(T any) ([]byte, error) {
func Кодировать(данныеДляКодирования interface{}) ([]byte, error) {

	b, err := jsoniter.Marshal(&данныеДляКодирования)
	if err != nil {
		Ошибка("  %+v \n", err)
		return nil, err
	}
	данные := make([]byte, len(b)+4)
	binary.LittleEndian.PutUint32(данные, uint32(len(b)))
	copy(данные[4:], b)
	return данные, nil

}

// func (з ЗапросВОбработку) КодироватьВБинарныйФормат() ([]byte, error) {
// 	// ∴ ⊶ ⁝  ⁖
// 	// ⁝ - конец сообщения.
// 	// Сообщение должно начинатся с размера

// 	// Инфо(" размер  %+v %+v \n", "∴",  len("∴"))
// 	// Инфо(" размер  %+v %+v \n", "⊶",  len("⊶"))
// 	// Инфо(" размер  %+v %+v \n", "⁝",  len("⁝"))

// 	// Создаем буфер нужного размера для сериализации

// 	буфер := new(bytes.Buffer)

// 	binary.Write(буфер, binary.LittleEndian, int32(6))
// 	binary.Write(буфер, binary.LittleEndian, [6]byte{208, 184, 208, 180, 208, 186})

// 	binary.Write(буфер, binary.LittleEndian, int32(len(з.ИдКлиента)))
// 	binary.Write(буфер, binary.LittleEndian, з.ИдКлиента)

// 	binary.Write(буфер, binary.LittleEndian, int32(len(з.Запрос)))
// 	binary.Write(буфер, binary.LittleEndian, з.Запрос)

// 	binary.Write(буфер, binary.LittleEndian, [4]byte{226, 129, 157, 0}) // ⁝ - записываем разделитель между сообщениями на всякий случай

// 	Инфо("бинарныеДанные  %+s ;Bytes %+v \n", буфер, int32(буфер.Len()))

// 	буферВОтправку := new(bytes.Buffer)
// 	binary.Write(буферВОтправку, binary.LittleEndian, int32(буфер.Len()))
// 	binary.Write(буферВОтправку, binary.LittleEndian, буфер.Bytes())
// 	// буферВОтправку.Write(буфер.Bytes())
// 	// Возвращаем сериализованные бинарные данные и ошибку (если есть)
// 	return буферВОтправку.Bytes(), nil
// }

func ОтправитьОтветКлиенту(сервер *tls.Conn, каналЗапросов chan Запрос) {

	for {
		var ОтветКлиенту ОтветКлиенту
		длина := make([]byte, 4)
		n, err := io.ReadFull(сервер, длина)
		Инфо("  %+v \n", n)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		lenData := binary.LittleEndian.Uint32(длина)

		буфер := make([]byte, lenData)
		i, err := io.ReadFull(сервер, буфер)
		Инфо("  %+v \n", i)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		err = binary.Read(bytes.NewReader(буфер), binary.LittleEndian, &ОтветКлиенту)
		if err != nil {
			Ошибка("Ошибка при десериализации структуры: %+v ", err)
		}

		клиенты[ОтветКлиенту.ИдКлиента].КаналОтвета <- ОтветКлиенту

	}
}

func ДеКодироватьОтветКлиенту(бинарныеДанные []byte) (*ОтветКлиенту, error) {
	буфер := bytes.NewReader(бинарныеДанные)
	var длинаИдКлиента int32
	if err := binary.Read(буфер, binary.LittleEndian, &длинаИдКлиента); err != nil {
		Ошибка("  %+v \n", err)
	}
	идКлиентаBytes := make([]byte, длинаИдКлиента)
	if err := binary.Read(буфер, binary.LittleEndian, &идКлиентаBytes); err != nil {
		return nil, fmt.Errorf("ошибка чтения ИдКлиента: %v", err)
	}
	идКлиента := идКлиентаBytes

	var значениеBytes []byte
	if err := binary.Read(буфер, binary.LittleEndian, &значениеBytes); err != nil {
		return nil, fmt.Errorf("ошибка чтения значения типа string: %v", err)
	}
	ответ := string(значениеBytes)
	ответКлиенту := &ОтветКлиенту{
		ИдКлиента: uuid.UUID(идКлиента),
		Ответ:     ответ,
	}

	return ответКлиенту, nil
}
