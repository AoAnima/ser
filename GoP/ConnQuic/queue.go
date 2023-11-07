package ConnQuic

import (
	"errors"
	"sync"

	. "aoanima.ru/logger"
	quic "github.com/quic-go/quic-go"
)

type ОчередьПотоковКанал struct {
	Потоки chan quic.Stream
}

func НоваяОчередьПотоковКанал(размер int) *ОчередьПотоковКанал {
	return &ОчередьПотоковКанал{
		Потоки: make(chan quic.Stream, размер),
	}
}
func (о *ОчередьПотоковКанал) Взять() (quic.Stream, error) {
	select {
	case поток := <-о.Потоки:
		return поток, nil
	default:
		return nil, errors.New("Нет свободных потоков")
	}

}

func (о *ОчередьПотоковКанал) Вернуть(поток quic.Stream) {
	select {
	case о.Потоки <- поток:
	default:
		// Если канал полон, просто закрываем поток
		// поток.Close()
	}
}

// --- Очередь для работы с потоками и сессиями quic
var Блок = sync.RWMutex{}

type КартаСервисов map[Сервис]*КартаСессий

type Сессия struct {
	// Блок       sync.RWMutex
	Соединение quic.Connection
	Потоки     []quic.Stream
}
type КартаСессий struct {
	Блок           sync.RWMutex
	СессииСервисов []Сессия        // кладём соовтетсвие сессий и потоков
	ОчередьПотоков *ОчередьПотоков // все потоки всех сессий кладём в одну очередь
}

type Поток struct {
	поток     *quic.Stream
	следующий *Поток
}

type ОчередьПотоков struct {
	Блок       sync.RWMutex
	Первый     *Поток
	Последний  *Поток
	Количество int
}

func НоваяОчередьПотоков() *ОчередьПотоков {
	return &ОчередьПотоков{}
}

func (очередь *ОчередьПотоков) Добавить(новыйПоток *quic.Stream) {
	очередь.Вернуть(новыйПоток)
}
<<<<<<< HEAD
func (очередь *ОчередьПотоков) Вернуть(новыйПоток quic.Stream) {
	очередь.Блок.RLock()
=======
func (очередь *ОчередьПотоков) Вернуть(новыйПоток *quic.Stream) {
>>>>>>> 749006ec09c54c1e21404de823aefea1a35f2753
	поток := &Поток{поток: новыйПоток}
	if очередь.Последний == nil {
		очередь.Первый = поток
		очередь.Последний = поток
	} else {
		очередь.Последний.следующий = поток
		очередь.Последний = поток
	}
	очередь.Количество++
	очередь.Блок.RUnlock()
}

<<<<<<< HEAD
func (очередь *ОчередьПотоков) Взять() quic.Stream {
	очередь.Блок.RLock()
	defer очередь.Блок.RUnlock()
=======
func (очередь *ОчередьПотоков) Взять() *quic.Stream {
>>>>>>> 749006ec09c54c1e21404de823aefea1a35f2753
	if очередь.Пусто() {
		return nil
	}
	очереднойЭлемент := очередь.Первый.поток
	очередь.Первый = очередь.Первый.следующий
	if очередь.Первый == nil {
		очередь.Последний = nil
	}
	очередь.Количество--
	return очереднойЭлемент
}

func (очередь *ОчередьПотоков) Пусто() bool {
	return очередь.Первый == nil
}

// --- Очередь для любого типа данных

type Узел struct {
	значение  interface{}
	следующий *Узел
}

type Очередь struct {
	Блок      sync.RWMutex
	Первый    *Узел
	Последний *Узел
}

func НоваяОчередь() *Очередь {
	return &Очередь{}
}

func (очередь *Очередь) Добавить(новыйУзел interface{}) {
	очередь.Блок.RLock()
	defer очередь.Блок.RUnlock()
	узел := &Узел{значение: новыйУзел}
	if очередь.Последний == nil {
		очередь.Первый = узел
		очередь.Последний = узел
	} else {
		очередь.Последний.следующий = узел
		очередь.Последний = узел
	}
}

func (очередь *Очередь) Далее() interface{} {
	очередь.Блок.RLock()
	defer очередь.Блок.RUnlock()
	if очередь.Пусто() {
		return nil
	}
	очереднойЭлемент := очередь.Первый.значение
	очередь.Первый = очередь.Первый.следующий
	if очередь.Первый == nil {
		очередь.Последний = nil
	}
	return очереднойЭлемент
}

func (очередь *Очередь) Пусто() bool {
	return очередь.Первый == nil
}

func (сессииСервиса *КартаСессий) СоздатьНовыйПоток() *quic.Stream {
	for _, сессия := range сессииСервиса.СессииСервисов {
		новыйПоток, err := сессия.Сессия.OpenStream()
		if err != nil {
			Ошибка(" Не удалось создать поток для открытой сессиии, нужно провреить открыта ли ещё сессия %+v \n", err)
		} else {
			МьютексАктивныхСессий.Lock()
			сессия.Потоки = append(сессия.Потоки, &новыйПоток)
			// сессииСервиса.ОчередьПотоков.Добавить(&новыйПоток)
			МьютексАктивныхСессий.Unlock()
			return &новыйПоток
		}

	}
	return nil
}

// func ПроверитьСессию (сессия quic.Connection){
// 	состояние := сессия.ConnectionState()
// 	if состояние {
// 		Ошибка("  %+v ", состояние)
// 	}
// }
