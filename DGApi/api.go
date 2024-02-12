package DGApi

import (
	"context"
	"log"
	"strings"

	. "aoanima.ru/Logger"
	. "aoanima.ru/QErrors"

	dgo "github.com/dgraph-io/dgo/v230"
	"github.com/dgraph-io/dgo/v230/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// https://github.com/dgraph-io/dgo/blob/master/example_set_object_test.go

type ЗакрытьСоединение func()
type КаналДанных struct {
	КаналОтвет chan string
	Ошибка     chan string
	ДанныеЗапроса
}
type ДанныеЗапроса struct {
	Запрос string
	Данные map[string]string
}
type КлиентДГраф *dgo.Dgraph

// func ДГраф(каналДанных chan КаналДанны/х) {
func ДГраф() (*dgo.Dgraph, ЗакрытьСоединение) {
	// Dial a gRPC connection. The address to dial to can be configured when
	// setting up the dgraph cluster.
	связь, err := grpc.Dial("localhost:9080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	dc := api.NewDgraphClient(связь)
	граф := dgo.NewDgraphClient(dc)
	// ctx := context.Background()

	// for данные := range каналДанных {
	// 	данные := данные
	// 	go func(граф *dgo.Dgraph, данные КаналДанных) {

	// 		ctx := context.Background()

	// 		мутация := &api.Mutation{
	// 			CommitNow: true,
	// 		}
	// 		// pb, err := json.Marshal(p)
	// 		// if err != nil {
	// 		// 	log.Fatal(err)
	// 		// }

	// 		мутация.SetJson = []byte(данные.Запрос)
	// 		результат, ошибка := граф.NewTxn().Mutate(ctx, мутация)

	// 		if ошибка != nil {
	// 			данные.Ошибка <- ошибка.Error()
	// 		} else {
	// 			данные.КаналОтвет <- результат.String()
	// 		}
	// 		return
	// 	}(граф, данные)
	// }
	// Инфо(" канал закрылся, цикл прервался %+v \n", каналДанных)
	// Авторизация, пока пропустим
	// Perform login call. If the Dgraph cluster does not have ACL and
	// enterprise features enabled, this call should be skipped.
	// for {
	// 	// Keep retrying until we succeed or receive a non-retriable error.
	// 	err = dg.Login(ctx, "groot", "password")
	// 	if err == nil || !strings.Contains(err.Error(), "Please retry") {
	// 		break
	// 	}
	// 	time.Sleep(time.Second)
	// }
	// if err != nil {
	// 	log.Fatalf("While trying to login %v", err.Error())
	// }
	// if err := связь.Close(); err != nil {
	// 	Ошибка(" Ошибка закрытия соединения %+v \n", err)
	// }
	// resp, err := граф.NewTxn().QueryWithVars(ctx, q, variables)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	return граф, func() {
		if err := связь.Close(); err != nil {
			Ошибка(" Ошибка закрытия соединения %+v \n", err)
		}
	}
}

/*
Изменить открывает транзакцию на изменение, принимает запрос на измнение, и данные для подстановки
Передавать можно любые запросы на вставку, и чтение.
Берёт соединнение из пула, отправляет запрос и возвращает соелинение в пул
*/
// func Изменить(запрос ДанныеЗапроса, граф *dgo.Dgraph) (string, СтатусСервиса) {
// 	ctx := context.Background()

// 	мутация := &api.Mutation{
// 		CommitNow: true,
// 	}
// 	мутация.SetJson = []byte(запрос.Запрос)
// 	результат, ошибка := граф.NewTxn().Mutate(ctx, мутация)

//		if ошибка != nil {
//			return результат.String(), СтатусСервиса{
//				Код:   ОшибкаЗаписи,
//				Текст: ошибка.Error(),
//			}
//		}
//		return результат.String(), СтатусСервиса{
//			Код: Ок,
//		}
//	}
func Изменить(запрос ДанныеЗапроса, граф *dgo.Dgraph) (string, СтатусСервиса) {
	for {
		ctx := context.Background()
		транзакция := граф.NewTxn()
		defer транзакция.Discard(ctx)

		мутация := &api.Mutation{
			CommitNow: true,
		}
		мутация.SetJson = []byte(запрос.Запрос)
		результат, ошибка := транзакция.Mutate(ctx, мутация)

		if ошибка != nil {
			if strings.Contains(ошибка.Error(), "conflict") {
				// Конфликт транзакции, повторяем
				Инфо(" Конфликт транзакции, повторяем %+v \n", ошибка.Error())
				continue
			}
			return "", СтатусСервиса{
				Код:   ОшибкаЗаписи,
				Текст: ошибка.Error(),
			}
		}
		// не делаем комит вручную, так как установлен флаг CommitNow
		// ошибка = транзакция.Commit(ctx)
		// if ошибка != nil {
		// 	if strings.Contains(ошибка.Error(), "conflict") {
		// 		// Конфликт транзакции, повторяем
		// 		continue
		// 	}
		// 	return "", СтатусСервиса{
		// 		Код:   ОшибкаЗаписи,
		// 		Текст: ошибка.Error(),
		// 	}
		// }

		return результат.String(), СтатусСервиса{
			Код: Ок,
		}
	}
}

/*
Поллучить открывает транзакцию на выборку данных, отправляет запрос, возвращает результат в  виде json строки
Берёт соединнение из пула, отправляет запрос и возвращает соелинение в пул
*/
func Получить(Запрос string) string {

	return "резултать запроса"
}
