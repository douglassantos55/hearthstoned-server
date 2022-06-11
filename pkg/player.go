package pkg

import (
	"container/list"
	"errors"

	"github.com/google/uuid"
)

const MAX_MANA = 10
const MAX_MINIONS = 7

type Player struct {
	currMana  int
	totalMana int

	board  *Board
	hand   *Hand
	deck   *Deck
	socket *Socket
}

func NewPlayer(socket *Socket) *Player {
	return &Player{
		totalMana: 0,
		currMana:  0,

		board:  NewBoard(),
		deck:   NewDeck(),
		socket: socket,
		hand:   NewHand(list.New()),
	}
}

func (p *Player) DrawCards(qty int) []*Card {
	out := []*Card{}
	cards := p.deck.Draw(qty)

	for cur := cards.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(*Card)
		p.hand.Add(card)
		out = append(out, card)
	}

	return out
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

func (p *Player) RefillMana() {
	p.currMana = p.totalMana
}

func (p *Player) GainMana(qty int) {
	p.totalMana += qty
	if p.totalMana > MAX_MANA {
		p.totalMana = MAX_MANA
	}
}

func (p *Player) GetMana() int {
	return p.currMana
}

func (p *Player) PlayCard(card *Card) error {
	// reduce player's current mana
	p.currMana -= card.GetMana()

	// add card to player's board
	err := p.board.Place(card)
	if err != nil {
		return err
	}

	return nil
}

func (p *Player) CardsOnBoardCount() int {
	return p.board.MinionsCount()
}

func (p *Player) NotifyDamage(event GameEvent) {
	minion := event.GetData().(*Minion)
	go p.Send(DamageTaken(minion.Card))
}

func (p *Player) NotifyDestroyed(event GameEvent) {
	minion := event.GetData().(*Minion)
	go p.Send(MinionDestroyedMessage(minion.Card))
}

func (p *Player) NotifyCardPlayed(event GameEvent) {
	card := event.GetData().(*Card)
	go p.Send(Response{
		Type:    CardPlayed,
		Payload: card,
	})
}

type Board struct {
	minions map[uuid.UUID]*Minion
}

func NewBoard() *Board {
	return &Board{
		minions: make(map[uuid.UUID]*Minion),
	}
}

func (b *Board) MinionsCount() int {
	return len(b.minions)
}

func (b *Board) GetMinion(minionId uuid.UUID) (*Minion, bool) {
	minion, ok := b.minions[minionId]
	return minion, ok
}

func (b *Board) Remove(minion *Minion) {
	delete(b.minions, minion.Id)
}

func (b *Board) Place(card *Card) error {
	if b.MinionsCount() == MAX_MINIONS {
		return errors.New("Cannot place minion, board is full")
	}
	b.minions[card.Id] = NewMinion(card)
	return nil
}
