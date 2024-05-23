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

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/DGApi"
	. "aoanima.ru/Logger"
	. "aoanima.ru/QErrors"

	json "github.com/json-iterator/go"
	"github.com/quic-go/quic-go"
)

var клиент = make(Клиент)
var Сервис ИмяСервиса = "Авторизация"

// TODO

// type ДанныеКлиента struct {
// 	Аутентифицирован            bool      `json:"аутентифицирован,omitempty"`
// 	КоличествоОшибокАвторизации bool      `json:"количество_неудачных_попыток_входа,omitempty"`
// 	Имя                         string    `json:"имя,omitempty"`
// 	Фамилия                     string    `json:"фамилия,omitempty"`
// 	Отчество                    string    `json:"отчество,omitempty"`
// 	ИдКлиента                   uuid.UUID `json:"ид_клиента"`
// 	Роль                        []string  `json:"роль,omitempty"`
// 	Права                       []string  `json:"права_доступа,omitempty"`
// 	Статус                      string    `json:"статус,omitempty"`
// 	Аватар                      string    `json:"аватар,omitempty"`
// 	Email                       string    `json:"email,omitempty"`
// 	Логин                       string    `json:"логин,omitempty"`
// 	Пароль                      string    `json:"пароль,omitempty"`
// 	JWT                         string    `json:"jwt,omitempty"`
// 	Телефон                     string    `json:"телефон,omitempty"`
// 	Адрес                       Адрес     `json:"адрес,omitempty"`
// 	Создан                      time.Time `json:"создан,omitempty"`
// 	Обновлен                    time.Time `json:"обновлен,omitempty"`
// 	ОСебе                       string    `json:"о_себе,omitempty"`
// 	СоцСети                     []string  `json:"социальные_ссылки,omitempty"`
// 	Профиль                     map[string]interface{}
// }

// type Адрес struct {
// 	Страна        string `json:"страна,omitempty"`
// 	Город         string `json:"город,omitempty"`
// 	Район         string `json:"район,omitempty"`
// 	ТипУлицы      string `json:"тип_улицы,omitempty"`
// 	НазваниеУлицы string `json:"название_улицы,omitempty"`
// 	НомерДома     string `json:"номер_дома,omitempty"`
// 	Корпус        string `json:"корпус,omitempty"`
// 	НомерКвартиры string `json:"номер_квартиры,omitempty"`
// }
// type Секрет struct {
// 	ИдКлиента string    `json:"ид_клиента"`
// 	Секрет    string    `json:"секрет"`
// 	Обновлен  time.Time `json:"обновлен"`
// }

var База СоединениеСДГраф
var СекретноеСоединение СоединениеСДГраф

// var ПраваДоступа = []string{"чтение", "просмотр", "изменение своего", "создание пользователей", "удаление пользователей", "изменение ролей"}

var РолиПользователей = []string{"гость", "администратор", "покупатель", "продавец", "управляющий"}

var СхемаБазы = `<ид_клиента>: string @index(exact) @upsert .
				<права_доступа>: [string] .
				<секрет> : string .
				<СекретКлиента> : uid .
				<имя>: string  .
				<фамилия>: string  .
				<отчество>: string  .
				<логин>: string @index(exact) @upsert .
				<пароль>: password .
				email: string @index(exact) @upsert .
				<телефон>: string @index(exact)  @upsert .
				<роль>: string  .
				<создан>: datetime .
				<обновлен>: string  .
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
				<количество_неудачных_попыток_входа>: int .
				<количество_удачных_попыток_входа>: int .
				<aутентифицирован>: bool .
				jwt: string .
						type <СекретКлиента> {	
								<ид_клиента> 
								<секрет> 
								<обновлен> 							
							}
							type <Адрес> {
									<страна>
									<город>
									<район>
									<тип_улицы>
									<название_улицы>
									<номер_дома>
									<корпус>
									<номер_квартиры>
							}
							type <Пользователь> {
								<aутентифицирован>
								<количество_неудачных_попыток_входа>	
								<количество_удачных_попыток_входа>	
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

	// каталогСтатичныхФайлов = "../../jetHTML/static/"
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

	статус := База.Схема(СхемаБазы)

	if статус.Код != Ок {
		Ошибка(" Ошибка записи схемы  %+v \n", статус)
	}

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

	Ошибка("  ОбработчикЗапросовСервера НОВЫЙ ЗАПРОС  %+v \n", сообщение)
	Инфо("  ОбработчикЗапросовСервера %+v \n", сообщение)
	var err error
	var статусСервиса СтатусСервиса
	var ok bool
	var Действие string

	if сообщение.Запрос.Действие != "" {
		Действие = сообщение.Запрос.Действие
	} else {
		маршрутЗапроса, err := url.Parse(сообщение.Запрос.МаршрутЗапроса)
		Инфо(" маршрутЗапроса %+v \n", маршрутЗапроса)
		if err != nil {
			Ошибка("Ошибка при парсинге СтрокаЗапроса запроса:", err)
		}

		маршрутЗапроса.Path = strings.Trim(маршрутЗапроса.Path, "/")
		дейсвтия := strings.Split(маршрутЗапроса.Path, "/")

		if len(дейсвтия) == 0 {
			Инфо(" Пустой маршрут, добавляем в маршруты обработку по умолчанию: авторизация \n")
			// Читаем заголовки парсим и проверяем JWT
			Действие = "авторизация" //првоерим и валидируем токен, получим права доступа

		} else {
			Действие = дейсвтия[0]
		}
	}
	Инфо(" Выполняем: %+v \n", Действие)

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
			// ответ.ИмяШаблона = "ВыборСпособа2ФАвторизации"

			ответ.Данные = map[string]interface{}{
				"Аутентифицирован": true,
				"ДанныеКлиента":    данныеКлиента,
			}
		} else {
			// ответ.ИмяШаблона = "Ошибка авторизации"
			ответ.Данные = map[string]interface{}{
				"Аутентифицирован": false,
				"Ошибка":           статус.Текст,
				"Код ошибки":       статус.Код,
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
			ответ.Данные = map[string]interface{}{
				"ТокенВерный": true,
			}
		} else {
			ответ.Данные = map[string]interface{}{
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
		Инфо(" %+v \n", "авторизация")

		// Авторизация, получение прав доступа пользователя....
		ПрошёлВалидацию, статус := ВлаидацияТокена(&сообщение)
		if статус.Код != Ок {
			Ошибка("  %+v \n", статус)
		}
		статус = Авторизация(ПрошёлВалидацию, &сообщение) // получем права доступа пользователя
		Инфо("Авторизация статус %+v \n", статус)
		if статус.Код != Ок {
			Ошибка("  %+v \n", статус)
		}

		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ЗапросОбработан = true

		if статус.Код != Ок {
			ответ.Данные = map[string]interface{}{
				"ТокенВерный": true,
			}
		} else {
			ответ.Данные = map[string]interface{}{
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
		if статус.Код != Ок {
			Ошибка("  %+v \n", статус)
		}
		статус = Авторизация(ПрошёлВалидацию, &сообщение) // получем права доступа пользователя, записываем их в структуру  сообщение.ТокенКлиента.Права
		if статус.Код != Ок {
			Ошибка("  %+v \n", статус)
		}
		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ЗапросОбработан = true
		if статус.Код == Ок {
			ответ.Данные = map[string]interface{}{
				"ТокенВерный": true,
			}
		} else {
			ответ.Данные = map[string]interface{}{
				"ТокенВерный": false,
			}
		}
		ответ.СтатусОтвета = статус
		сообщение.Ответ[Сервис] = ответ

		ОтправитьСообщение(поток, сообщение)

		return
	}

}
func ПолуитьПраваДоступаИзБД(ИдКлиента string) ([]byte, СтатусСервиса) {
	Инфо(" ПолуитьПраваДоступаИзБД %+v \n", ИдКлиента)

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
			сообщение.ТокенКлиента.Роль = []string{"гость"}
			return статус
		}
	}
	// Если пользователь не прошёл валидацию то назначаем ему права Гостя
	сообщение.ТокенКлиента.Роль = []string{"гость"}
	return СтатусСервиса{
		Код:   ПользовательНеОпознан,
		Текст: "Пользователь не опознан",
	}

}

func АутентифицироватьПользователяВБД(логин string, пароль string) ([]byte, СтатусСервиса) {
	данные := ДанныеЗапроса{
		Запрос: `query User($login: string, $pass: string) {
					 var(func: eq(<логин>, $login) ) {
						<статус_пароля> as checkpwd(<пароль>, $pass)
					}
					<ПарольВерный>(func: eq(val(<статус_пароля>), 1)) {
						<aутентифицирован>: val(<статус_пароля>)
						uid
						<ид_клиента>
						<имя>
						<отчество>
						<логин>
						email
						<права_доступа>
						<статус>
						jwt					
						<количество_неудачных_попыток_входа>
						<удача> as <количество_удачных_попыток_входа>
						<количество_удач> as math(<удача>+1)
						<секрет> {
							<секрет> 
							<обновлен> 		
						}	
					}

					<ПарольНеВерный>(func: eq(val(<статус_пароля>), 0)) {
						<aутентифицирован>: val(<статус_пароля>)
						<неудача> as <количество_неудачных_попыток_входа>
						<количество_неудач> as math(<неудача>+1)	
					}
				}
				`,
		Мутация: []Мутация{
			{
				Условие: "@if(ge(len(<удача>), 1))",
				Мутация: []byte(`
									{
									"uid": "uid(статус_пароля)",
									"количество_удачных_попыток_входа": "val(количество_удач)",
									"количество_неудачных_попыток_входа": 0
									}
								`),
			},
			{
				Условие: "@if(ge(len(<неудача>), 1))",
				Мутация: []byte(`
									{
									"uid": "uid(статус_пароля)",
									"количество_неудачных_попыток_входа": "val(количество_неудач)"
									}
				`),
			},
		},
		Данные: map[string]string{
			"$login": логин,
			"$pass":  пароль,
		},
	}
	ответ, статусЗапроса := База.Изменить(данные)
	if статусЗапроса.Код != Ок {
		return ответ, СтатусСервиса{
			Код:   ОшибкаАвторизации,
			Текст: статусЗапроса.Текст,
		}
	}

	Инфо(" %+v  %+v \n", ответ, статусЗапроса)

	return ответ, СтатусСервиса{
		Код:   Ок,
		Текст: статусЗапроса.Текст,
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
		данныеАутентификации, статус := АутентифицироватьПользователяВБД(логин[0], пароль[0])

		if статус.Код != Ок {
			Ошибка("АутентифицироватьПользователяВБД статус %+v \n", статус)
			return ДанныеКлиента{}, СтатусСервиса{
				Текст: статус.Текст,
				Код:   ОшибкаАвторизации,
			}
		}

		данныеКлиента := ДанныеКлиента{}
		ошибка := ИзJson(данныеАутентификации, &клиент)
		if ошибка != nil {
			Ошибка(" шибка преобразования  %+v \n", ошибка.Error())
			return ДанныеКлиента{}, СтатусСервиса{
				Текст: ошибка.Error(),
				Код:   ОшибкаПреобразованияДокумента,
			}
		}

		if !данныеКлиента.Аутентифицирован {
			return данныеКлиента, СтатусСервиса{
				Текст: "Логин и пароль не совпадают, попробуйте ещё раз",
				Код:   ОшибкаАвторизации,
			}
		}

		// сообщение.ТокенКлиента = данныеКлиента.
		JWT, ошибкаСозданияJWT := СоздатьJWT(данныеКлиента.ИдКлиента.String(), Секрет{})
		if ошибкаСозданияJWT.Код != Ок {
			Ошибка(" не удалось создать токен  %+v \n", ошибкаСозданияJWT)
			return данныеКлиента, СтатусСервиса(ошибкаСозданияJWT)
		}
		данныеКлиента.JWT = JWT
		сообщение.JWT = JWT
		сообщение.ДанныеКлиента = данныеКлиента
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
	ответ.Данные = map[string]interface{}{
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
	свободен, статусЗапроса := ЛогинСвободен(логин)

	Инфо(" ЛогинСвободен ок  %+v \n", статусЗапроса)

	ответ := сообщение.Ответ[Сервис]

	ответ.СтатусОтвета = статусЗапроса
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true
	ответ.Данные = map[string]interface{}{
		"Статус": "Логин свободен",
	}

	сообщение.Ответ[Сервис] = ответ

	return свободен, статусЗапроса
}

func ЛогинСвободен(логин string) (bool, СтатусСервиса) {
	Инфо("ЛогинСвободен , нужно проверить логин на свободу \n")

	ответ, статус := База.Получить(ДанныеЗапроса{
		Запрос: `
			query checkLogin($login: string){
				<Логин>(func: eq(<логин>, $login)) {
				  <занято>:count(uid)
				}
			}`,
		Данные: map[string]string{
			"$login": логин,
		},
	})

	if статус.Код != Ок {
		Инфо(" %+v \n", статус.Текст)
		return false, СтатусСервиса{
			Код:   статус.Код,
			Текст: статус.Текст,
		}
	}

	картаОтвета := ОтветИзБазы{}
	ошибкаРазбора := ИзJson(ответ, &картаОтвета)
	if ошибкаРазбора != nil {
		Ошибка(" Не удалось разобрать ответ %+v \n", ошибкаРазбора.Error())
	}

	for имя, массивДанных := range картаОтвета {
		if len(массивДанных) == 1 {
			Инфо(" %+v  %+v \n", имя, массивДанных)
			Занято := массивДанных[0]["занято"]
			Инфо(" %+v  %+v \n", Занято, uint8(Занято.(float64)) > 0)

			if uint8(Занято.(float64)) > 0 {

				return false, СтатусСервиса{
					Код:   ЛогинЗанят,
					Текст: "Логин занят",
				}
			} else {
				return true, СтатусСервиса{
					Код:   Ок,
					Текст: "Логин свободен",
				}
			}
		} else if len(массивДанных) > 1 {
			Ошибка(" Возвращено более 1 записи %+v \n", массивДанных)
			return false, СтатусСервиса{
				Код:   ЛогинЗанят,
				Текст: "Логин занят",
			}
		}
	}

	Инфо(" %+s %+v \n", ответ, статус)

	return true, СтатусСервиса{
		Код:   Ок,
		Текст: "Логин свободен",
	}
}

func ПроверитьEmail(сообщение *Сообщение) (bool, СтатусСервиса) {
	email := сообщение.Запрос.Форма["Email"][0]
	свободен, статусЗапроса := EmailСвободен(email)

	ответ := сообщение.Ответ[Сервис]

	ответ.СтатусОтвета = статусЗапроса
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true
	ответ.Данные = map[string]interface{}{
		"Статус": статусЗапроса.Текст,
	}

	сообщение.Ответ[Сервис] = ответ
	Инфо(" EmailСвободен ок  %+v \n", статусЗапроса)
	return свободен, статусЗапроса
}

func EmailСвободен(email string) (bool, СтатусСервиса) {

	ответ, статус := База.Получить(ДанныеЗапроса{
		Запрос: `
			query checkEmail ($email: string){
				Emails(func: eq(email, $email)) {
					<занято>: count(uid)
				}
			}`,
		Данные: map[string]string{
			"$email": email,
		},
	})
	Инфо(" %+s %+v \n", ответ, статус)

	if статус.Код != Ок {
		Инфо(" %+v \n", статус.Текст)
		return false, СтатусСервиса{
			Код:   статус.Код,
			Текст: статус.Текст,
		}
	}

	картаОтвета := ОтветИзБазы{}
	ошибкаРазбора := ИзJson(ответ, &картаОтвета)
	if ошибкаРазбора != nil {
		Ошибка(" Не удалось разобрать ответ %+v \n", ошибкаРазбора.Error())
	}

	for имя, массивДанных := range картаОтвета {
		if len(массивДанных) == 1 {
			Инфо(" %+v  %+v \n", имя, массивДанных)
			Занято := массивДанных[0]["занято"]
			Инфо(" %+v  %+v \n", Занято, uint8(Занято.(float64)) > 0)

			if uint8(Занято.(float64)) > 0 {

				return false, СтатусСервиса{
					Код:   EmailЗанят,
					Текст: "Email занят",
				}
			} else {
				return true, СтатусСервиса{
					Код:   Ок,
					Текст: "Email свободен",
				}
			}
		} else if len(массивДанных) > 1 {
			Ошибка(" Возвращено более 1 записи %+v \n", массивДанных)
			return false, СтатусСервиса{
				Код:   EmailЗанят,
				Текст: "Email занят",
			}
		}
	}

	Инфо(" %+s %+v \n", ответ, статус)

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
			ответ.Данные = map[string]interface{}{
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

/*
принимает массив возможных Имён искомого поля формы,[]string{"login", "Логин"} []string{"password", "Пароль"} возвращает значение первого попавшегося значения даже если их несколько
*/
// func ПолучитьЗначениеПоляФормы(вариантИмениПоля []string, форма map[string][]string) (string, СтатусСервиса) {
// 	for _, имяПоля := range вариантИмениПоля {
// 		значение, есть := форма[имяПоля]
// 		if есть {
// 			if len(значение) == 1 {
// 				return значение[0], СтатусСервиса{
// 					Код: Ок,
// 				}
// 			} else {
// 				return "", СтатусСервиса{
// 					Код:   БолееОдногоЗначения,
// 					Текст: "Более одного значения в поле формы " + имяПоля,
// 				}
// 			}
// 		}
// 		return "", СтатусСервиса{
// 			Код:   ПустоеПолеФормы,
// 			Текст: "Пустое Поле Формы " + имяПоля,
// 		}
// 	}
// 	return "", СтатусСервиса{
// 		Код:   ПустоеПолеФормы,
// 		Текст: "Нет полей для поиска",
// 	}
// }

// func ПолучитьВсеЗначенияПоляФормы(имяПоля string, форма map[string][]string) ([]string, ОшибкаСервиса) {
// 	значение, есть := форма[имяПоля]
// 	if есть {
// 		if len(значение) > 0 {
// 			return значение, ОшибкаСервиса{
// 				Код: Ок,
// 			}
// 		} else {
// 			return nil, ОшибкаСервиса{
// 				Код:   ПустоеПолеФормы,
// 				Текст: "Пустое Поле Формы " + имяПоля,
// 			}
// 		}
// 	}
// 	return nil, ОшибкаСервиса{
// 		Код:   ПустоеПолеФормы,
// 		Текст: "Пустое Поле Формы " + имяПоля,
// 	}
// }

// func ПроверитьДанныеАвторизацииВБД(логин string, пароль string) (ДанныеКлиента, СтатусБазы) {

// 	запрос := ДанныеЗапроса{
// 		Запрос: `query User($login: string, $password: string) {

// 			<ПроверитьПароль>(func: eq(<логин>, $login) ) {
// 				<статус_пароля> as   checkpwd(<пароль>, $password)
// 			}
// 			<ПарольВерный>(func: eq(val(<статус_пароля>), 1)) {
// 				<аутентифицирован>: val(<статус_пароля>)
// 				uid
// 				<ид_клиента>
// 				<имя>
// 				<фамилия>
// 				<отчество>
// 				<логин>
// 				email
// 				<права_доступа>
// 				<статус>
// 				jwt
// 			  }

// 			  <ПарольНеВерный>(func: eq(val(<статус_пароля>), 0)) {
// 				<аутентифицирован>: val(<статус_пароля>)
// 				expand(_all_)
// 			  }

// 		}`,
// 		Данные: map[string]string{
// 			"$login":    логин,
// 			"$password": пароль,
// 		},
// 	}

// 	ответ, статус := База.Получить(запрос) // граф - экземпляр вашего Dgraph API

// 	if статус.Код != Ок {
// 		return ДанныеКлиента{}, статус
// 	}

// 	клиент := ДанныеКлиента{}
// 	ошибка := ИзJson(ответ, &клиент)

// 	if ошибка != nil {
// 		Ошибка("  %+v \n", ошибка)
// 	}

// 	return клиент, СтатусБазы{
// 		Код:   Ок,
// 		Текст: "Логин и пароль успещно проверены, данные аутентификации в соответствющем поле ДанныхКлиента",
// 	}

// }

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
// })has(email)
// query:{
//     checkEmail(func: has(<маршрут>)) {
//  		   expand(_all_)
//  		}
//   }
// ответ, статусхемы := База.Получить(ДанныеЗапроса{
// 	Запрос: `{
// 		checkLogin(func: eq(<логин>, "user6")) {
// 		  count(uid)
// 		}
// 		checkEmail(func: eq(email, "alexey@example.com")) {
// 		  count(uid)
// 		}
// 	  }`,
// 	Данные: nil,
// })
// {
//   me(func: eq(name@en, "Steven Spielberg")) @filter(has(director.film)) {
//     name@en
//     director.film @filter(allofterms(name@en, "jones indiana") OR allofterms(name@en, "jurassic park"))  {
//       uid
//       name@en
//     }
//   }
// }

// Инфо("   %+s %+v \n", ответ, статусхемы)

// ответ, статус := База.Получить(ДанныеЗапроса{
// 	Запрос: `schema {
// 		type
// 		index
// 		}`,
// query: {
// 	checkRoute(func: allofterms(<маршрут>, "рабочийСтол")) {
// 					  <описание>
// 				  }
//   }

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
