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

func (p *Player) DrawCards(qty int) []Card {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	out := []Card{}
	cards := p.deck.Draw(qty)
	for cur := cards.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(Card)
		p.hand.Add(card)
		out = append(out, card)
	}
	return out
}

func (p *Player) Send(message Response) {
	p.socket.Send(message)
}

func (p *Player) GetHand() *Hand {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.hand
}

func (p *Player) Discard(cardId uuid.UUID) Card {
	card := p.hand.Find(cardId)
	if card != nil {
		p.hand.Remove(card)
	}
	return card
}

func (p *Player) AddToDeck(card Card) {
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

func (p *Player) ReduceMana(qty int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.currMana -= qty
}

func (p *Player) GetMana() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.currMana
}

func (p *Player) GetTotalMana() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.totalMana
}

func (p *Player) PlayCard(card Card) (Card, error) {
	// reduce player's current mana
	p.ReduceMana(card.GetMana())

	// add card to player's board
	if minion, ok := card.(*Minion); ok {
		activeMinion := NewMinion(minion, p)
		err := p.board.Place(activeMinion)
		if err != nil {
			return nil, err
		}
		return activeMinion, nil
	}

	return card, nil
}

func (p *Player) CardsOnBoardCount() int {
	return p.board.MinionsCount()
}

func (p *Player) NotifyDamage(event GameEvent) bool {
	minion := event.GetData().(*ActiveMinion)
	go p.Send(DamageTaken(minion.Minion))
	return false
}

func (p *Player) NotifyDestroyed(event GameEvent) bool {
	minion := event.GetData().(*ActiveMinion)
	go p.Send(MinionDestroyedMessage(minion.Minion))
	return false
}

func (p *Player) NotifyCardPlayed(event GameEvent) bool {
	card := event.GetData().(Card)
	go p.Send(Response{
		Type:    CardPlayed,
		Payload: card,
	})
	return false
}

func (p *Player) NotifyManaChanges(event GameEvent) bool {
	player := event.GetData().(*Player)
	go p.Send(Response{
		Type:    ManaChanged,
		Payload: player,
	})
	return false
}

func (p *Player) NotifyAttributeChanges(event GameEvent) bool {
	minion := event.GetData().(*Minion)
	go p.Send(Response{
		Type:    AttributeChanged,
		Payload: minion,
	})
	return false
}

func (p *Player) NotifyTurnStarted(event GameEvent) bool {
	player := event.GetData().(*Player)

	if player == p {
		go p.Send(Response{
			Type: StartTurn,
			Payload: TurnPayload{
				Cards:       player.hand.GetCards(),
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
	return false
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

func (b *Board) Place(card *ActiveMinion) error {
	if b.MinionsCount() == MAX_MINIONS {
		return errors.New("Cannot place minion, board is full")
	}
	b.minions[card.Id] = card
	return nil
}

func (b *Board) ActivateAll() {
	for _, minion := range b.minions {
		minion.SetState(Active{})
	}
}
