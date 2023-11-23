package main

import (
	"net/http"
	"sync"
	"time"

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/Logger"
	"github.com/google/uuid"
)

var БлокКлиентов = sync.RWMutex{}
var клиенты = make(map[uuid.UUID]КартаКлиентов)

type КартаКлиентов struct {
	Запросы             map[Уид]Запрос
	ПоследняяАктивность time.Time
}

// type ЗапросВОбработку struct {
// 	УИДЗапроса string
// 	Сервис     []byte
// 	ИдКлиента  uuid.UUID
// 	УрлПуть    []byte
// 	Запрос     ЗапросОтКлиента
// }

func ПолучитьУидКЛиента(req *http.Request) uuid.UUID {
	var ИД uuid.UUID

	if cookieUuid, err := req.Cookie("uuid"); err != nil {
		Ошибка("в куках нет uuid или jwt %+v \n", err)
		ИД = uuid.New()
	} else {
		ИД, err = uuid.Parse(cookieUuid.Value)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
	}

	if cookieJWT, err := req.Cookie("jwt"); err != nil {
		Ошибка("в куках нет jwt %+v \n", err)
		ИД = uuid.New()
	} else {
		Инфо(" реализовать прасинг JWT %+v \n", cookieJWT)
	}

	return ИД
}

/*
ОтправитьЗапросВОбработку формирует Сообщение
Вызывает функцию для получения потока из очереди, и отправляет сообщение в поток, ждёт ответа и пишет ответ в исходящий поток
*/
func ОтправитьЗапросВОбработку(ЗапросОтКлиента *http.Request) (Сообщение, error) {
	// for ЗапросОтКлиента := range каналЗапросов {
	// Отправка сообщений серверу
	Инфо(" ЗапросОтКлиента %+v \n", ЗапросОтКлиента)
	ИдКлиента := ПолучитьУидКЛиента(ЗапросОтКлиента)
	// запрос.УИДЗапроса = УИДЗапроса(&ИдКлиента, []byte(ЗапросОтКлиента.URL.Path))
	запрос := Запрос{
		ТипОтвета:      0,
		ТипЗапроса:     0,
		СтрокаЗапроса:  ЗапросОтКлиента.URL,
		МаршрутЗапроса: ЗапросОтКлиента.URL.Path,
		Форма:          nil,
		Файл:           "",
		УИДЗапроса:     УИДЗапроса(&ИдКлиента, []byte(ЗапросОтКлиента.URL.Path)),
	}

	типДанных := ЗапросОтКлиента.Header.Get("Content-Type")
	if ЗапросОтКлиента.Method == http.MethodPost {
		if типДанных == "multipart/form-data" {
			Инфо("нужно реализовать декодирование, я так понимаю тут передаются файлы через форму %+v \n", "multipart/form-data")
		}
		ЗапросОтКлиента.ParseForm()
		запрос.Форма = ЗапросОтКлиента.Form
	}
	// if ЗапросОтКлиента.Method == "AJAX" || ЗапросОтКлиента.Method == "AJAXPost" {
	// 	if типДанных == "application/json" {
	// 		Запрос.ТипЗапроса = connector.AJAX
	// 	}
	// }
	switch ЗапросОтКлиента.Method {
	case "AJAX":
		запрос.ТипЗапроса = AJAX
		запрос.ТипОтвета = AjaxHTML

	case "AJAXPost":
		запрос.ТипЗапроса = AJAXPost
		запрос.ТипОтвета = AjaxHTML

	case http.MethodGet:
		запрос.ТипЗапроса = GET
		запрос.ТипОтвета = HTML

	case http.MethodPost:
		запрос.ТипЗапроса = POST
		запрос.ТипОтвета = HTML

	}

	//каналЗапросов читается в функции ОтправитьЗапросВОбработку, которая отправляет данные в synqTCP поэтому если нужно обраьботкть запро сперед отправкой, то его можно либо обрабатывать тут, перед отправкой в каналЗапросов, лобо внутри фкнции ОтправитьЗапросВОбработку перед записью данный в соединение с synqTCp
	// ИдКлиента := ПолучитьУидКЛиента(&ЗапросОтКлиента)
	// УИДЗапроса := fmt.Sprintf("%+s.%+s.%+s", time.Now().Unix(), ИдКлиента, metro.Hash64([]byte(ЗапросОтКлиента.URL.Path), 0))

	сообщение := Сообщение{
		Сервис:       "КлиентСервер",
		Запрос:       запрос,
		ТокенКлиента: []byte(""),
		ИдКлиента:    ИдКлиента,
		УИДСообщения: запрос.УИДЗапроса,
	}

	БлокКлиентов.Lock()
	// ЗапросОтКлиента созраням в карте клиенты, сокращённый вариант сообщения который отправляется через коннектор в synqTCP
	if _, нетКлиента := клиенты[сообщение.ИдКлиента]; !нетКлиента {
		// клиенты[сообщение.ИдКлиента].Запросы =make(map[Уид]Запрос)
		клиенты[сообщение.ИдКлиента] = КартаКлиентов{
			Запросы: map[Уид]Запрос{
				сообщение.УИДСообщения: запрос,
			},
			ПоследняяАктивность: time.Now(),
		}
	} else {
		СуществующийКлиент := клиенты[сообщение.ИдКлиента]
		СуществующийКлиент.Запросы[сообщение.УИДСообщения] = запрос
		СуществующийКлиент.ПоследняяАктивность = time.Now()

		клиенты[сообщение.ИдКлиента] = СуществующийКлиент
	}
	БлокКлиентов.Unlock()

	Инфо("ОтправитьЗапросВОбработку  клиенты %+v \n", клиенты[сообщение.ИдКлиента])

	потокСессии, err := ПолучитьSynQuicПотокДляОтправки()
	if err != nil {
		Ошибка("  %+v \n", err)
		return Сообщение{}, err
	}
	// Отправялем сообщение в SynQuic
	Инфо(" Отправляем сообщение в SynQuic  %+v  потокСессии  %+v \n", сообщение, потокСессии)
	ОтправитьСообщение(потокСессии.Поток, сообщение)
	// Получаем ответ от SynQuic
	ответ := ЧитатьСообщение(потокСессии.Поток)
	// возвращаем поток в очередь
	Инфо(" ответ SynQuic  %+v \n, возвращаем поток SynQuic в очередь", ответ)
	go ВернутьSynQuicПотокВочередь(потокСессии)
	go УдалитьЗапросКлиента(ответ.ИдКлиента, ответ.Запрос.УИДЗапроса)
	// возвращаем ответ для отправки клиенту
	return ответ, nil

	// }
}

func УдалитьЗапросКлиента(ИдКлиента uuid.UUID, УИДЗапроса Уид) {
	if _, нетКлиента := клиенты[ИдКлиента]; !нетКлиента {
		Инфо(" %+v нет в карте клиенты %+v \n", ИдКлиента, клиенты)
		return
	}

	if _, есть := клиенты[ИдКлиента].Запросы[УИДЗапроса]; есть {
		БлокКлиентов.RLock()
		delete(клиенты[ИдКлиента].Запросы, УИДЗапроса)
		БлокКлиентов.RUnlock()

	} else {
		Инфо(" нет Уид Запроса в карте запросов %+v УИДЗапроса %+v \n", клиенты[ИдКлиента].Запросы, УИДЗапроса)
	}

	Инфо(" клиенты %+v %+v \n", клиенты, len(клиенты))
	go ОчисткаКартыКЛиентов()
}

func ОчисткаКартыКЛиентов() {
	лимитАктивностиКЛинета := 1
	Инфо("ОчисткаКартыКЛиентов проверяет как давно была последняя активность клиента, если более v минут назад то удалёт клинета из карты\n", лимитАктивностиКЛинета)

	for ИдКлиента, КартаЗапросов := range клиенты {
		if len(КартаЗапросов.Запросы) == 0 {
			if time.Since(КартаЗапросов.ПоследняяАктивность) > time.Duration(лимитАктивностиКЛинета)*time.Minute {
				БлокКлиентов.Lock()
				delete(клиенты, ИдКлиента)
				БлокКлиентов.Unlock()
			}
		}
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

// func ОтправитьОтветКлиенту(каналПолученияСообщений chan []byte) {
// 	for пакетОтвета := range каналПолученияСообщений {

// 		// Нужно будет проверить ответ, что пришло, в каком формате, соответсвует ли ответу, и затем отправлять клинету

// 		// Пока просто декодируем, получаем ИдКлиента и отправляем всё что пришло
// 		// var ОтветКлиентуКарта map[string]interface{}

// 		var СообщениеДляОтвета connector.Сообщение

// 		err := jsoniter.Unmarshal(пакетОтвета, &СообщениеДляОтвета)
// 		if err != nil {
// 			Ошибка("  %+v \n", err)
// 		}
// 		Инфо(" ОтветКлиентуКарта %+s \n", СообщениеДляОтвета)
// 		Инфо(" ОтветКлиентуКарта ИдКлиента %+s \n", СообщениеДляОтвета.ИдКлиента)

// 		// ИдКлиента := [16]byte{}
// 		// copy(ИдКлиента[:], ОтветКлиентуКарта["ИдКлиента"].(string))
// 		// ИдКлиента, err := uuid.Parse(СообщениеДляОтвета.ИдКлиента)
// 		if err != nil {
// 			Ошибка("  %+v \n", err)
// 		}
// 		// УИДЗапроса := СообщениеДляОтвета.УИДСообщения
// 		// УИДЗапроса := СообщениеДляОтвета.Ответ.УИДЗапроса
// 		// if err != nil {
// 		// 	Ошибка("  %+v \n", err)
// 		// }
// 		// Инфо(" ИдКлиента %+v; УИДЗапроса %+v \n", УИДЗапроса)

// 		// if клиент, есть := клиенты[СообщениеДляОтвета.ИдКлиента]; есть {
// 		// 	if Запрос, естьУидЗапроса := клиент[СообщениеДляОтвета.УИДСообщения]; естьУидЗапроса {

// 		// 		Инфо(" Отправляем ответ клиенту %+v  %+v  %+v \n", СообщениеДляОтвета.ИдКлиента, клиент, Запрос)

// 		// 		Ответ := СообщениеДляОтвета.Ответ
// 		// 		// Кодировать(Ответ["Рендер"].Данные)
// 		// 		Запрос.КаналОтвета <- ОтветКлиенту{
// 		// 			УИДЗапроса: string(Запрос.УИДЗапроса),
// 		// 			ИдКлиента:  Запрос.ИдКлиента,
// 		// 			Ответ:      string(Ответ["Рендер"].Данные),
// 		// 		}
// 		// 		// ответ на запрос отправлен, удалим его из карты
// 		// 		delete(клиент, СообщениеДляОтвета.Запрос.УИДЗапроса)
// 		// 	}
// 		// } else {
// 		// 	Инфо(" Клиент с СообщениеДляОтвета %+v не найден %+v \n", СообщениеДляОтвета, клиенты)
// 		// }
// 	}
// }

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
