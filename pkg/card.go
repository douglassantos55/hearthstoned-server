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

type Ability struct {
	Effect      Effect
	Trigger     *Trigger
	Description string
}

func (a *Ability) SetTarget(target interface{}) {
	a.Effect.SetTarget(target)
}

func (a *Ability) Cast() GameEvent {
	return a.Effect.Cast()
}

type Effect interface {
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

func GainManaEffect(amount int) *GainMana {
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
	minion *ActiveMinion
}

func (g *GainDamage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"Name":        "Gain damage",
		"Description": fmt.Sprintf("Increase %v damage", g.amount),
	})
}

func GainDamageEffect(amount int) *GainDamage {
	return &GainDamage{
		amount: amount,
	}
}

func (g *GainDamage) Cast() GameEvent {
	g.minion.GainDamage(g.amount)
	return &DamageIncreased{Minion: g.minion}
}

func (g *GainDamage) SetTarget(target interface{}) {
	g.minion = target.(*ActiveMinion)
}

// Spell card
type Spell struct {
	Id     uuid.UUID
	Name   string
	Mana   int
	Effect Effect
}

func NewSpell(name string, mana int, effect Effect) *Spell {
	return &Spell{
		Id:     uuid.New(),
		Name:   name,
		Mana:   mana,
		Effect: effect,
	}
}

func (s *Spell) GetId() uuid.UUID {
	return s.Id
}

func (s *Spell) GetMana() int {
	return s.Mana
}

func (s *Spell) Execute(caster *Player) GameEvent {
	s.Effect.SetTarget(caster)
	return s.Effect.Cast()
}

type Minion struct {
	Id    uuid.UUID
	mutex *sync.Mutex

	Name    string
	Mana    int
	Damage  int
	Health  int
	Ability *Ability
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

func (m *Minion) SetAbility(ability *Ability) {
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

func (m *Minion) GetHealth() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.Health
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state
}

func (m *ActiveMinion) SetState(state MinionState) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state = state
	m.State = reflect.TypeOf(state).Name()
}

// Reduces minion health and returns wether it survives or not
func (m *ActiveMinion) RemoveHealth(amount int) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Health -= amount
	return m.Health > 0
}

func (m *ActiveMinion) CastAbility() GameEvent {
	m.Ability.SetTarget(m)
	return m.Ability.Cast()
}
