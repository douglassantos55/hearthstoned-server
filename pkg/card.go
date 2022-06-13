package pkg

import (
	"container/list"
	"sync"

	"github.com/google/uuid"
)

type Card interface {
	GetId() uuid.UUID
	GetMana() int
}

type Ability interface {
	Cast()
}

type TriggerableSpell struct {
	*Spell
	event GameEventType
}

func Trigerable(event GameEventType, spell *Spell) *TriggerableSpell {
	return &TriggerableSpell{
		Spell: spell,
		event: event,
	}
}

type GainMana struct {
	amount int
	player *Player
}

func GainManaAbility(amount int, player *Player) GainMana {
	return GainMana{
		amount: amount,
		player: player,
	}
}

func (g GainMana) Cast() {
	g.player.GainMana(g.amount)
}

type Spell struct {
	Id      uuid.UUID
	Mana    int
	Ability Ability
}

func NewSpell(mana int, ability Ability) *Spell {
	return &Spell{
		Id:      uuid.New(),
		Mana:    mana,
		Ability: ability,
	}
}

func (s *Spell) GetId() uuid.UUID {
	return s.Id
}

func (s *Spell) GetMana() int {
	return s.Mana
}

func (s *Spell) Execute() {
	s.Ability.Cast()
}

type Minion struct {
	Id uuid.UUID

	Mana   int
	Damage int
	Health int
}

func NewCard(mana, damage, health int) *Minion {
	return &Minion{
		Id: uuid.New(),

		Mana:   mana,
		Damage: damage,
		Health: health,
	}
}

func (c *Minion) GetId() uuid.UUID {
	return c.Id
}

func (c *Minion) GetMana() int {
	return c.Mana
}

type Hand struct {
	cards map[uuid.UUID]Card
	mutex *sync.Mutex
}

func NewHand(items *list.List) *Hand {
	cards := map[uuid.UUID]Card{}
	for cur := items.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(*Minion)
		cards[card.Id] = card
	}
	return &Hand{
		cards: cards,
		mutex: new(sync.Mutex),
	}
}

func (h *Hand) Get(index int) Card {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	counter := 0
	for _, card := range h.cards {
		if counter == index {
			return card
		}
		counter++
	}
	return nil
}

func (h *Hand) Add(card Card) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.cards[card.GetId()] = card
}

func (h *Hand) GetCards() []Card {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	cards := []Card{}
	for _, card := range h.cards {
		cards = append(cards, card)
	}
	return cards
}

func (h *Hand) Find(cardId uuid.UUID) Card {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.cards[cardId]
}

func (h *Hand) Remove(card Card) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.cards, card.GetId())
}

func (h *Hand) Length() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return len(h.cards)
}

type MinionState interface {
	CanAttack() bool
	CanCounterAttack() bool
}

type Active struct{}

func (e Active) CanAttack() bool {
	return true
}

func (e Active) CanCounterAttack() bool {
	return true
}

type Exhausted struct{}

func (e Exhausted) CanAttack() bool {
	return false
}

func (e Exhausted) CanCounterAttack() bool {
	return true
}

// ActiveMinion represents a placed card on board, giving it the
// ability to attack and states
type ActiveMinion struct {
	*Minion
	state MinionState
}

func NewMinion(card *Minion) *ActiveMinion {
	return &ActiveMinion{
		Minion: card,
		state:  Exhausted{},
	}
}

func (m *ActiveMinion) CanAttack() bool {
	return m.state.CanAttack()
}

func (m *ActiveMinion) CanCounterAttack() bool {
	return m.state.CanCounterAttack()
}

func (m *ActiveMinion) GetState() MinionState {
	return m.state
}

func (m *ActiveMinion) SetState(state MinionState) {
	m.state = state
}

// Reduces minion health and returns wether it survives or not
func (m *ActiveMinion) RemoveHealth(amount int) bool {
	m.Health -= amount
	return m.Health > 0
}
