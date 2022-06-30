package pkg

import (
	"container/list"
	"sync"

	"github.com/google/uuid"
)

type Hand struct {
	cards map[uuid.UUID]Card
	mutex *sync.Mutex
}

func NewHand(items *list.List) *Hand {
	cards := map[uuid.UUID]Card{}
	for cur := items.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(Card)
		cards[card.GetId()] = card
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
