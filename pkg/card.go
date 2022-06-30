package pkg

import (
	"container/list"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type HasAbility interface {
	HasAbility() bool
	GetAbility() *Ability
}

type Card interface {
	HasAbility
	GetId() uuid.UUID
	GetMana() int
}

type ActiveCard interface {
	Card
	CastAbility() GameEvent
	GetPlayer() *Player
	SetPlayer(player *Player)
}

type Ability struct {
	effect  Effect
	trigger *Trigger
}

func (a *Ability) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"Description": strings.Join([]string{
			a.trigger.Description,
			a.effect.GetDescription(),
		}, ", "),
	})
}

func (a *Ability) SetTarget(target interface{}) {
	a.effect.SetTarget(target)
}

func (a *Ability) Cast() GameEvent {
	return a.effect.Cast()
}

func (a *Ability) SetTrigger(trigger *Trigger) {
	a.trigger = trigger
}

type Effect interface {
	Cast() GameEvent
	GetDescription() string
	SetTarget(target interface{})
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

func (g *GainMana) GetDescription() string {
	return fmt.Sprintf("gain %v mana", g.amount)
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

func (g *GainDamage) GetDescription() string {
	return fmt.Sprintf("gain %v damage", g.amount)
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
	Id      uuid.UUID
	Name    string
	Mana    int
	Ability *Ability
}

func NewSpell(name string, mana int, ability *Ability) *Spell {
	return &Spell{
		Id:      uuid.New(),
		Name:    name,
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

func (s *Spell) HasAbility() bool {
	return s.Ability != nil
}

func (s *Spell) GetAbility() *Ability {
	return s.Ability
}

func (s *Spell) Activate() *ActiveSpell {
	return &ActiveSpell{
		Spell: s,
	}
}

type ActiveSpell struct {
	*Spell
	player *Player
}

func (s *ActiveSpell) GetPlayer() *Player {
	return s.player
}

func (s *ActiveSpell) SetPlayer(player *Player) {
	s.player = player
}

func (s *ActiveSpell) CastAbility() GameEvent {
	return s.Execute(s.GetPlayer())
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

func (m *Minion) GetAbility() *Ability {
	return m.Ability
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

func NewMinion(card *Minion) *ActiveMinion {
	state := Exhausted{}

	return &ActiveMinion{
		Minion: card,
		state:  state,
		State:  reflect.TypeOf(state).Name(),
	}
}

func (m *ActiveMinion) GetPlayer() *Player {
	return m.player
}

func (m *ActiveMinion) SetPlayer(player *Player) {
	m.player = player
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
