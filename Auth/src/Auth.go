package main

import (
	"crypto/rand"
	"math"
	"math/big"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/DGApi"
	. "aoanima.ru/Logger"
	. "aoanima.ru/QErrors"
	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"github.com/quic-go/quic-go"
)

var клиент = make(Клиент)
var Сервис ИмяСервиса = "Авторизация"

// TODO

type ДанныеКлиента struct {
	Имя       string    `json:"имя,omitempty"`
	Фамилия   string    `json:"фамилия,omitempty"`
	Отчество  string    `json:"отчество,omitempty"`
	ИдКлиента uuid.UUID `json:"ид_клиента"`
	Роль      []string  `json:"роль,omitempty"`
	Права     []string  `json:"права_доступа,omitempty"`
	Статус    string    `json:"статус,omitempty"`
	Аватар    string    `json:"аватар,omitempty"`
	Email     string    `json:"email,omitempty"`
	Логин     string    `json:"логин,omitempty"`
	Пароль    string    `json:"пароль,omitempty"`
	JWT       string    `json:"jwt,omitempty"`
	Телефон   string    `json:"телефон,omitempty"`
	Адрес     Адрес     `json:"адрес,omitempty"`
	Создан    time.Time `json:"создан,omitempty"`
	Обновлен  time.Time `json:"обновлен,omitempty"`
	ОСебе     string    `json:"о_себе,omitempty"`
	СоцСети   []string  `json:"социальные_ссылки,omitempty"`
	Профиль   map[string]interface{}
}

type Адрес struct {
	Страна        string `json:"страна,omitempty"`
	Город         string `json:"город,omitempty"`
	Район         string `json:"район,omitempty"`
	ТипУлицы      string `json:"тип_улицы,omitempty"`
	НазваниеУлицы string `json:"название_улицы,omitempty"`
	НомерДома     string `json:"номер_дома,omitempty"`
	Корпус        string `json:"корпус,omitempty"`
	НомерКвартиры string `json:"номер_квартиры,omitempty"`
}
type Секрет struct {
	ИдКлиента string    `json:"ид_клиента"`
	Секрет    string    `json:"секрет"`
	Обновлен  time.Time `json:"обновлен"`
}

var База СоединениеСДГраф
var СекретноеСоединение СоединениеСДГраф

var ПраваДоступа = []string{"чтение", "просмотр", "изменение своего", "создание пользователей", "удаление пользователей", "изменение ролей"}

var РолиПользователей = []string{"гость", "администратор", "покупатель", "продавец", "управляющий"}

var СхемаБазы = `<ид_клиента>: string @index(exact) @upsert .
				<права_доступа>: [string] .
				<секрет> : string .
				<имя>: string  .
				<фамилия>: string  .
				<отчество>: string  .
				<логин>: string @index(exact) @upsert .
				<пароль>: password .
				email: string @index(exact) @upsert .
				<телефон>: string @index(exact)  @upsert .
				<роль>: string  .
				<создан>: datetime .
				<обновлен>: datetime  .
				<статус>: string  .
				<аватар>: string .
				<о_себе>: string .
				<социальные_ссылки>: [string] .							
				<адрес>: uid .
				<страна>: string  .
				<город>: string  .
				<район>: string  .
				<тип_улицы>: string  .
				<название_улицы>: string  .
				<номер_дома>: string  .
				<корпус>: string  .
				<номер_квартиры>: string  .
				jwt: string .
				type <Секрет> {	
								<ид_клиента> 
								<секрет> 
								<обновлен> 							
							}
							type <Adres> {
									<страна>
									<город>
									<район>
									<тип_улицы>
									<название_улицы>
									<номер_дома>
									<корпус>
									<номер_квартиры>
							}
							type <User> {
									<ид_клиента>
									<имя>
									<фамилия>
									<отчество>
									<логин>
									<пароль>
									email
									<телефон>
									<адрес>
									<права_доступа>
									<создан>
									<обновлен>
									<статус>
									<аватар>
									<о_себе>
									<социальные_ссылки>							
									jwt
							}							
						`

type Конфигурация struct{}

// var каталогСтатичныхФайлов string
var Конфиг = &Конфигурация{}

func init() {
	Инфо(" проверяем какие аргументы переданы при запуске, если пусто то читаем конфиг, если конфига нет то устанавливаем значения по умолчанию %+v \n")

	// каталогСтатичныхФайлов = "../../HTML/static/"
	ЧитатьКонфиг(Конфиг)
	// База = СоединениеСДГраф{}
	База = ДГраф()
	СекретноеСоединение = ДГраф()
	// ответ, статусхемы := База.Получить(ДанныеЗапроса{
	// 	Запрос: `schema {
	// 		type
	// 		index
	// 		}`,
	// 	Данные: nil,
	// })
	// Инфо(" ответ %+s %+v \n", ответ, статусхемы)

	статус := База.Схема(ДанныеЗапроса{
		Запрос: СхемаБазы,
	})

	if статус.Код != Ок {
		Ошибка(" Ошибка записи схемы  %+v \n", статус)
	}

	// ответЛогин, статусОтвета := ЛогинСвободен("anima")
	// Инфо(" %+v  %+v \n", ответЛогин, статусОтвета)
	// добавить := "{ set { _:Пользователь <логин> \"Michael\" . } }"

	// добавить := `[
	//     {
	//         "логин": "user5",
	//         "имя": "Алексей Алексеев",
	//         "email": "alexey@example.com",
	//         "dgraph.type": "Пользователи"
	//     },
	//     {
	//         "логин": "user6",
	//         "имя": "Наталья Натальева", ит
	//         "email": "natalya@example.com",
	//         "dgraph.type": "Пользователи"
	//     }
	// ]`
	// ответиз, статусИзменения := База.Изменить(ДанныеЗапроса{
	// 	Запрос: добавить,
	// })
	// Инфо(" ответ %+s %+v \n", ответиз, статусИзменения)

	// ответ, статусхемы := База.Получить(ДанныеЗапроса{
	// 	Запрос: `{
	// 		Pols(func: has(email)) {
	// 		  email
	// 		  name
	// 		  uid
	// 		  dgraph.type
	// 		  <логин>
	// 		}
	// 	  }`,
	// 	Данные: nil,
	// })

	ответ, статусхемы := База.Получить(ДанныеЗапроса{
		Запрос: `{
			checkLogin(func: eq(<логин>, "user6")) {
			  count(uid)
			}
			checkEmail(func: eq(email, "alexey@example.com")) {
			  count(uid)
			}
		  }`,
		Данные: nil,
	})
	// {
	//   me(func: eq(name@en, "Steven Spielberg")) @filter(has(director.film)) {
	//     name@en
	//     director.film @filter(allofterms(name@en, "jones indiana") OR allofterms(name@en, "jurassic park"))  {
	//       uid
	//       name@en
	//     }
	//   }
	// }

	Инфо("   %+s %+v \n", ответ, статусхемы)

	// ответ, статус := База.Получить(ДанныеЗапроса{
	// 	Запрос: `schema {
	// 		type
	// 		index
	// 		}`,
	// 	Данные: nil,
	// })
	// схема := map[string]interface{}{}
	// err := ИзJson(ответ, &схема)
	// if err != nil {
	// 	Ошибка(" ОписаниеОшибки %+v \n", err.Error())
	// }

	// Инфо(" ответ %+s схема %+v \n", ответ, схема)

	// if статус.Код != Ок {
	// 	Ошибка(" Не удалось получить схему данных  %+v ответ  %+v \n", статус)
	// }

}

func main() {
	Инфо("  %+v \n", " Запуск сервиса Авторизации")
	сервер := &СхемаСервера{
		Имя:   "SynQuic",
		Адрес: "localhost:4242",
		ДанныеСессии: ДанныеСессии{
			Блок:   &sync.RWMutex{},
			Потоки: []quic.Stream{},
		},
	}

	сообщениеРегистрации := Сообщение{
		Сервис:      Сервис,
		Регистрация: true,
		Маршруты: []Маршрут{"reg",
			"auth",
			"verify",
			"checkLogin",
			"code",
			"userAccess",
			"регистрация",
			"авторизация",
			"верификация",
			"идентификация",
			"праваДоступа",
			"проверитьЛогин",
			"проверитьКод",
			"проверитьEmail"},
	}

	клиент.Соединиться(сервер,
		сообщениеРегистрации,
		ОбработчикОтветаРегистрации,
		ОбработчикЗапросовСервера)
}

func ОбработчикОтветаРегистрации(сообщение Сообщение) {
	Инфо("  ОбработчикОтветаРегистрации %+v \n", сообщение)
}

func ОбработчикЗапросовСервера(поток quic.Stream, сообщение Сообщение) {
	Инфо("  ОбработчикЗапросовСервера %+v \n", сообщение)
	var err error
	var статусСервиса СтатусСервиса
	var ok bool

	параметрыЗапроса, err := url.Parse(сообщение.Запрос.МаршрутЗапроса)
	Инфо(" параметрыЗапроса %+v \n", параметрыЗапроса)
	if err != nil {
		Ошибка("Ошибка при парсинге СтрокаЗапроса запроса:", err)
	}

	параметрыЗапроса.Path = strings.Trim(параметрыЗапроса.Path, "/")
	дейсвтия := strings.Split(параметрыЗапроса.Path, "/")

	var Действие string
	if len(дейсвтия) == 0 {
		Инфо(" Пустой маршрут, добавляем в маршруты обработку по умолчанию: авторизация \n")
		// Читаем заголовки парсим и проверяем JWT
		Действие = "авторизация" //првоерим и валидируем токен, получим права доступа

	} else {
		Действие = дейсвтия[0]
	}

	switch Действие {
	case "reg", "регистрация":
		ok, статусСервиса = Регистрация(&сообщение)

		Инфо(" Регистрация ок  %+v статусСервиса %+v \n", ok, статусСервиса)

		отправить, err := Кодировать(сообщение)
		if err != nil {
			Ошибка("  %+v \n", err)
		}
		поток.Write(отправить)
		return
	case "auth", "аутентификация":
		// Проверяет логин и папроль и создаёт jwt тоен клиента.
		данныеКлиента, статус := Аутентификация(&сообщение) // Проверка пары логин и паролль

		// тут  у нас получается что сревис будет Авторизация, значит нужно добавить в сообщение имяШаблона который нужно рендерить!!!! если успешная авторизация то Показать шаблон подтверждения личности через отправку кода на смс или месенджер или почту,
		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ЗапросОбработан = true
		ответ.СтатусОтвета = статус

		if статус.Код == Ок {
			ответ.ИмяШаблона = "ВыборСпособа2ФАвторизации"

			ответ.Данные = map[string]interface{}{
				"ПользовательОпознан": true,
				"ДанныеКлиента":       данныеКлиента,
			}
		} else {
			ответ.ИмяШаблона = "Ошибка авторизации"

			ответ.Данные = map[string]bool{
				"ПользовательОпознан": false,
			}
		}
		// 	// не нужно из сервиса выполнять функцию которая должна быть вынесено в отдельный сервис отпраки email и  смс и сообщения в месенджер... Нужно  сделать отдельный сервис,
		// 	// если какому то сервису нужно отправить сообщение клиенту то он возвращает ответ с маршрутом который необходимо выполнить... нужно подумать о приоритете таких маршрутов перед теми которые были созданы в самом начале в SynQuic

		// 	ОтправитьПроверочныйКод(&сообщение, email)

		сообщение.Ответ[Сервис] = ответ

		return

	case "code", "проверитьКод":
		ok, err = ПроверитьКод(&сообщение)
		if err != nil {
			Ошибка(" ОписаниеОшибки %+v \n", err.Error())
		}

		Инфо(" ПроверитьКод ок  %+v \n", ok)
	case "checkLogin", "проверитьЛогин":
		_, статусСервиса = ПроверитьЛогин(&сообщение)
		if статусСервиса.Код != Ок {
			Ошибка(" ОписаниеОшибки %+v \n", статусСервиса)
		}
		// отправить, err := Кодировать(сообщение)
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }
		// поток.Write(отправить)
		ОтправитьСообщение(поток, сообщение)
		return
	case "checkEmail", "проверитьEmail":
		_, статусСервиса = ПроверитьEmail(&сообщение)
		ОтправитьСообщение(поток, сообщение)
		// отправить, err := Кодировать(сообщение)
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }
		// поток.Write(отправить)
		return
	case "verify", "верификация", "идентификация":
		// Проверяет jwt токен.
		_, статус := ВлаидацияТокена(&сообщение)

		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ЗапросОбработан = true

		if статус.Код == Ок {
			ответ.Данные = map[string]bool{
				"ТокенВерный": true,
			}
		} else {
			ответ.Данные = map[string]bool{
				"ТокенВерный": false,
			}
		}
		ответ.СтатусОтвета = статус
		сообщение.Ответ[Сервис] = ответ

		// отправить, err := Кодировать(сообщение)
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }
		// поток.Write(отправить)
		ОтправитьСообщение(поток, сообщение)
		// if ok {
		// 	отправить, err := Кодировать(сообщение)
		// 	if err != nil {
		// 		Ошибка("  %+v \n", err)
		// 	}
		// 	поток.Write(отправить)
		// } else {
		// 	Ошибка("  %+v \n", статус)
		// }
		return
	case "праваДоступа", "авторизация", "userAccess":
		// Авторизация, получение прав доступа пользователя....
		ПрошёлВалидацию, статус := ВлаидацияТокена(&сообщение)
		статус = Авторизация(ПрошёлВалидацию, &сообщение) // получем права доступа пользователя

		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ЗапросОбработан = true

		if статус.Код != Ок {
			ответ.Данные = map[string]bool{
				"ТокенВерный": true,
			}
		} else {
			ответ.Данные = map[string]bool{
				"ТокенВерный": false,
			}
		}
		ответ.СтатусОтвета = статус
		сообщение.Ответ[Сервис] = ответ

		// отправить, err := Кодировать(сообщение)
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }
		// поток.Write(отправить)
		ОтправитьСообщение(поток, сообщение)
		return
	default:
		// если не совпадает ни с одним из действий то проверим если токен не пустой то проверим подпись
		ПрошёлВалидацию, статус := ВлаидацияТокена(&сообщение)
		статус = Авторизация(ПрошёлВалидацию, &сообщение) // получем права доступа пользователя, записываем их в структуру  сообщение.ТокенКлиента.Права

		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ЗапросОбработан = true
		if статус.Код == Ок {
			ответ.Данные = map[string]bool{
				"ТокенВерный": true,
			}
		} else {
			ответ.Данные = map[string]bool{
				"ТокенВерный": false,
			}
		}
		ответ.СтатусОтвета = статус
		сообщение.Ответ[Сервис] = ответ

		// отправить, err := Кодировать(сообщение)
		// if err != nil {
		// 	Ошибка("  %+v \n", err)
		// }
		// поток.Write(отправить)
		ОтправитьСообщение(поток, сообщение)
		// if ok {
		// 	отправить, err := Кодировать(сообщение)
		// 	if err != nil {
		// 		Ошибка("  %+v \n", err)
		// 	}
		// 	поток.Write(отправить)
		// } else {
		// 	Ошибка("  %+v \n", статус)
		// }
		return
	}

}
func ПолуитьПраваДоступаИзБД(ИдКлиента string) ([]byte, СтатусСервиса) {
	ответ, статус := База.Получить(ДанныеЗапроса{
		Запрос: `query User(<$ид_клиента>: string)  {
			getUserAccess (func: eq(<ид_клиента>, <$ид_клиента>)) {
			  <права_доступа>
			}			
		  }`,
		Данные: map[string]string{
			"$ид_клиента": ИдКлиента,
		},
	})
	return ответ, СтатусСервиса{
		Код:   статус.Код,
		Текст: статус.Текст,
	}
}

// Авторизация получем права доступа пользователя, записываем их в структуру  сообщение.ТокенКлиента.Права
func Авторизация(ПрошёлВалидацию bool, сообщение *Сообщение) СтатусСервиса {

	if ПрошёлВалидацию {
		праваДоступа, статус := ПолуитьПраваДоступаИзБД(сообщение.ИдКлиента.String())
		if статус.Код == Ок {
			var права = []string{}
			ИзJson(праваДоступа, &права)
			сообщение.ТокенКлиента.Права = права
			return статус
		} else {
			сообщение.ТокенКлиента.Права = []string{"гость"}
			return статус
		}
	}
	// Если пользователь не прошёл валидацию то назначаем ему права Гостя
	сообщение.ТокенКлиента.Права = []string{"гость"}
	return СтатусСервиса{
		Код:   ПользовательНеОпознан,
		Текст: "Пользователь не опознан",
	}

}

func Аутентификация(сообщение *Сообщение) (ДанныеКлиента, СтатусСервиса) {
	форма := сообщение.Запрос.Форма
	if len(форма) > 0 {
		var логин, пароль []string

		if логин = форма["login"]; len(логин) > 0 && логин[0] == "" {
			return ДанныеКлиента{}, СтатусСервиса{
				Текст: "нет логина",
				Код:   ОшибкаАвторизации,
			}
		}
		if пароль = форма["password"]; len(пароль) > 0 && пароль[0] == "" {
			return ДанныеКлиента{}, СтатусСервиса{
				Текст: "нет пароля",
				Код:   ОшибкаАвторизации,
			}
		}
		// не известно чё хотел
		данныеКлиента, статус := ПроверитьДанныеАвторизацииВБД(сообщение.ИдКлиента.String(), логин[0], пароль[0])
		if статус.Код != Ок {
			return данныеКлиента, СтатусСервиса{
				Текст: статус.Текст,
				Код:   ОшибкаАвторизации,
			}
		}

		// сообщение.ТокенКлиента = данныеКлиента.
		JWT, ошибкаСозданияJWT := СоздатьJWT(данныеКлиента)
		if ошибкаСозданияJWT.Код != Ок {
			Ошибка(" не удалось создать токен  %+v \n", ошибкаСозданияJWT)
			return данныеКлиента, СтатусСервиса(ошибкаСозданияJWT)
		}
		данныеКлиента.JWT = JWT
		сообщение.JWT = JWT

		return данныеКлиента, СтатусСервиса{
			Текст: "авторизация прошла успешно",
			Код:   Ок,
		}
	} else {
		return ДанныеКлиента{}, СтатусСервиса{
			Текст: "нет данных для авторизации, логин и пароль обязательны",
			Код:   ОшибкаАвторизации,
		}
	}
}

func ОтправитьПроверочныйКод(сообщение *Сообщение, email string) {
	Инфо(" ОтправитьПроверочныйКод ? отправялем сообщение на почту или смс %+v \n")

	// в сообщение в ответ добавляем токен для сверки с кодом, который будет подставлен в форму подтверждения
	ответ := сообщение.Ответ[Сервис]
	токенВерификации := СоздатьСлучайныйТокен(16)
	ответ.Данные = map[string]string{
		"токенВерификации": токенВерификации,
	}
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true

	проверочныйКод := СгенерироватьПроверочныйКод(6)
	СохранитьКодАвторизации(email, сообщение.ИдКлиента.String(), проверочныйКод, токенВерификации)

	сообщение.Ответ[Сервис] = ответ

	go ОтправитьEmail([]string{email}, "Проверочный код", "Введите проверочный код на странице атворизации "+проверочныйКод)

}

func ПроверитьЛогин(сообщение *Сообщение) (bool, СтатусСервиса) {
	логин := сообщение.Запрос.Форма["Логин"][0]
	ok, ошибкаСервиса := ЛогинСвободен(логин)
	if ошибкаСервиса.Код != Ок {
		Ошибка("  %+v \n", ошибкаСервиса)
	}
	Инфо(" ЛогинСвободен ок  %+v \n", ошибкаСервиса)

	ответ := сообщение.Ответ[Сервис]
	ответ.СтатусОтвета = ошибкаСервиса
	ответ.Данные = map[string]string{
		"Статус": "Логин свободен",
	}
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true
	сообщение.Ответ[Сервис] = ответ
	return ok, ошибкаСервиса
}

func ЛогинСвободен(логин string) (bool, СтатусСервиса) {
	Инфо("ЛогинСвободен , нужно проверить логин на свободу \n")

	ответ, статус := База.Получить(ДанныеЗапроса{
		Запрос: `query {
			checkLogin(func: eq(<логин>, "ваш_логин")) {
			  count(uid)
			}			
		  }`,
		Данные: nil,
	})
	Инфо(" %+s %+v \n", ответ, статус)

	// данные, ошибка := БазаКлиентов.Найти("Логин", логин)

	// if ошибка.Код == ОшибкаКлючНеНайден {
	// 	Инфо("  %+v  %+v \n", данные, ошибка)
	// 	return true, ОшибкаСервиса{
	// 		Код:   Ок,
	// 		Текст: "Логин свободен",
	// 	}
	// } else if ошибка.Код == Ок {
	// 	Инфо("  %+v  %+v \n", данные, ошибка)
	// 	if len(данные) > 0 {
	// 		return false, ОшибкаСервиса{
	// 			Код:   Прочее,
	// 			Текст: "Логин занят",
	// 		}
	// 	}
	// 	return false, ОшибкаСервиса{
	// 		Код:   Прочее,
	// 		Текст: "Логин занят",
	// 	}
	// }
	// Инфо("  %+v  %+v \n", данные, ошибка)
	return true, СтатусСервиса{
		Код:   Ок,
		Текст: "Логин свободен",
	}
}

func ПроверитьEmail(сообщение *Сообщение) (bool, СтатусСервиса) {
	email := сообщение.Запрос.Форма["Email"][0]
	ok, ошибкаСервиса := EmailСвободен(email)
	if ошибкаСервиса.Код != Ок {
		Ошибка("  %+v \n", ошибкаСервиса)
	}
	ответ := сообщение.Ответ[Сервис]
	ответ.СтатусОтвета = ошибкаСервиса
	ответ.Данные = map[string]string{
		"Статус": "Email свободен",
	}
	сообщение.Ответ[Сервис] = ответ
	Инфо(" EmailСвободен ок  %+v \n", ошибкаСервиса)
	return ok, ошибкаСервиса
}

func EmailСвободен(email string) (bool, СтатусСервиса) {

	// данные, ошибка := БазаКлиентов.Найти("Email", email)
	// if ошибка.Код == ОшибкаКлючНеНайден {
	// 	Инфо("  %+v  %+v \n", данные, ошибка)
	// 	return true, ОшибкаСервиса{
	// 		Код:   Ок,
	// 		Текст: "Email свободен",
	// 	}
	// } else if ошибка.Код == Ок {
	// 	Инфо("  %+v  %+v \n", данные, ошибка)
	// 	if len(данные) > 0 {
	// 		return false, ОшибкаСервиса{
	// 			Код:   Прочее,
	// 			Текст: "Email занят",
	// 		}
	// 	}
	// 	return false, ОшибкаСервиса{
	// 		Код:   Прочее,
	// 		Текст: "Email занят",
	// 	}
	// }
	// Инфо("  %+v  %+v \n", данные, ошибка)
	return true, СтатусСервиса{
		Код:   Ок,
		Текст: "Email свободен",
	}
}

func ПроверитьКод(сообщение *Сообщение) (bool, error) {
	Инфо("ПроверитьКод , нужно проверить отправленный смс код или код на email \n")
	файл, err := os.ReadFile("authCode/" + сообщение.ИдКлиента.String())
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	// кодПроверки := map[string]string{
	// 	токенВерификации: проверочныйКод,
	// 	"email":          email,
	// }
	кодПроверки := make(map[string]string)
	json.Unmarshal(файл, &кодПроверки)

	форма := сообщение.Запрос.Форма
	токенВерификации, естьТокен := форма["token"]
	if !естьТокен || len(токенВерификации) < 1 {
		Ошибка(" Нет токена верификации в данных формы %+v \n", форма)
	}
	код, естьКод := форма["code"]
	if !естьКод || len(код) < 1 {
		Ошибка(" Нет кода подтверждения в данных формы %+v \n", форма)
	}

	if СохранённыйКод, есть := кодПроверки[токенВерификации[0]]; есть {
		if СохранённыйКод == код[0] {

			ответ := сообщение.Ответ[Сервис]
			ответ.Сервис = Сервис
			ответ.ЗапросОбработан = true
			ответ.Данные = map[string]string{
				"статусПроверкиКода": "Проверочный код верный",
			}

			return true, nil
		} else {
			if кодПроверки["email"] != "" {
				//
				ОтправитьПроверочныйКод(сообщение, кодПроверки["email"])
			}

		}
	}

	return true, nil
}

/*
принимает массив возможных Имён искомого поля формы,[]string{"login", "Логин"} []string{"password", "Пароль"} возвращает значение первого попавшегося значения даже если их несколько
*/
func ПолучитьЗначениеПоляФормы(вариантИмениПоля []string, форма map[string][]string) (string, СтатусСервиса) {
	for _, имяПоля := range вариантИмениПоля {
		значение, есть := форма[имяПоля]
		if есть {
			if len(значение) == 1 {
				return значение[0], СтатусСервиса{
					Код: Ок,
				}
			} else {
				return "", СтатусСервиса{
					Код:   БолееОдногоЗначения,
					Текст: "Более одного значения в поле формы " + имяПоля,
				}
			}
		}
		return "", СтатусСервиса{
			Код:   ПустоеПолеФормы,
			Текст: "Пустое Поле Формы " + имяПоля,
		}
	}
	return "", СтатусСервиса{
		Код:   ПустоеПолеФормы,
		Текст: "Нет полей для поиска",
	}
}

func ПолучитьВсеЗначенияПоляФормы(имяПоля string, форма map[string][]string) ([]string, ОшибкаСервиса) {
	значение, есть := форма[имяПоля]
	if есть {
		if len(значение) > 0 {
			return значение, ОшибкаСервиса{
				Код: Ок,
			}
		} else {
			return nil, ОшибкаСервиса{
				Код:   ПустоеПолеФормы,
				Текст: "Пустое Поле Формы " + имяПоля,
			}
		}
	}
	return nil, ОшибкаСервиса{
		Код:   ПустоеПолеФормы,
		Текст: "Пустое Поле Формы " + имяПоля,
	}
}

func ПроверитьДанныеАвторизацииВБД(ИдКлиента string, логин string, пароль string) (ДанныеКлиента, СтатусБазы) {
	// хэшПароля, err := bcrypt.GenerateFromPassword([]byte(пароль), bcrypt.DefaultCost)
	// if err != nil {
	// 	Ошибка(" %+v \n", err.Error())
	// // }
	// данныеКлиента, статус := БазаКлиентов.Найти("Логин", логин)
	// if статус.Код != Ок {
	// 	return ДанныеКлиента{}, СтатусБазы{
	// 		Код:   ОшибкаКлючНеНайден,
	// 		Текст: "пользователь с логином [" + логин + "] не найден",
	// 	}
	// }

	// парольИзБазы := данныеКлиента[логин].Данные["Пароль"].(string)

	// if err := bcrypt.CompareHashAndPassword([]byte(парольИзБазы), []byte(пароль)); err != nil {
	// 	return ДанныеКлиента{}, СтатусБазы{
	// 		Код:   ОшибкаАвторизации,
	// 		Текст: "Неверный логин или пароль",
	// 	}
	// }
	// return ПреобразоватьДокументБДвДанныеКлиента(данныеКлиента[логин]), СтатусБазы{
	// 	Код:   Ок,
	// 	Текст: "Пользователь [" + логин + "] Авторизован",
	// }
	запрос := ДанныеЗапроса{
		Запрос: `query User($login: string, $password: string) {
			User(func: eq(<логин>, $login)) @filter(checkpwd(<пароль>, $password)) {
				<идКлиента>
				<имя>
				<фамилия>
				<отчество>
				<логин>				
				email					
				<права_доступа>				
				<статус>
				<аватар>				
				jwt
			}
		}
		`,
		Данные: map[string]string{
			"$login":    логин,
			"$password": пароль,
		},
	}

	ответ, статус := База.Получить(запрос) // граф - экземпляр вашего Dgraph API

	if статус.Код != Ок {
		return ДанныеКлиента{}, статус
	}

	клиент := ДанныеКлиента{}
	ошибка := ИзJson(ответ, &клиент)

	if ошибка != nil {
		Ошибка("  %+v \n", ошибка)
	}

	return клиент, СтатусБазы{
		Код: Ок,
	}

}

// func ПреобразоватьДанныеКлиентаВДокументБД(данныеКлиента ДанныеКлиента) *Документ {
// 	return &Документ{
// 		Данные: map[string]interface{}{
// 			"ИдКлиента": данныеКлиента.ИдКлиента,
// 			"Логин":     данныеКлиента.Логин,
// 			"Пароль":    данныеКлиента.Пароль,
// 			"Email":     данныеКлиента.Email,
// 			"Роль":      данныеКлиента.Роль,
// 			"Права":     данныеКлиента.Права,
// 			"JWT":       данныеКлиента.JWT,
// 		},
// 	}
// }

// func ПреобразоватьДокументБДвДанныеКлиента(данныеКлиента Документ) ДанныеКлиента {

//		ИдКлиента, err := uuid.Parse(данныеКлиента.Данные["ИдКлиента"].(string))
//		if err != nil {
//			Ошибка("  %+v \n", err.Error())
//		}
//		return ДанныеКлиента{
//			ИдКлиента: ИдКлиента,
//			Логин:     данныеКлиента.Данные["Логин"].(string),
//			// Пароль:    данныеКлиента.Данные["Пароль"].(string),
//			Email: данныеКлиента.Данные["Email"].(string),
//			Роль:  данныеКлиента.Данные["Роль"].([]string),
//			Права: данныеКлиента.Данные["Права"].([]string),
//			JWT:   данныеКлиента.Данные["JWT"].(string),
//		}
//	}
func СохранитьКлиентаВБД(новыйКлиент ДанныеКлиента) СтатусСервиса {

	// ошибкаБазы := БазаКлиентов.ВставитьДокумент(ПреобразоватьДанныеКлиентаВДокументБД(новыйКлиент), false)
	// if ошибкаБазы.Код != Ок {
	// 	return ОшибкаСервиса{
	// 		Код:   ошибкаБазы.Код,
	// 		Текст: ошибкаБазы.Текст,
	// 	}
	// }

	return СтатусСервиса{
		Код:   Ок,
		Текст: "успешная запись",
	}
}

func СохранитьКодАвторизации(email string, ИдКлиента string, проверочныйКод string, токенВерификации string) error {

	кодПроверки := map[string]string{
		токенВерификации: проверочныйКод,
		"email":          email,
	}

	данныеДляСохранения, err := json.Marshal(кодПроверки)
	if err != nil {
		Ошибка("  %+v \n", err)
		return err
	}
	err = os.WriteFile("authCode/"+ИдКлиента, данныеДляСохранения, 0644)
	if err != nil {
		Ошибка("  %+v \n", err)
		return err
	}
	return nil
}
func ОтправитьEmail(кому []string, тема string, тело string) error {
	from := "79880970078@ya.ru"
	password := "Satori@27$"
	smtpHost := "smtp.yandex.ru"
	smtpPort := "465"

	auth := smtp.PlainAuth("", from, password, smtpHost)

	msg := []byte("To: " + кому[0] + "\r\n" +
		"Subject: " + тема + "\r\n" +
		"\r\n" +
		тело + "\r\n")

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, кому, msg)
	if err != nil {
		return err
	}

	return nil
}
func СгенерироватьПроверочныйКод(КоличествоСимволов int) string {
	код, err := rand.Int(
		rand.Reader,
		big.NewInt(int64(math.Pow(10, float64(КоличествоСимволов)))),
	)
	if err != nil {
		panic(err)
	}

	str := код.String()
	if len(str) < КоличествоСимволов {
		str = "0" + str
	}
	return str
}
