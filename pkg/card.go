package pkg

import (
	"container/list"

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

type Cards struct {
	cards []*Card
}

func (c *Cards) Add(card *Card) {
	if card != nil {
		c.cards = append(c.cards, card)
	}
}

func (c *Cards) Length() int {
	return len(c.cards)
}

type Hand struct {
	cards map[uuid.UUID]*Card
}

func NewHand(items *list.List) *Hand {
	cards := map[uuid.UUID]*Card{}
	for cur := items.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(*Card)
		cards[card.Id] = card
	}
	return &Hand{cards: cards}
}

func (h *Hand) Get(index int) *Card {
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
	h.cards[card.Id] = card
}

func (h *Hand) Find(cardId uuid.UUID) *Card {
	return h.cards[cardId]
}

func (h *Hand) Remove(card *Card) {
	delete(h.cards, card.Id)
}

func (h *Hand) Length() int {
	return len(h.cards)
}
