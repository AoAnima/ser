package main

import (
	"encoding/binary"
	"net/http"
	"sync"

	connector "aoanima.ru/connector"
	. "aoanima.ru/logger"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

var клиенты = make(map[[16]byte]map[connector.Уид]Запрос)
var мьютекс = sync.Mutex{}

// type ЗапросВОбработку struct {
// 	УИДЗапроса string
// 	Сервис     []byte
// 	ИдКлиента  uuid.UUID
// 	УрлПуть    []byte
// 	Запрос     ЗапросОтКлиента
// }

// func ПолучитьУидКЛиента(req *http.Request) uuid.UUID {
// 	var ИД uuid.UUID

// 	if cookieUuid, err := req.Cookie("uuid"); err != nil {
// 		Ошибка(" %+v \n", err)
// 		ИД = Уид()
// 	} else {
// 		ИД, err = uuid.Parse(cookieUuid.Value)
// 		if err != nil {
// 			Ошибка("  %+v \n", err)
// 		}
// 	}
// 	return ИД
// }

func ОтправитьЗапросВОбработку(каналОтправкиСообщений chan []byte, каналЗапросов chan Запрос) {
	for ЗапросОтКлиента := range каналЗапросов {
		// Отправка сообщений серверу
		Инфо(" ЗапросОтКлиента %+v \n", ЗапросОтКлиента)
		req := ЗапросОтКлиента.Req

		данныеЗапроса := connector.Запрос{
			ТипОтвета:      0,
			ТипЗапроса:     0,
			СтрокаЗапроса:  req.URL,
			МаршрутЗапроса: req.URL.Path,
			Форма:          nil,
			Файл:           "",
			УИДЗапроса:     "",
		}

		Сообщение := connector.Сообщение{
			Сервис:       "КлиентСервер",
			Запрос:       данныеЗапроса,
			ТокенКлиента: []byte(""),
			ИдКлиента:    ПолучитьУидКЛиента(&req),
		}

		типДанных := req.Header.Get("Content-Type")
		if req.Method == http.MethodPost {
			if типДанных == "multipart/form-data" {
				Инфо("нужно реализовать декодирование, я так понимаю тут передаются файлы через форму %+v \n", "multipart/form-data")
			}
			req.ParseForm()
			данныеЗапроса.Форма = req.Form
		}
		// if req.Method == "AJAX" || req.Method == "AJAXPost" {
		// 	if типДанных == "application/json" {
		// 		Запрос.ТипЗапроса = connector.AJAX
		// 	}
		// }
		switch req.Method {
		case "AJAX":
			данныеЗапроса.ТипЗапроса = connector.AJAX
			данныеЗапроса.ТипОтвета = connector.AjaxHTML

			break
		case "AJAXPost":
			данныеЗапроса.ТипЗапроса = connector.AJAXPost
			данныеЗапроса.ТипОтвета = connector.AjaxHTML
			break
		case http.MethodGet:
			данныеЗапроса.ТипЗапроса = connector.GET
			данныеЗапроса.ТипОтвета = connector.HTML
			break
		case http.MethodPost:
			данныеЗапроса.ТипЗапроса = connector.POST
			данныеЗапроса.ТипОтвета = connector.HTML
			break
		}

		//каналЗапросов читается в функции ОтправитьЗапросВОбработку, которая отправляет данные в synqTCP поэтому если нужно обраьботкть запро сперед отправкой, то его можно либо обрабатывать тут, перед отправкой в каналЗапросов, лобо внутри фкнции ОтправитьЗапросВОбработку перед записью данный в соединение с synqTCp
		// ИдКлиента := ПолучитьУидКЛиента(&req)
		// УИДЗапроса := fmt.Sprintf("%+s.%+s.%+s", time.Now().Unix(), ИдКлиента, metro.Hash64([]byte(req.URL.Path), 0))
		данныеЗапроса.УИДЗапроса = connector.УИДЗапроса(&Сообщение.ИдКлиента, []byte(req.URL.Path))
		Сообщение.УИДСообщения = данныеЗапроса.УИДЗапроса
		мьютекс.Lock()
		// дополним структуру запроса для карты клиенты
		ЗапросОтКлиента.ИдКлиента = Сообщение.ИдКлиента
		ЗапросОтКлиента.УИДЗапроса = Сообщение.УИДСообщения

		// ЗапросОтКлиента созраням в карте клиенты, сокращённый вариант сообщения который отправляется через коннектор в synqTCP
		if _, ok := клиенты[Сообщение.ИдКлиента]; !ok {
			клиенты[Сообщение.ИдКлиента] = map[connector.Уид]Запрос{}
		} else {
			клиенты[Сообщение.ИдКлиента][данныеЗапроса.УИДЗапроса] = ЗапросОтКлиента
		}
		клиенты[Сообщение.ИдКлиента][данныеЗапроса.УИДЗапроса] = ЗапросОтКлиента
		// клиенты[ЗапросОтКлиента.ИдКлиента][ЗапросОтКлиента.УИДЗапроса] = ЗапросОтКлиента
		мьютекс.Unlock()
		Инфо("ОтправитьЗапросВОбработку  клиенты %+v \n", клиенты)

		// Отправляем данные a synqTCP
		каналОтправкиСообщений <- Кодировать(Сообщение)

	}
}

// func (з ЗапросВОбработку) Кодировать(T any) ([]byte, error) {
// func Кодировать(данныеДляКодирования interface{}) []byte {

// 	b, err := jsoniter.Marshal(&данныеДляКодирования)
// 	if err != nil {
// 		Ошибка("  %+v \n", err)
// 		return nil
// 	}
// 	данные := make([]byte, len(b)+4)
// 	binary.LittleEndian.PutUint32(данные, uint32(len(b)))
// 	copy(данные[4:], b)

// 	return данные

// }

func ОтправитьОтветКлиенту(каналПолученияСообщений chan []byte) {
	for пакетОтвета := range каналПолученияСообщений {

		// Нужно будет проверить ответ, что пришло, в каком формате, соответсвует ли ответу, и затем отправлять клинету

		// Пока просто декодируем, получаем ИдКлиента и отправляем всё что пришло
		// var ОтветКлиентуКарта map[string]interface{}

		var СообщениеДляОтвета connector.Сообщение

		err := jsoniter.Unmarshal(пакетОтвета, &СообщениеДляОтвета)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		Инфо(" ОтветКлиентуКарта %+s \n", СообщениеДляОтвета)
		Инфо(" ОтветКлиентуКарта ИдКлиента %+s \n", СообщениеДляОтвета.ИдКлиента)

		// ИдКлиента := [16]byte{}
		// copy(ИдКлиента[:], ОтветКлиентуКарта["ИдКлиента"].(string))
		// ИдКлиента, err := uuid.Parse(СообщениеДляОтвета.ИдКлиента)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		// УИДЗапроса := СообщениеДляОтвета.УИДСообщения
		// УИДЗапроса := СообщениеДляОтвета.Ответ.УИДЗапроса
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }
		// Инфо(" ИдКлиента %+v; УИДЗапроса %+v \n", УИДЗапроса)

		if клиент, есть := клиенты[СообщениеДляОтвета.ИдКлиента]; есть {
			if Запрос, естьУидЗапроса := клиент[СообщениеДляОтвета.УИДСообщения]; естьУидЗапроса {

				Инфо(" Отправляем ответ клиенту %+v  %+v  %+v \n", СообщениеДляОтвета.ИдКлиента, клиент, Запрос)

				Ответ := СообщениеДляОтвета.Ответ
				// Кодировать(Ответ["Рендер"].Данные)
				Запрос.КаналОтвета <- ОтветКлиенту{
					УИДЗапроса: string(Запрос.УИДЗапроса),
					ИдКлиента:  Запрос.ИдКлиента,
					Ответ:      string(Ответ["Рендер"].Данные),
				}
				// ответ на запрос отправлен, удалим его из карты
				delete(клиент, СообщениеДляОтвета.Запрос.УИДЗапроса)
			}
		} else {
			Инфо(" Клиент с СообщениеДляОтвета %+v не найден %+v \n", СообщениеДляОтвета, клиенты)
		}
	}
}

// func ДеКодироватьОтветКлиенту(бинарныеДанные []byte) (*ОтветКлиенту, error) {
// 	буфер := bytes.NewReader(бинарныеДанные)
// 	var длинаИдКлиента int32
// 	if err := binary.Read(буфер, binary.LittleEndian, &длинаИдКлиента); err != nil {
// 		Ошибка("  %+v \n", err)
// 	}
// 	идКлиентаBytes := make([]byte, длинаИдКлиента)
// 	if err := binary.Read(буфер, binary.LittleEndian, &идКлиентаBytes); err != nil {
// 		return nil, fmt.Errorf("ошибка чтения ИдКлиента: %v", err)
// 	}
// 	идКлиента := идКлиентаBytes

// 	var значениеBytes []byte
// 	if err := binary.Read(буфер, binary.LittleEndian, &значениеBytes); err != nil {
// 		return nil, fmt.Errorf("ошибка чтения значения типа string: %v", err)
// 	}
// 	ответ := string(значениеBytes)
// 	ответКлиенту := &ОтветКлиенту{
// 		ИдКлиента: uuid.UUID(идКлиента),
// 		Ответ:     ответ,
// 	}

// 	return ответКлиенту, nil
// }

// func ПингПонг(сервер *tls.Conn) {
// 	for {
// 		err := сервер.Handshake()
// 		if err != nil {
// 			Инфо("Соединение разорвано!  %+v", err)
// 		} else {
// 			Инфо("Соединение установлено успешно! %+v", err)
// 			i, err := сервер.Write([]byte("ping"))
// 			if err != nil {
// 				Ошибка(" i %+v err %+v\n", i, err)
// 				сервер.Close()

// 				break
// 			}
// 		}
// 		time.Sleep(5 * time.Second)
// 	}
// }

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
