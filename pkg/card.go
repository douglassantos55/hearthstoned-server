package pkg

import (
	"container/list"
	"sync"

	"github.com/google/uuid"
)

type Card struct {
	Id uuid.UUID

	Mana   int
	Damage int
	Health int
}

func NewCard(mana, damage, health int) *Card {
	return &Card{
		Id: uuid.New(),

		Mana:   mana,
		Damage: damage,
		Health: health,
	}
}

func (c *Card) GetMana() int {
	return c.Mana
}

type Hand struct {
	cards map[uuid.UUID]*Card
	mutex *sync.Mutex
}

func NewHand(items *list.List) *Hand {
	cards := map[uuid.UUID]*Card{}
	for cur := items.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(*Card)
		cards[card.Id] = card
	}
	return &Hand{
		cards: cards,
		mutex: new(sync.Mutex),
	}
}

func (h *Hand) Get(index int) *Card {
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

func (h *Hand) Add(card *Card) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.cards[card.Id] = card
}

func (h *Hand) Find(cardId uuid.UUID) *Card {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.cards[cardId]
}

func (h *Hand) Remove(card *Card) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.cards, card.Id)
}

func (h *Hand) Length() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return len(h.cards)
}

type MinionState interface {
	CanAttack() bool
}

type Active struct{}

func (e Active) CanAttack() bool {
	return true
}

type Exhausted struct{}

func (e Exhausted) CanAttack() bool {
	return false
}

// Minion represents a placed card on board, giving it the
// ability to attack and states
type Minion struct {
	*Card
	state MinionState
}

func NewMinion(card *Card) *Minion {
	return &Minion{
		Card:  card,
		state: Exhausted{},
	}
}

func (m *Minion) GetState() MinionState {
	return m.state
}

// Reduces minion health and returns wether it survives or not
func (m *Minion) RemoveHealth(amount int) bool {
	m.Health -= amount
	return m.Health > 0
}
