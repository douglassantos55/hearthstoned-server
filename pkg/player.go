package pkg

import (
	"container/list"

	"github.com/google/uuid"
)

const MAX_MANA = 10

type Player struct {
	mana   int
	hand   *Hand
	deck   *Deck
	socket *Socket
}

func NewPlayer(socket *Socket) *Player {
	return &Player{
		mana:   0,
		deck:   NewDeck(),
		socket: socket,
		hand:   NewHand(list.New()),
	}
}

func (p *Player) DrawCards(qty int) {
	cards := p.deck.Draw(qty)
	for cur := cards.Front(); cur != nil; cur = cur.Next() {
		p.hand.Add(cur.Value.(*Card))
	}
}

func (p *Player) Send(message Response) {
	p.socket.Send(message)
}

func (p *Player) GetHand() *Hand {
	return p.hand
}

func (p *Player) Discard(cardId uuid.UUID) *Card {
	card := p.hand.Find(cardId)
	if card != nil {
		p.hand.Remove(card)
	}
	return card
}

func (p *Player) AddToDeck(card *Card) {
	p.deck.Push(card)
}

func (p *Player) GainMana(qty int) {
	p.mana += qty
	if p.mana > MAX_MANA {
		p.mana = MAX_MANA
	}
}

func (p *Player) GetMana() int {
	return p.mana
}
