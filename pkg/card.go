package pkg

import (
	"container/list"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

type Card interface {
	GetId() uuid.UUID
	GetMana() int
}

type Ability interface {
	Cast() GameEvent
	SetTarget(target interface{})
}

type TriggerableSpell struct {
	*Spell
	Trigger *Trigger
}

func Trigerable(trigger *Trigger, spell *Spell) *TriggerableSpell {
	return &TriggerableSpell{
		Spell:   spell,
		Trigger: trigger,
	}
}

type GainMana struct {
	amount int
	player *Player
}

func GainManaAbility(amount int) *GainMana {
	return &GainMana{
		amount: amount,
	}
}

func (g *GainMana) SetTarget(target interface{}) {
	g.player = target.(*Player)
}

func (g *GainMana) Cast() GameEvent {
	g.player.GainMana(g.amount)

	return &ManaGained{
		Player: g.player,
	}
}

type GainDamage struct {
	amount int
	minion *Minion
}

func (g *GainDamage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"Name":        "Gain damage",
		"Description": fmt.Sprintf("Increase %v damage", g.amount),
	})
}

func GainDamageAbility(amount int) *GainDamage {
	return &GainDamage{
		amount: amount,
	}
}

func (g *GainDamage) Cast() GameEvent {
	g.minion.GainDamage(g.amount)
	return &DamageIncreased{Minion: g.minion}
}

func (g *GainDamage) SetTarget(target interface{}) {
	g.minion = target.(*Minion)
}

// Spell card
type Spell struct {
	Id      uuid.UUID
	Mana    int
	Ability Ability
}

func NewSpell(mana int, ability Ability) *Spell {
	return &Spell{
		Id:      uuid.New(),
		Mana:    mana,
		Ability: ability,
	}
}

func (s *Spell) GetId() uuid.UUID {
	return s.Id
}

func (s *Spell) GetMana() int {
	return s.Mana
}

func (s *Spell) Execute(caster *Player) GameEvent {
	s.Ability.SetTarget(caster)
	return s.Ability.Cast()
}

type Minion struct {
	Id    uuid.UUID
	mutex *sync.Mutex

	Name    string
	Mana    int
	Damage  int
	Health  int
	Ability Ability
	trigger *Trigger
}

func NewCard(name string, mana, damage, health int) *Minion {
	return &Minion{
		Id:    uuid.New(),
		mutex: new(sync.Mutex),

		Name:   name,
		Mana:   mana,
		Damage: damage,
		Health: health,
	}
}

func (c *Minion) GetId() uuid.UUID {
	return c.Id
}

func (c *Minion) GetMana() int {
	return c.Mana
}

func (m *Minion) HasAbility() bool {
	return m.Ability != nil
}

func (m *Minion) CastAbility() GameEvent {
	m.Ability.SetTarget(m)
	return m.Ability.Cast()
}

func (m *Minion) SetAbility(trigger *Trigger, ability Ability) {
	m.trigger = trigger
	m.Ability = ability
}

func (m *Minion) GainDamage(amount int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Damage += amount
}

func (m *Minion) GetDamage() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.Damage
}

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

type MinionState interface {
	CanAttack() bool
	CanCounterAttack() bool
}

type Active struct{}

func (e Active) CanAttack() bool {
	return true
}

func (e Active) CanCounterAttack() bool {
	return true
}

type Exhausted struct{}

func (e Exhausted) CanAttack() bool {
	return false
}

func (e Exhausted) CanCounterAttack() bool {
	return true
}

// ActiveMinion represents a placed card on board, giving it the
// ability to attack and states
type ActiveMinion struct {
	*Minion
	player *Player
	state  MinionState
	State  string
}

func NewMinion(card *Minion, player *Player) *ActiveMinion {
	state := Exhausted{}
	return &ActiveMinion{
		Minion: card,
		player: player,
		state:  state,
		State:  reflect.TypeOf(state).Name(),
	}
}

func (m *ActiveMinion) CanAttack() bool {
	return m.state.CanAttack()
}

func (m *ActiveMinion) CanCounterAttack() bool {
	return m.state.CanCounterAttack()
}

func (m *ActiveMinion) GetState() MinionState {
	return m.state
}

func (m *ActiveMinion) SetState(state MinionState) {
	m.state = state
	m.State = reflect.TypeOf(state).Name()
}

// Reduces minion health and returns wether it survives or not
func (m *ActiveMinion) RemoveHealth(amount int) bool {
	m.Health -= amount
	return m.Health > 0
}
