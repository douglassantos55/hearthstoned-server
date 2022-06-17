package pkg

import (
	"encoding/json"
	"fmt"
)

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

func CreateCard(cardJson string) (Card, error) {
	var data CardData
	err := json.Unmarshal([]byte(cardJson), &data)

	if err != nil {
		return nil, err
	}

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
	minion := NewCard(data.Mana, data.Damage, data.Health)
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
			dead := event.GetData().(*ActiveMinion)
			_, exists := dead.player.board.GetMinion(card.GetId())
			return exists
		}
	case "self":
		return func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			player := event.GetData().(*Player)
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
