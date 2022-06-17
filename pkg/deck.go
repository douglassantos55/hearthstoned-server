package pkg

import (
	"container/list"
	"math/rand"
	"sync"
	"time"
)

const MAX_CARDS = 120
const MAX_DECK_SIZE = 60

type Deck struct {
	mutex *sync.Mutex
	cards *list.List
}

func NewDeck() *Deck {
	cards := list.New()
	seen := make(map[int]bool)

	rand.Seed(time.Now().UnixNano())

	for cards.Len() != MAX_DECK_SIZE {
		idx := rand.Intn(MAX_CARDS)
		if _, ok := seen[idx]; !ok {
			seen[idx] = true
			card := GetCards()[idx]
			cards.PushBack(card)
		}
	}
	return &Deck{
		mutex: new(sync.Mutex),
		cards: cards,
	}
}

func (d *Deck) Push(card Card) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if card != nil {
		d.cards.PushBack(card)
	}
}

func (d *Deck) Pop() Card {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.cards.Len() == 0 {
		return nil
	}
	card := d.cards.Remove(d.cards.Front())
	return card.(Card)
}

func (d *Deck) Draw(qty int) *list.List {
	if d.cards.Len() == 0 {
		return nil
	}
	cards := list.New()
	for i := 0; i < qty; i++ {
		card := d.Pop()
		cards.PushBack(card)
	}
	return cards
}
