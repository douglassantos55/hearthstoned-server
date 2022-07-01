package pkg

import "fmt"

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
	if player, ok := target.(*Player); ok {
		g.player = player
	} else if card, ok := target.(ActiveCard); ok {
		g.player = card.GetPlayer()
	}
}

func (g *GainMana) Cast() GameEvent {
	g.player.AddMana(g.amount)

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

type DrawCard struct {
	amount int
	player *Player
}

func (d *DrawCard) GetDescription() string {
	return fmt.Sprintf("draw %v cards", d.amount)
}

func (d *DrawCard) SetTarget(target interface{}) {
	if player, ok := target.(*Player); ok {
		d.player = player
	} else if card, ok := target.(ActiveCard); ok {
		d.player = card.GetPlayer()
	}
}

func (d *DrawCard) Cast() GameEvent {
	cards := d.player.DrawCards(d.amount)
	return &CardsDrawn{
		Cards:  cards,
		Player: d.player,
	}
}
