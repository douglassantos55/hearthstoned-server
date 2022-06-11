package pkg

import (
	"container/list"
	"errors"
	"sync"

	"github.com/google/uuid"
)

const MAX_MANA = 10
const MAX_MINIONS = 7

type Player struct {
	currMana  int
	totalMana int

	mutex  *sync.Mutex
	board  *Board
	hand   *Hand
	deck   *Deck
	socket *Socket
}

func NewPlayer(socket *Socket) *Player {
	return &Player{
		totalMana: 0,
		currMana:  0,

		mutex:  new(sync.Mutex),
		board:  NewBoard(),
		deck:   NewDeck(),
		socket: socket,
		hand:   NewHand(list.New()),
	}
}

func (p *Player) DrawCards(qty int) []*Minion {
	out := []*Minion{}
	cards := p.deck.Draw(qty)

	for cur := cards.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(*Minion)
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

func (p *Player) Discard(cardId uuid.UUID) *Minion {
	card := p.hand.Find(cardId)
	if card != nil {
		p.hand.Remove(card)
	}
	return card
}

func (p *Player) AddToDeck(card *Minion) {
	p.deck.Push(card)
}

func (p *Player) RefillMana() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.currMana = p.totalMana
}

func (p *Player) AddMana(qty int) {
	p.currMana += qty
	if p.currMana > p.totalMana {
		p.currMana = p.totalMana
	}
}

func (p *Player) GainMana(qty int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.totalMana += qty
	if p.totalMana > MAX_MANA {
		p.totalMana = MAX_MANA
	}
}

func (p *Player) GetMana() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.currMana
}

func (p *Player) PlayCard(card Card) error {
	// reduce player's current mana
	p.currMana -= card.GetMana()

	// add card to player's board
	if minion, ok := card.(*Minion); ok {
		err := p.board.Place(minion)
		if err != nil {
			return err
		}
	}

	if spell, ok := card.(*Spell); ok {
		spell.Cast()
	}

	return nil
}

func (p *Player) CardsOnBoardCount() int {
	return p.board.MinionsCount()
}

func (p *Player) NotifyDamage(event GameEvent) {
	minion := event.GetData().(*ActiveMinion)
	go p.Send(DamageTaken(minion.Minion))
}

func (p *Player) NotifyDestroyed(event GameEvent) {
	minion := event.GetData().(*ActiveMinion)
	go p.Send(MinionDestroyedMessage(minion.Minion))
}

func (p *Player) NotifyCardPlayed(event GameEvent) {
	card := event.GetData().(*Minion)
	go p.Send(Response{
		Type:    CardPlayed,
		Payload: card,
	})
}

func (p *Player) NotifyTurnStarted(event GameEvent) {
	player := event.GetData().(*Player)
	cards := []*Minion{}
	for _, card := range player.hand.cards {
		cards = append(cards, card)
	}

	if player == p {
		go p.Send(Response{
			Type: StartTurn,
			Payload: TurnPayload{
				Cards:       cards,
				Mana:        player.GetMana(),
				CardsInHand: player.hand.Length(),
			},
		})
	} else {
		go p.Send(Response{
			Type: WaitTurn,
			Payload: TurnPayload{
				Mana:        player.GetMana(),
				CardsInHand: player.hand.Length(),
			},
		})
	}
}

type Board struct {
	minions map[uuid.UUID]*ActiveMinion
}

func NewBoard() *Board {
	return &Board{
		minions: make(map[uuid.UUID]*ActiveMinion),
	}
}

func (b *Board) MinionsCount() int {
	return len(b.minions)
}

func (b *Board) GetMinion(minionId uuid.UUID) (*ActiveMinion, bool) {
	minion, ok := b.minions[minionId]
	return minion, ok
}

func (b *Board) Remove(minion *ActiveMinion) {
	delete(b.minions, minion.Id)
}

func (b *Board) Place(card *Minion) error {
	if b.MinionsCount() == MAX_MINIONS {
		return errors.New("Cannot place minion, board is full")
	}
	b.minions[card.Id] = NewMinion(card)
	return nil
}

func (b *Board) ActivateAll() {
	for _, minion := range b.minions {
		minion.SetState(Active{})
	}
}
