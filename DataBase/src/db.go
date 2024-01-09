package DataBase

import (
	_ "aoanima.ru/ConnQuic"
	. "aoanima.ru/Logger"
	badger "github.com/dgraph-io/badger/v4"
)

/*

	ошибкаПодключения := т.база.Подключиться(ПутьКФайламБазы + т.Имя + ".indexes")
	if ошибкаПодключения != nil {
		Ошибка(" ошибка подключения %+v \n", ошибкаПодключения)
		return ОшибкаБазы{
			Код:   ОшибкаПодключения,
			Текст: ошибкаПодключения.Error(),
		}
	}
	т.трз = Транзакция{т.база.NewTransaction(читать)}
	таблица.ПолучитьСигнатуруТаблицы()

	func (т Таблица  )Вставить(){

	}

		// ошибка := т.база.Транзакция(изменить, func(трз *Транзакция) ОшибкаБазы {
		// 	трз.Вставить(данные map[string][]byte, заменитьУникальныйКлюч bool)
		// })
		// или
		// ошибка := т.база.Транзакция(изменить, func(трз *Транзакция) ОшибкаБазы {

		// 	трз.Вставить(данные map[string][]byte, заменитьУникальныйКлюч bool)
		// })

*/

/*

ВСТАВИТЬ не открывет подклчючений к базе и транзакцию, это должно быть создано до вызова функции Всавить
Также до вызова функции должны быть получены индексы таблицы, и записаны в структуру в соответствующие поля
Если поля индексы пустые, то функция считает что индексов нет, и просто делает за  ись в базу данных
*/
// func (т *Таблица) Вставить (данные *map[string]interface{}, заменитьУникальныйКлюч bool) {

//   /*
//   1. прооверяем существует ли запись с ключём документа:
//     если ключ свободен то записывааем данные в базу но не закрываем транзакцию

//   2. Исппользуем
//   */

//		if т.индексы != nil {
//			индексы, ошибка := СоздатьЗначенияИндексов(данные, &т.индексы, ложь)
//			if ошибка != nil {
//				Ошибка("  %+v \n", ошибка)
//			}
//		}
//		if т.униикальныеИндексы != nil {
//			униикальныеИндексы, ошибка := СоздатьЗначенияИндексов(данные, &т.униикальныеИндексы, истина)
//			if ошибка != nil {
//				Ошибка("  %+v \n", ошибка)
//			}
//		}
//	}
type Таблица struct {
	Имя               string
	Дирректория       string
	ПервичныйКлюч     ПервичныйКлюч // путь к значению в документе которое используется для генерации ключа документа, во всех документах однойтаблицы все первичные ключи должны .snm подобны : имяТаблицы.первичныйКлюч:значениеб
	индексы           ТаблицаИндексов
	уникальныеИндексы ТаблицаИндексов
	// индексы            map[ПутьИндекса]struct{}
	// униикальныеИндексы map[ПутьИндекса]struct{}
	база База
	трз  Транзакция
	// базаИндексов       *БазаИднексов
}

func (таблица *Таблица) Подключиться(путь string) ОшибкаБазы {
	if путь == "" {
		путь = таблица.Дирректория + "." + таблица.Имя
	}
	база, err := badger.Open(badger.DefaultOptions(путь))
	if err != nil {
		Ошибка("  %+v \n", err)
		return ОшибкаБазы{
			Код:   ОшибкаПодключения,
			Текст: err.Error(),
		}
	}
	таблица.база = База{база}
	return ОшибкаБазы{
		Код:   0,
		Текст: "успешное подключение",
	}
}

/*


 */

/*
Созздаёт покдлючения к базе данныхи и базе индексов , запускает горутину которя возвращает канал, из которого она читает данные для записис в таблицу,
так же создаёт аналогичные каналы для индексов
*/
func (таблица *Таблица) АктивироватьТаблицу() ОшибкаБазы {

	ошибка := таблица.Подключиться("") // открываем соединение к базе данных
	if ошибка.Код != 0 {
		return ошибка
	}
	ошибка = таблица.ПроверитьСоздатьИндексы()
	if ошибка.Код != 0 {
		return ошибка
	}
	return ОшибкаБазы{
		Код:   Ок,
		Текст: "успешно",
	}
}
