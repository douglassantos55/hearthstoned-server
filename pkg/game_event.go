package pkg

import (
	"container/list"
	"sync"
)

type GameEventType = string

const (
	MinionDamagedEvent   GameEventType = "minion_damaged"
	MinionDestroyedEvent GameEventType = "minion_destroyed"
	CardPlayedEvent      GameEventType = "card_played"
	TurnStartedEvent     GameEventType = "turn_started"
)

// Listener takes an event and returns true if it should be removed after
// executing or false if it should remain and be executed multiple times
type Listener = func(event GameEvent) bool

type Dispatcher interface {
	Dispatch(event GameEvent)
	Subscribe(event GameEventType, listener Listener)
}

type GameDispatcher struct {
	mutex     *sync.Mutex
	listeners map[GameEventType]*list.List
}

func NewGameDispatcher() *GameDispatcher {
	return &GameDispatcher{
		mutex:     new(sync.Mutex),
		listeners: make(map[GameEventType]*list.List),
	}
}

func (d *GameDispatcher) Subscribe(event GameEventType, listener Listener) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if _, ok := d.listeners[event]; !ok {
		d.listeners[event] = list.New()
	}
	d.listeners[event].PushBack(listener)
}

func (d *GameDispatcher) Dispatch(event GameEvent) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	listeners := d.listeners[event.GetType()]
	for cur := listeners.Front(); cur != nil; cur = cur.Next() {
		listener := cur.Value.(Listener)
		if listener(event) {
			listeners.Remove(cur)
		}
	}
}

type GameEvent interface {
	GetData() interface{}
	GetType() GameEventType
}

type DamageEvent struct {
	minion *ActiveMinion
}

func NewDamageEvent(minion *ActiveMinion) DamageEvent {
	return DamageEvent{
		minion,
	}
}

func (d DamageEvent) GetData() interface{} {
	return d.minion
}

func (d DamageEvent) GetType() GameEventType {
	return MinionDamagedEvent
}

type DestroyedEvent struct {
	minion *ActiveMinion
}

func NewDestroyedEvent(minion *ActiveMinion) DestroyedEvent {
	return DestroyedEvent{
		minion,
	}
}

func (d DestroyedEvent) GetData() interface{} {
	return d.minion
}

func (d DestroyedEvent) GetType() GameEventType {
	return MinionDestroyedEvent
}

type CardPlacedEvent struct {
	card Card
}

func NewCardPlayedEvent(card Card) CardPlacedEvent {
	return CardPlacedEvent{
		card,
	}
}

func (c CardPlacedEvent) GetData() interface{} {
	return c.card
}

func (c CardPlacedEvent) GetType() GameEventType {
	return CardPlayedEvent
}

type TurnStarted struct {
	player *Player
}

func NewTurnStartedEvent(player *Player) TurnStarted {
	return TurnStarted{
		player: player,
	}
}

func (t TurnStarted) GetData() interface{} {
	return t.player
}

func (t TurnStarted) GetType() GameEventType {
	return TurnStartedEvent
}

type Trigger struct {
	Event     GameEventType
	Condition func(card Card, event GameEvent) bool // Determines whether this trigger should be activated
}
