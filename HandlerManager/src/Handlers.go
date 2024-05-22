package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/DGApi"
	. "aoanima.ru/Logger"
	. "aoanima.ru/QErrors"
	"github.com/quic-go/quic-go"
)

/*
Очередь обработчиков
Каждый запрос от клиента может быть обработан одним и более количеством микросервисов, для того чтобы правильно отправлять запрос в сервисы нужно описать последовательность обработки запроса Сервиссами, и указать какой HTNL шаблон рендерить, или куда сделать редирект например после автоизации или регистрации)

КРоме того необходимо учитывать права доступа и роли пользователя, чтобы один и тот же маршрут по разному обрабатывался в заивсимости от роли польтзователя и его прав доступа.

дествие  | сервис | маршрут | роль | права | шаблон |  статусОтвета | редирект | ассинхронно |

	| Рендер | /формаРеситрации | ["гость"] | "формаРегистрации" | Ок | /личныйКабинет | нет

регистрация  | Авторизация | /формаРеситрации (url же не меняется) | ["гость"] |   |  |
*/

/*
Обработчик может указываться если значеине в поле действие называетя иначе чем функция которая должна обработать запрос
например действие: регистрация
Обработчик : сохранитьПользователя

таким образом разные действия могут обрабатыватся одним и тем же обработчиком или несколькитми обработчиками

например

дейсвтие: 	оформитьЗаказ
обработчики: сохранитьЗаказ

	отправитьПисьмоКлиенту
	отправитьУведомлениеПродавцу

маршрут: /формаЗаказа - по идее маршрут может быть любой. И обрабатывается если нет дейсвтия
шаблон: /формаЗаказа

По факту нужно проектировать запросы так что если есть только маршрут
exotiki.ru/каталог/цветы - то тут обработчик должен определяться из маршрута, типа шаблон каталог, со списком из категории цветы.

Обращаемся к обработчику отвечающему за базу с товарами
обработчик : получитьТовары - получаем товары из категории "цветы"
шаблон:      каталогТоваров

# Если нужно в зависимости от результата обработки выводить разный шаблон, то можно использовать структуру СОобщение

	map[int]Шаблон - int = Код из QErrors
	Шаблон структура, в которой опиывается при каком коде ответа работы сервиса , какой шаблон рендерить
*/

/*
Добавляет данные об обработчике в БД, особенно важно права доступа
Маршрут может быть пустой если есть действие, и наоборот, если есть маршрут а обработчика нету. не страшно, обработчик будет вычисляться из маршрута.
если есть Действие то оно в приоритете

	// рабочий запрос
	// {
	// 	node(func: eq(<маршрут>, "рабочий/настройки"))  {
	// 	  <доступ>{	//
	// 	  <роль> @filter(eq(<код>, 2)) {
	// 		uid
	// 		<код>
	// 		<имя_роли>
	// 	  }

	// 	}
	//   }
	//   }
*/
func ДобавитьОбработчик(поток quic.Stream, сообщение Сообщение) {

	ответ := сообщение.Ответ[Сервис]
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true

	маршрут, статусМаршрут := ПолучитьЗначениеПоляФормы("маршрут", сообщение.Запрос.Форма)

	комманда, статусДейсвтия := ПолучитьЗначениеПоляФормы("имяКоманды", сообщение.Запрос.Форма)

	if статусДейсвтия.Код != Ок && статусМаршрут.Код != Ок {
		ответ.СтатусОтвета = СтатусСервиса{
			Код:   Прочее,
			Текст: "действие и маршрут не заданы, должно быть установлено одно или оба поля",
		}
		сообщение.Ответ[Сервис] = ответ
		ОтправитьСообщение(поток, сообщение)
	}

	очередьОбработчиков, ассинхронныеОбработчики, праваДоступа := СтруктурироватьДанныеФормы(сообщение.Запрос.Форма)

	Инфо("очередьОбработчиков  %+v ассинхронныеОбработчики %+v \n", очередьОбработчиков, ассинхронныеОбработчики)

	// Шаблон можно указывать как путь рабочийСтол/обработчики - где рабочийСтол это основной слой контента, а обработчики это шаблон который вставляется внутрь предидущего слоя.

	шаблон, статусШаблон := ПолучитьЗначениеПоляФормы("шаблон", сообщение.Запрос.Форма)
	if статусШаблон.Код != Ок {
		Ошибка(" статус получения описания  %+v \n", статусШаблон)
	}
	ш := Шаблон{
		Тип:        "Шаблон",
		Код:        new(int),
		ИмяШаблона: шаблон,
	}
	*ш.Код = Ок

	новыйОбработчик := &ОбработчикМаршрута{
		Тип:                 "ОбработчикМаршрута",
		Маршрут:             маршрут,
		Комманда:            комманда,
		ОчередьОбработчиков: очередьОбработчиков,
		АссинхроннаяОчередьОбработчиков: ассинхронныеОбработчики,
		Шаблон:       ш,
		ИмяШаблона:   шаблон,
		ПраваДоступа: праваДоступа,
	}

	обработчикБин, статус := Json(новыйОбработчик)
	if статус != nil {
		Ошибка(" статус %+v обработчикБин %+v \n", статус, обработчикБин)
	}
	Инфо(" новыйОбработчик %+v  обработчикБин %+s \n", новыйОбработчик, обработчикБин)

	запрос, данныеПодстановки := собратьЗапросВставкиОбработчика(маршрут, праваДоступа)
	Инфо(" %+v %+v \n", запрос, данныеПодстановки)

	данные := ДанныеЗапроса{
		Запрос: запрос,
		Мутация: []Мутация{
			{
				Условие: "@if(eq(len(count_roles),0))",
				Мутация: обработчикБин,
			},
		},
		Данные: данныеПодстановки,
	}

	Инфо("данные  %+v \n", данные)

	результатИзменения, статусБазы := База.Изменить(данные)
	if статусБазы.Код != Ок {
		Ошибка(" статус %+v \n данные %+v \n", статусБазы, данные)
	}
	var данныеОтвета interface{}
	ИзJson(результатИзменения, &данныеОтвета)
	Инфо("Исходные данные %+v \n ответ %+s \n", данные, данныеОтвета)

	ответ.СтатусОтвета = СтатусСервиса{
		Код:   статусБазы.Код,
		Текст: статусБазы.Текст,
	}

	/******Получим данные нового узла, с UID для вставки в html */
	данные = ДанныеЗапроса{
		Запрос: `query <Обработчики>($handler : string, $path : string, $action : string) {					
							<Обработчик>(func: has(<обработчик>)){	
								uid			
								<маршрут>
								<действие>
								<обработчик>
								<доступ>{
									<пользователи>
									<права>
									<роль>
									uid
									dgraph.type
								}
								<описание>
								<шаблонизатор> {
									uid
									<имя_шаблона>
									<код>
									dgraph.type
								}
								<ассинхронно>
								dgraph.type
								expand(_all_)
							}	
			 			} `,

		Данные: map[string]string{
			"$handler": "создатьОбработчик",
			"$path":    "/редакторОбработчиков",
			"$action":  "создатьОбработчик",
		},
	}

	результатИзменения, статусБазы = База.Получить(данные)
	if статусБазы.Код != Ок {
		Ошибка(" статус %+v \n", статусБазы)
	}
	var данныеНовогоУзла КонфигурацияОбработчика
	ИзJson(результатИзменения, &данныеНовогоУзла)
	Инфо(" данныеНовогоУзла %+s \n", данныеНовогоУзла)

	ответ.Данные = данныеНовогоУзла
	ответ.ИмяШаблона = "новыйОбработчик"
	сообщение.Ответ[Сервис] = ответ
	ОтправитьСообщение(поток, сообщение)
}

/*
ДОК:
Суть запроса для вставки с проверкой на уникальность, запросить данные только того поля уникальность которого проверяется,
например как тут
нужно проверить маршрут+код.роли
код.роли находиться - Данные.Доступ.Роль.Код
Поэтому первым запросом мы получем записи с маршуртом , и фильтруем роли по нужным значениям, и возвращаем только данные  объекта Роль.
в условии проверки перед вставкой работает только одна фукнция len она вохвращает тупо количество символов в ответе первого запроса, поэтому если мы будем в первом запросе возвращать uid родительского объекта Роли, то мы не сможем определить есть роли или нет.
*/
func собратьЗапросВставкиОбработчика(маршрут string, роли []ПраваДоступа) (string, map[string]string) {
	var фильтрРолей string
	var сигнатураРолей string
	данныеПодстановки := map[string]string{}

	// nodes as var(func: eq(<маршрут>,"настройки/рабочий")) {
	// 	<доступ>{
	// 	   <роль> @filter(eq(<код.роли>, 6) OR eq(<код.роли>, 5)){
	// 		   uid
	// 		   <код.роли>
	// 		   <имя.роли>
	// 		 }
	//    }
	//  }

	for i, ролиДоступа := range роли {
		фильтрРолей += fmt.Sprintf("eq(<код.роли>, $role%d)", i+1)
		сигнатураРолей += fmt.Sprintf("$role%d: string", i+1)
		if i < len(роли)-1 {
			фильтрРолей += " or "
			сигнатураРолей += ", "
		}
		данныеПодстановки["$role"+strconv.Itoa(i+1)] = strconv.Itoa(ролиДоступа.Роль.Код)
	}
	// eq(<код.роли>, 11) OR eq(<код.роли>, 12)
	запрос := fmt.Sprintf(`query <СохранитьОбработчик>($path : string, %s ) {
				var(func: eq(<маршрут>, $path)) {  
				<доступ>{     				
					count_roles as <роль> @filter(%s){
						uid
						<код.роли>
						<имя.роли>
					}    	
				} 				
			}				
			}`, сигнатураРолей, фильтрРолей)

	данныеПодстановки["$path"] = маршрут

	return запрос, данныеПодстановки
}

// func generateRoleParams(roles []string) string {
// 	var params string
// 	for i, _ := range roles {
// 		params += fmt.Sprintf("$role%d: string", i+1)
// 		if i < len(roles)-1 {
// 			params += ", "
// 		}
// 	}
// 	return params
// }

func СтруктурироватьДанныеФормы(форма map[string][]string) ([]Обработчик, []Обработчик, []ПраваДоступа) {

	очередьОбработчиков := []Обработчик{}
	очередьАссинхронныхОбработчиков := []Обработчик{}
	праваДоступа := []ПраваДоступа{}

	данные := make(map[string]map[string][]string)

	for ключ, значение := range форма {

		индексКвадратныхСкобок := strings.Index(ключ, "[")
		if индексКвадратныхСкобок == -1 {
			continue
		}

		имяПоля := ключ[:индексКвадратныхСкобок]
		идПоля := ключ[индексКвадратныхСкобок+1 : len(ключ)-1]

		if _, ok := данные[идПоля]; !ok {
			данные[идПоля] = make(map[string][]string)
		}

		данные[идПоля][имяПоля] = значение
	}

	for _, группа := range данные {
		if роль, естьРоль := группа["роль"]; естьРоль {

			кодРоли, ошибка := strconv.Atoi(роль[0])
			if ошибка != nil {
				Ошибка(" ошибка преобразования роли в число  %+v \n", ошибка)
			}

			права := make([]Права, len(группа["права_доступа"]))
			for и, кодПрав := range группа["права_доступа"] {

				КодПрав, ошибка := strconv.Atoi(кодПрав)
				if ошибка != nil {
					Ошибка(" ошибка преобразования роли в число  %+v \n", ошибка)
				}
				права[и] = Права{
					Тип:     "Права",
					Код:     КодПрав,
					ИмяПрав: "права_" + роль[0] + "_" + кодПрав,
				}
			}

			праваДоступа = append(праваДоступа, ПраваДоступа{
				Тип: "ПраваДоступа",
				Роль: Роль{
					Тип:     "Роль",
					Код:     кодРоли,
					ИмяРоли: "роль_" + роль[0],
				},
				Права: права,
			})
		}

		if очередь, естьОчередь := группа["очередь"]; естьОчередь {
			номерОчереди, ошибка := strconv.Atoi(очередь[0])

			if ошибка != nil {
				Ошибка(" ошибка преобразования очереди в число  %+v \n", ошибка)
			}

			имяСервиса, естьИмяСервиса := группа["сервис"]
			if !естьИмяСервиса {
				Ошибка(" не нашли имя сервиса %+v \n", имяСервиса)
			}
			имяОбработчика, естьимяОбработчика := группа["обработчик"]
			if !естьимяОбработчика {
				Ошибка(" не нашли имя  Обработчика %+v \n", естьимяОбработчика)
			}
			о := Обработчик{
				Очередь:        new(int),
				ИмяСервиса:     имяСервиса[0],
				ИмяОбработчика: имяОбработчика[0],
			}
			*о.Очередь = номерОчереди

			очередьОбработчиков = append(очередьОбработчиков, о)

		}
		if _, естьАссинхронно := группа["ассинхронно"]; естьАссинхронно {
			имяСервиса, естьИмяСервиса := группа["сервис"]
			if !естьИмяСервиса {
				Ошибка(" не нашли имя сервиса %+v \n", имяСервиса)
				continue
			}
			имяОбработчика, естьимяОбработчика := группа["обработчик"]
			if !естьимяОбработчика {
				Ошибка(" не нашли имя  Обработчика %+v \n", естьимяОбработчика)
				continue
			}
			очередьАссинхронныхОбработчиков = append(очередьАссинхронныхОбработчиков, Обработчик{
				ИмяСервиса:     имяСервиса[0],
				ИмяОбработчика: имяОбработчика[0],
			})

		}
	}

	Инфо(" данные%+v \n", данные)

	return очередьОбработчиков, очередьАссинхронныхОбработчиков, праваДоступа
}

func ИзменитьОбработчик(поток quic.Stream, сообщение Сообщение) {

}
func УдалитьОбработчик(поток quic.Stream, сообщение Сообщение) {

	ответ := сообщение.Ответ[Сервис]
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true

	ид_обработчика, статус := ПолучитьЗначениеПоляФормы("ид_обработчика", сообщение.Запрос.Форма)
	if статус.Код != Ок {
		Ошибка("  %+v \n", статус)
	}

	данные := ДанныеЗапроса{
		Запрос: `query <УдалитьОбработчик>($uid : string) {
							<УдаляемыеУзлы>(func: uid($uid)) {				
								<обработчик_ид> as uid
								<доступ> {
									<доступ_ид> as uid
								}
								<шаблонизатор> {
									<шаблонизатор_ид> as uid
								}
							}
			 			} `,
		Мутация: []Мутация{
			{
				Удалить: []byte(`[
					{"uid": "uid(доступ_ид)"},
					{"uid": "uid(обработчик_ид)"},
					{"uid": "uid(шаблонизатор_ид)"}
					]`),
			},
		},
		Данные: map[string]string{
			"$uid": ид_обработчика,
		},
	}
	результатИзменения, статусБазы := База.Изменить(данные)
	if статусБазы.Код != Ок {
		Ошибка(" статус %+v \n данные %+v \n", статусБазы, данные)
	}
	var данныеОтвета interface{}
	ИзJson(результатИзменения, &данныеОтвета)
	Инфо("Исходные данные %+v \n ответ %+s \n", данные, данныеОтвета)

	ответ.СтатусОтвета = СтатусСервиса{
		Код:   статусБазы.Код,
		Текст: статусБазы.Текст,
	}
	ответ.Данные = данныеОтвета
	ответ.ИмяШаблона = "всплывающееСообщение"

	сообщение.Ответ[Сервис] = ответ
	ОтправитьСообщение(поток, сообщение)

}
func СоздатьОчередьОбработчиков(поток quic.Stream, сообщение Сообщение) {

}

func ИзменитьОчередьОбработчиков(поток quic.Stream, сообщение Сообщение) {

}
func ДобавитьМаршрут(поток quic.Stream, сообщение Сообщение) {

}
func ИзменитьМаршрут(поток quic.Stream, сообщение Сообщение) {

}
func УдалитьМаршрут(поток quic.Stream, сообщение Сообщение) {

}

func ДобавитьРоль(поток quic.Stream, сообщение Сообщение) {
	ответ := сообщение.Ответ[Сервис]
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true
	Инфо(" ДобавитьРоль%+v \n", сообщение.Запрос.Форма)
	данныеНовойРоли := сообщение.Запрос.Форма

	if len(данныеНовойРоли["имя.роли"]) < 1 || данныеНовойРоли["имя.роли"][0] == "" {
		ответ.СтатусОтвета = СтатусСервиса{
			Код:   ПустоеПолеФормы,
			Текст: "Имя роли не может быть пустым",
		}
		сообщение.Ответ[Сервис] = ответ
		Ошибка(" ответ %+v \n", ответ)

		ОтправитьСообщение(поток, сообщение)
		return
	}
	if len(данныеНовойРоли["код.роли"]) < 1 {

		ответ.СтатусОтвета = СтатусСервиса{
			Код:   ПустоеПолеФормы,
			Текст: "Код роли не может быть пустым",
		}
		сообщение.Ответ[Сервис] = ответ
		Ошибка(" ответ %+v \n", ответ)
		ОтправитьСообщение(поток, сообщение)
		return
	}
	код, ошибка := strconv.Atoi(данныеНовойРоли["код.роли"][0])
	if ошибка != nil {
		Ошибка(" Ошибка конвертации кода в число  %+v \n", ошибка.Error())
	}

	новаяРоль := Роль{
		Тип:     "Роль",
		ИмяРоли: данныеНовойРоли["имя.роли"][0],
		Код:     код,
	}
	// роли := []Роль{
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Админ",
	// 		Код:     1,
	// 	},
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Модератор",
	// 		Код:     2,
	// 	},
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Продавец",
	// 		Код:     3,
	// 	},
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Модератор продавца",
	// 		Код:     4,
	// 	},
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Администратор продавца",
	// 		Код:     5,
	// 	},
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Клиент",
	// 		Код:     6,
	// 	},
	// 	{
	// 		Тип:     "Роль",
	// 		ИмяРоли: "Гость",
	// 		Код:     7,
	// 	},
	// }
	// запрос := fmt.Sprintf(`query <СохранитьОбработчик>($path : string, %s ) {
	// 	var(func: eq(<маршрут>, $path)) {
	// 	<доступ>{
	// 		count_roles as <роль> @filter(%s){
	// 			uid
	// 			<код.роли>
	// 			<имя.роли>
	// 		}
	// 	}
	// }
	// }`, сигнатураРолей, фильтрРолей)

	// запрос :=

	ролиБинар, статус := Json(новаяРоль)
	if статус != nil {
		Ошибка(" статус %+v ролиБинар %+v \n", статус, ролиБинар)
	}
	данные := ДанныеЗапроса{
		Запрос: `query <ПроверитьНаличиеРоли>($code: string){
					var(func: eq(<код.роли>, $code)) {			
						uid_role as uid			
					}
					<существующиеРоли>(func: uid(uid_role)) {						
						<код.роли>
						<имя.роли>
						uid
					}     
					
		}`,
		Мутация: []Мутация{
			{
				Условие: "@if(eq(len(uid_role), 0))",
				Мутация: ролиБинар,
			},
		},
		Данные: map[string]string{
			"$code": strconv.Itoa(новаяРоль.Код),
		},
	}
	Инфо(" %+v \n", данные)

	результатИзменения, статусБазы := База.Изменить(данные)
	if статусБазы.Код != Ок {
		Ошибка(" статус %+v \n данные %+v \n", статусБазы, данные)
	}
	Инфо(" результатИзменения %+s\n", результатИзменения)
	ответ.СтатусОтвета = СтатусСервиса{
		Код:   статусБазы.Код,
		Текст: "Новая роль добавлена",
	}
	сообщение.Ответ[Сервис] = ответ
	ОтправитьСообщение(поток, сообщение)
}
func ИзменитьРоль(поток quic.Stream, сообщение Сообщение) {

}
func УдалитьРоль(поток quic.Stream, сообщение Сообщение) {

}
func ДобавитьПрава(поток quic.Stream, сообщение Сообщение) {

}
func ИзменитьПрава(поток quic.Stream, сообщение Сообщение) {

}
func УдалитьПрава(поток quic.Stream, сообщение Сообщение) {

}

func ПолучитьОчередьОбработчиков(поток quic.Stream, сообщение Сообщение) {
	// query {
	// 	var(func: eq(<маршрут>, "/some/route")) {
	// 	  <доступ> @filter(eq(<роль>, "role1")) {
	// 		uid
	// 	  }
	// 	}

	// 	обработчики(func: uid(uid)) {
	// 	  <маршрут>
	// 	  <доступ> {
	// 		<роль> {
	// 		  <имяРоли>
	// 		}
	// 	  }
	// 	  # Другие поля обработчика маршрута
	// 	}
	//   }
	маршрутЗапроса, err := url.Parse(сообщение.Запрос.МаршрутЗапроса)
	Инфо(" маршрутЗапроса %+v \n", маршрутЗапроса)

	if err != nil {
		Ошибка("Parse маршрутЗапроса: ", err)
	}
	маршрутЗапроса.Path = strings.Trim(маршрутЗапроса.Path, "/")
	urlКарта := strings.Split(маршрутЗапроса.Path, "/")

	/*
		Для получения очереди обработчиков нужно проанализировать url и данные формы
		если метод post то аналиируем форму
		если метод get то анализируем Сообщение.ЗАпрос.СтрокаЗапроса содержащую Query часть
		если там не передан параметр "действие" то ищем обработчик из path




	*/
	if сообщение.Запрос.ТипЗапроса == GET || сообщение.Запрос.ТипЗапроса == AJAX {
		// анализируем url параметры
		параметрыЗапроса := маршрутЗапроса.Query()
		дейсвтие, естьДействие := параметрыЗапроса["действие"]
		if естьДействие {
			// получить очередь из БД
			Инфо("получить очередь из БД для: %+v \n", дейсвтие)

		} else {
			if len(urlКарта) > 0 {
				/*
					может пройти по всем частам url и получить очереь обрабочиковдля каждого шага ?
					или брать только первый ?
				*/
				дейсвтие := urlКарта[0]
				Инфо("получить очередь из БД для: %+v \n", дейсвтие)

			}
		}

	}

	if сообщение.Запрос.ТипЗапроса == POST || сообщение.Запрос.ТипЗапроса == AJAXPost {

	}

}

func ПолучитьСписокОчередей(поток quic.Stream, сообщение Сообщение) {

}
