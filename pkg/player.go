package pkg

import (
	"container/list"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

const MAX_MANA = 10
const MAX_HEALTH = 30
const MAX_MINIONS = 7

type Player struct {
	Id uuid.UUID

	Health  int
	Mana    int
	MaxMana int

	mutex  *sync.Mutex
	Board  *Board
	Hand   *Hand
	deck   *Deck
	socket *Socket
}

func NewPlayer(socket *Socket) *Player {
	return &Player{
		Id: uuid.New(),

		MaxMana: 0,
		Mana:    0,
		Health:  MAX_HEALTH,

		mutex:  new(sync.Mutex),
		Board:  NewBoard(),
		deck:   NewDeck(),
		socket: socket,
		Hand:   NewHand(list.New()),
	}
}

func (p *Player) DrawCards(qty int) []Card {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	out := []Card{}
	cards := p.deck.Draw(qty)
	for cur := cards.Front(); cur != nil; cur = cur.Next() {
		card := cur.Value.(Card)
		p.Hand.Add(card)
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

	return p.Hand
}

func (p *Player) Discard(cardId uuid.UUID) Card {
	card := p.Hand.Find(cardId)
	if card != nil {
		p.Hand.Remove(card)
	}
	return card
}

func (p *Player) AddToDeck(card Card) {
	p.deck.Push(card)
}

func (p *Player) RefillMana() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Mana = p.MaxMana
}

func (p *Player) AddMana(qty int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.Mana += qty
	if p.Mana > p.MaxMana {
		p.Mana = p.MaxMana
	}
}

func (p *Player) GainMana(qty int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.MaxMana += qty
	if p.MaxMana > MAX_MANA {
		p.MaxMana = MAX_MANA
	}
}

func (p *Player) ReduceHealth(qty int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Health -= qty
}

func (p *Player) GetHealth() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.Health
}

func (p *Player) ReduceMana(qty int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Mana -= qty
}

func (p *Player) GetMana() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.Mana
}

func (p *Player) GetTotalMana() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.MaxMana
}

func (p *Player) PlayCard(card Card) (ActiveCard, error) {
	var played ActiveCard

	// reduce player's current mana
	p.ReduceMana(card.GetMana())

	// add card to player's board
	if minion, ok := card.(*Minion); ok {
		played = NewMinion(minion)
		if err := p.Board.Place(played.(*ActiveMinion)); err != nil {
			return nil, err
		}
	} else {
		played = card.(*Spell).Activate()
	}

	return played, nil
}

func (p *Player) CardsOnBoardCount() int {
	return p.Board.MinionsCount()
}

func (p *Player) NotifyDamage(event GameEvent) bool {
	payload := event.GetData().(MinionDamagedPayload)
	p.Send(Response{
		Type:    MinionDamageTaken,
		Payload: payload,
	})
	return false
}

func (p *Player) NotifyPlayerDamage(event GameEvent) bool {
	payload := event.GetData().(PlayerDamagedPayload)
	go p.Send(Response{
		Type:    PlayerDamageTaken,
		Payload: payload,
	})
	return false
}

func (p *Player) NotifyDestroyed(event GameEvent) bool {
	minion := event.GetData().(*ActiveMinion)
	go p.Send(MinionDestroyedMessage(minion))
	return false
}

func (p *Player) NotifyCardPlayed(event GameEvent) bool {
	card := event.GetData()

	if minion, ok := card.(*ActiveMinion); ok {
		card = minion
	} else if spell, ok := card.(*Spell); ok {
		card = spell
	}

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
	minion := event.GetData().(*ActiveMinion)
	go p.Send(Response{
		Type:    AttributeChanged,
		Payload: minion,
	})
	return false
}

func (p *Player) NotifyTurnStarted(event GameEvent) bool {
	data := event.GetData().(map[string]interface{})
	player := data["Player"].(*Player)
	duration := data["Duration"].(time.Duration)

	if player == p {
		go p.Send(Response{
			Type: StartTurn,
			Payload: TurnPayload{
				PlayerId:    player.Id,
				Duration:    duration,
				Board:       player.Board.Minions,
				Cards:       player.Hand.GetCards(),
				Mana:        player.GetMana(),
				CardsInHand: player.Hand.Length(),
			},
		})
	} else {
		go p.Send(Response{
			Type: WaitTurn,
			Payload: TurnPayload{
				OpponentId:  player.Id,
				Mana:        player.GetMana(),
				Board:       player.Board.Minions,
				Duration:    duration,
				CardsInHand: player.Hand.Length(),
			},
		})
	}
	return false
}

type Board struct {
	Minions map[uuid.UUID]*ActiveMinion
}

func NewBoard() *Board {
	return &Board{
		Minions: make(map[uuid.UUID]*ActiveMinion),
	}
}

func (b *Board) MinionsCount() int {
	return len(b.Minions)
}

func (b *Board) GetMinion(minionId uuid.UUID) (*ActiveMinion, bool) {
	minion, ok := b.Minions[minionId]
	return minion, ok
}

func (b *Board) Remove(minion *ActiveMinion) {
	delete(b.Minions, minion.Id)
}

func (b *Board) Place(card *ActiveMinion) error {
	if b.MinionsCount() == MAX_MINIONS {
		return errors.New("Cannot place minion, board is full")
	}
	b.Minions[card.Id] = card
	return nil
}

func (b *Board) ActivateAll() map[uuid.UUID]*ActiveMinion {
	for _, minion := range b.Minions {
		minion.SetState(Active{})
	}
	return b.Minions
}
