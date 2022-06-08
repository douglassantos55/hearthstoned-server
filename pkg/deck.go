package pkg

import (
	"container/list"
	"math/rand"
	"time"
)

type Deck struct {
	cards *list.List
}

func NewDeck() *Deck {
	cards := list.New()
	rand.Seed(time.Now().UnixNano())
	// create 60 random cards
	for i := 0; i < 60; i++ {
		damage := rand.Intn(10)
		health := rand.Intn(10)
		cards.PushBack(NewCard(damage, health))
	}
	return &Deck{cards: cards}
}

func (d *Deck) Push(card *Card) {
	if card != nil {
		d.cards.PushBack(card)
	}
}

func (d *Deck) Pop() *Card {
	if d.cards.Len() == 0 {
		return nil
	}
	card := d.cards.Remove(d.cards.Front())
	return card.(*Card)
}

func (d *Deck) Draw(qty int) *list.List {
	if d.cards.Len() == 0 {
		return nil
	}
	cards := list.New()
	for i := 0; i < qty; i++ {
		cards.PushBack(d.Pop())
	}
	return cards
}
