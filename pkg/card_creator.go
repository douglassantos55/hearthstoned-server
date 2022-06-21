package pkg

import (
	"encoding/json"
	"fmt"
	"os"
)

var cards []Card

func GetCards() []Card {
	if len(cards) > 0 {
		return cards
	}

	data, err := loadCards("../cards.json")
	if err != nil {
		return cards
	}

	for _, data := range data {
		card, err := CreateCard(data)
		if err == nil {
			cards = append(cards, card)
		}
	}
	return cards
}

func loadCards(filename string) ([]CardData, error) {
	var cardsData []CardData
	contents, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, &cardsData)
	if err != nil {
		return nil, err
	}

	return cardsData, nil
}

type CardData struct {
	Type    string      `json:"type"`
	Name    string      `json:"name"`
	Mana    int         `json:"mana"`
	Damage  int         `json:"damage"`
	Health  int         `json:"health"`
	Ability AbilityData `json:"ability"`
}

type AbilityData struct {
	Type      string                 `json:"type"`
	Trigger   string                 `json:"trigger"`
	Condition string                 `json:"condition"`
	Params    map[string]interface{} `json:"params"`
}

type TriggerData struct {
	Event string `json:"string"`
}

func CreateCard(data CardData) (Card, error) {
	switch data.Type {
	case "minion":
		return CreateMinionCard(data)
	case "spell":
		return CreateSpellCard(data)
	default:
		return nil, fmt.Errorf("Invalid card type: %v", data.Type)
	}
}

func CreateMinionCard(data CardData) (Card, error) {
	minion := NewCard(data.Name, data.Mana, data.Damage, data.Health)
	ability, err := CreateAbility(data.Ability)

	if err == nil {
		trigger := CreateTrigger(data.Ability.Trigger, data.Ability.Condition)
		minion.SetAbility(trigger, ability)
	}

	return minion, nil
}

func CreateSpellCard(data CardData) (Card, error) {
	ability, err := CreateAbility(data.Ability)
	if err != nil {
		return nil, err
	}

	spell := NewSpell(data.Mana, ability)
	if data.Ability.Trigger != "" {
		trigger := CreateTrigger(data.Ability.Trigger, data.Ability.Condition)
		return Trigerable(trigger, spell), nil
	}

	return spell, nil
}

func CreateAbility(data AbilityData) (Ability, error) {
	switch data.Type {
	case "gain_damage":
		amount := data.Params["amount"].(float64)
		return &GainDamage{amount: int(amount)}, nil
	case "gain_mana":
		amount := data.Params["amount"].(float64)
		return &GainMana{amount: int(amount)}, nil
	default:
		return nil, fmt.Errorf("Invalid ability type: %v", data.Type)
	}
}

func CreateTrigger(event string, condition string) *Trigger {
	return &Trigger{
		Event:     event,
		Condition: CreateCondition(condition),
	}
}

func CreateCondition(identifier string) func(card Card, event GameEvent) bool {
	switch identifier {
	case "allied":
		return func(card Card, event GameEvent) bool {
			minion := event.GetData().(*ActiveMinion)
			_, exists := minion.player.board.GetMinion(card.GetId())
			return exists
		}
	case "self":
		return func(card Card, event GameEvent) bool {
			played := event.GetData().(Card)
			return card == played
		}
	case "current":
		return func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			data := event.GetData().(map[string]interface{})
			player := data["Player"].(*Player)
			return minion.player == player
		}
	case "opponent":
		return func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			player := event.GetData().(*Player)
			return minion.player != player
		}
	default:
		return nil
	}
}
