package ConnQuic

import (
	"errors"
	"sync"

	quic "github.com/quic-go/quic-go"
)

var БлокКартаСервисов sync.RWMutex

type КартаСервисов map[ИмяСервиса][]КартаСессий

type КартаСессий struct {
	*sync.RWMutex
	Соединение     quic.Connection
	ОчередьПотоков *ОчередьПотоков
	СистемныйПоток quic.Stream
}

// --- Очередь для работы с потоками и сессиями quic
var БлокКартаСервисов_ = sync.RWMutex{}

type КартаСервисов_ map[ИмяСервиса]*КартаСессий

type Сессия_ struct {
	// Блок       sync.RWMutex
	Соединение quic.Connection
	Потоки     []quic.Stream
}
type КартаСессий_ struct {
	*sync.RWMutex
	СессииСервисов []Сессия_       // кладём соовтетсвие сессий и потоков
	ОчередьПотоков *ОчередьПотоков // все потоки всех сессий кладём в одну очередь
}

type Поток struct {
	поток     quic.Stream
	следующий *Поток
}

type ОчередьПотоков struct {
	*sync.RWMutex
	Первый     *Поток
	Последний  *Поток
	Количество int
}

func НоваяОчередьПотоков() *ОчередьПотоков {
	return &ОчередьПотоков{}
}

func (очередь *ОчередьПотоков) Добавить(новыйПоток quic.Stream) {
	очередь.Вернуть(новыйПоток)
}
func (очередь *ОчередьПотоков) Вернуть(новыйПоток quic.Stream) {
	очередь.RLock()
	поток := &Поток{поток: новыйПоток}
	if очередь.Последний == nil {
		очередь.Первый = поток
		очередь.Последний = поток
	} else {
		очередь.Последний.следующий = поток
		очередь.Последний = поток
	}
	очередь.Количество++
	очередь.RUnlock()
}

func (очередь *ОчередьПотоков) Взять() quic.Stream {
	очередь.RLock()
	defer очередь.RUnlock()
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

// --- Очередь для любого типа данных

type Узел struct {
	значение  interface{}
	следующий *Узел
}

type Очередь struct {
	sync.RWMutex
	Первый    *Узел
	Последний *Узел
}

func НоваяОчередь() *Очередь {
	return &Очередь{}
}

func (очередь *Очередь) Добавить(новыйУзел interface{}) {
	очередь.RLock()
	defer очередь.RUnlock()
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
	очередь.RLock()
	defer очередь.RUnlock()
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
