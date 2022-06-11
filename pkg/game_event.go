package pkg

type GameEventType = int

const (
	MinionDamagedEvent GameEventType = iota
	MinionDestroyedEvent
	CardPlayedEvent
)

type Listener = func(event GameEvent)

type Dispatcher interface {
	Dispatch(event GameEvent)
	Subscribe(event GameEventType, listener Listener)
}

type GameDispatcher struct {
	listeners map[GameEventType][]Listener
}

func NewGameDispatcher() *GameDispatcher {
	return &GameDispatcher{
		listeners: make(map[GameEventType][]Listener),
	}
}

func (d *GameDispatcher) Subscribe(event GameEventType, listener Listener) {
	if _, ok := d.listeners[event]; !ok {
		d.listeners[event] = make([]Listener, 0)
	}
	d.listeners[event] = append(d.listeners[event], listener)
}

func (d *GameDispatcher) Dispatch(event GameEvent) {
	for _, listener := range d.listeners[event.GetType()] {
		listener(event)
	}
}

type GameEvent interface {
	GetData() interface{}
	GetType() GameEventType
}

type DamageEvent struct {
	minion *Minion
}

func NewDamageEvent(minion *Minion) DamageEvent {
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
	minion *Minion
}

func NewDestroyedEvent(minion *Minion) DestroyedEvent {
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
	card *Card
}

func NewCardPlayedEvent(card *Card) CardPlacedEvent {
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
