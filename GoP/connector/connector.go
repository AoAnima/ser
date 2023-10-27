package connector

import (
	"fmt"
	_ "net/http/pprof"
	"net/url"
	"time"

	. "aoanima.ru/logger"
	"github.com/dgryski/go-metro"
	"github.com/google/uuid"
)

// type Глюк struct {
// 	Текст     string
// 	КодОшибки int
// }

// func (e *Глюк) Error() string {
// 	return e.Текст
// }
// func (e *Глюк) Код() int {
// 	return e.КодОшибки
// }

type ТипОтвета int

const (
	AjaxHTML ТипОтвета = iota
	AjaxJSON
	HTML
)

type ТипЗапроса int

const (
	GET ТипЗапроса = iota
	POST
	AJAX
	AJAXPost
)

type Отпечаток struct {
	Сервис   string
	Маршруты map[string]*СтруктураМаршрута
}
type СтруктураМаршрута struct {
	Запрос map[string]interface{} // описывает  данные которые нужны для обработки маршрута
	Ответ  map[string]interface{} // описывает формат в котором вернёт данные
}
type Сервис string
type ОтветСервиса struct {
	Сервис          []byte // Имя сервиса который отправляет ответ
	УИДЗапроса      string // Копируется из запроса
	Данные          []byte // Ответ в бинарном формате
	ЗапросОбработан bool   // Признак того что запросы был получен и обработан соответсвуюбщим сервисом, в не зависимоти есть ли данные в ответе или нет, если данных нет, знаичт они не нужны... Выставляем в true в сеорвисе перед отправкой ответа
}
type Ответ map[Сервис]ОтветСервиса

type Сообщение struct {
	Сервис       []byte // Имя Сервиса который шлёт Сообщение, каждый сервис пишет своё имя в не зависимости что это ответ или запрос
	Запрос       *Запрос
	Ответ        *Ответ
	ИдКлиента    uuid.UUID
	УИДСообщения УидЗапроса // ХЗ по логике каждый сервис должен вставлять сюбда своё УИД
	ТокенКлиента []byte     // JWT сериализованный
}
type УидЗапроса string

type Запрос struct {
	ТипОтвета      ТипОтвета
	ТипЗапроса     ТипЗапроса
	СтрокаЗапроса  *url.URL // url Path Query
	МаршрутЗапроса string   // url Path Query
	Форма          map[string][]string
	Файл           string
	УИДЗапроса     УидЗапроса
}

var (
	ПортДляОтправкиСообщений  = "81"
	ПортДляПолученияСообщений = "82"
)

func УИДЗапроса(ИдКлиента *uuid.UUID, UrlPath []byte) УидЗапроса {
	return УидЗапроса(fmt.Sprintf("%+s.%+s.%+s", time.Now().Unix(), ИдКлиента, metro.Hash64(UrlPath, 0)))
}

// ПортИсходящихСообщений, ПортВходящихСообщений указывается те порты которые были исопльзованы в synqTCP сервер
// ПортДляОтправкиСообщений - соответсвует ВходящийПорт(synqTCP) - в этот порт серввис отправлят сообщения в synqTCP
// ПортДляПолученияСообщений - соответсвует ИсходящийПорт(synqTCP) -  из этого порта сервысы получают соощения из synqTCP
func ИнициализацияСервиса(
	адрес string,
	ПортИсходящихСообщений string,
	ПортВходящихСообщений string,
	отпечатокСервиса Отпечаток) (chan []byte,
	chan []byte) {

	Инфо(" ИнициализацияСервисов %+v \n", отпечатокСервиса)

	каналПолученияСообщений := make(chan []byte, 10)
	каналОтправкиСообщений := make(chan []byte, 10)

	go ПодключитсяКСерверуДляПолученияСообщений(каналПолученияСообщений, адрес, ПортВходящихСообщений, отпечатокСервиса)
	go ПодключитьсяКСерверуДляОтправкиСообщений(каналОтправкиСообщений, адрес, ПортИсходящихСообщений)

	return каналПолученияСообщений, каналОтправкиСообщений
}

// Сервис := Отпечаток{
// 	Сервис: "Каталог",
// 	Маршруты: map[string]*СтруктураМаршрута{

// 	 "/": {
// 		Запрос: {
// 			"ТипЗпроса": "int", // в заивисмости от типа запроса например ajax или обычный request будет возвращён ответ...
// 			"Строка": "string", // url Path Query
// 			"Форма": "map[string][]string",
// 			"Файл":   "string",
// 		},
// 		Ответ:	{
// 			"HTML": "string",
// 			"JSON": "string",
// 			},
// 		} ,
// 		"catalog": {
// 		Запрос: {
// 			"ТипЗпроса": "int", // в заивисмости от типа запроса например ajax или обычный request будет возвращён ответ...
// 			"Строка": "string", // url Path Query
// 			"Форма": "map[string][]string",
// 			"Файл":   "string",
// 		},
// 		Ответ:	{
// 			"HTML": "string",
// 			"JSON": "string",
// 			},
// 		} ,
// 	},
// }
