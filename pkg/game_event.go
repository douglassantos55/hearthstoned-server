package pkg

import (
	"container/list"
	"sync"
	"time"
)

type GameEventType = string

const (
	MinionDamagedEvent   GameEventType = "minion_damaged"
	MinionDestroyedEvent GameEventType = "minion_destroyed"
	CardPlayedEvent      GameEventType = "card_played"
	TurnStartedEvent     GameEventType = "turn_started"
	ManaGainedEvent      GameEventType = "mana_gained"
	DamageIncreasedEvent GameEventType = "damage_increased"
	PlayerDamagedEvent   GameEventType = "player_damaged"
	StateChangedEvent    GameEventType = "minion_state_changed"
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

	if listeners := d.listeners[event.GetType()]; listeners != nil {
		for cur := listeners.Front(); cur != nil; cur = cur.Next() {
			listener := cur.Value.(Listener)
			if listener(event) {
				listeners.Remove(cur)
			}
		}
	}
}

type GameEvent interface {
	GetData() interface{}
	GetType() GameEventType
}

type DamageEvent struct {
	defender *ActiveMinion
	attacker *ActiveMinion
}

func NewDamageEvent(defender *ActiveMinion, attacker *ActiveMinion) DamageEvent {
	return DamageEvent{
		defender: defender,
		attacker: attacker,
	}
}

func (d DamageEvent) GetData() interface{} {
	return MinionDamagedPayload{
		Attacker: d.attacker,
		Defender: d.defender,
	}
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
	player   *Player
	duration time.Duration
}

func NewTurnStartedEvent(player *Player, duration time.Duration) TurnStarted {
	return TurnStarted{
		player:   player,
		duration: duration,
	}
}

func (t TurnStarted) GetData() interface{} {
	return map[string]interface{}{
		"Player":   t.player,
		"Duration": t.duration,
	}
}

func (t TurnStarted) GetType() GameEventType {
	return TurnStartedEvent
}

type Trigger struct {
	Event     GameEventType
	Condition func(card Card, event GameEvent) bool // Determines whether this trigger should be activated
}

type ManaGained struct {
	Player *Player
}

func (m ManaGained) GetData() interface{} {
	return m.Player
}

func (m ManaGained) GetType() GameEventType {
	return ManaGainedEvent
}

type DamageIncreased struct {
	Minion *ActiveMinion
}

func (d DamageIncreased) GetData() interface{} {
	return d.Minion
}

func (m DamageIncreased) GetType() GameEventType {
	return DamageIncreasedEvent
}

type PlayerDamaged struct {
	player   *Player
	attacker *ActiveMinion
}

func NewPlayerDamagedEvent(player *Player, attacker *ActiveMinion) PlayerDamaged {
	return PlayerDamaged{
		player:   player,
		attacker: attacker,
	}
}

func (d PlayerDamaged) GetData() interface{} {
	return PlayerDamagedPayload{
		Player:   d.player,
		Attacker: d.attacker,
	}
}

func (m PlayerDamaged) GetType() GameEventType {
	return PlayerDamagedEvent
}

type StateChanged struct {
	minion *ActiveMinion
}

func NewStateChangedEvent(minion *ActiveMinion) StateChanged {
	return StateChanged{
		minion: minion,
	}
}

func (s StateChanged) GetData() interface{} {
	return s.minion
}

func (s StateChanged) GetType() GameEventType {
	return StateChangedEvent
}
