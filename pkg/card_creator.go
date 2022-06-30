package pkg

import (
	"encoding/json"
	"fmt"
	"os"
)

var cardData []CardData

func GetCards() []Card {
	cards := []Card{}
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
	if len(cardData) > 0 {
		return cardData, nil
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, &cardData)
	if err != nil {
		return nil, err
	}

	return cardData, nil
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
	Type    string                 `json:"type"`
	Trigger string                 `json:"trigger"`
	Params  map[string]interface{} `json:"params"`
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
		minion.SetAbility(ability)
	}

	return minion, nil
}

func CreateSpellCard(data CardData) (Card, error) {
	ability, err := CreateAbility(data.Ability)
	if err != nil {
		return nil, err
	}

	spell := NewSpell(data.Name, data.Mana, ability)
	if data.Ability.Trigger != "" {
		trigger := CreateTrigger(data.Ability.Trigger)
		return Trigerable(trigger, spell), nil
	}

	return spell, nil
}

func CreateAbility(data AbilityData) (*Ability, error) {
	var effect Effect

	switch data.Type {
	case "gain_damage":
		amount := data.Params["amount"].(float64)
		effect = GainDamageEffect(int(amount))
	case "gain_mana":
		amount := data.Params["amount"].(float64)
		effect = GainManaEffect(int(amount))
	default:
		return nil, fmt.Errorf("Invalid ability type: %v", data.Type)
	}

	return &Ability{
		Effect:  effect,
		Trigger: CreateTrigger(data.Trigger),
	}, nil
}

func CreateTrigger(identifier string) *Trigger {
	var event GameEventType
	var condition func(card Card, event GameEvent) bool

	switch identifier {
	case "card_played":
		event = CardPlayedEvent
		condition = func(card Card, event GameEvent) bool {
			// TODO: this won't work for spells
			minion := card.(*ActiveMinion)
			if played, ok := event.GetData().(*ActiveMinion); ok {
				fmt.Printf("played: %v\n", played)
				return minion.player == played.player
			}
			return false
		}
	case "opponent_card_played":
		event = CardPlayedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			played := event.GetData().(*ActiveMinion)
			return minion.player != played.player
		}
	case "turn_started":
		event = TurnStartedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			data := event.GetData().(map[string]interface{})
			return minion.player == data["Player"].(*Player)
		}
	case "opponent_turn_started":
		event = TurnStartedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			data := event.GetData().(map[string]interface{})
			return minion.player != data["Player"].(*Player)
		}
	case "mana_gained":
		event = ManaGainedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			player := event.GetData().(*Player)
			return minion.player == player
		}
	case "opponent_mana_gained":
		event = ManaGainedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			player := event.GetData().(*Player)
			return minion.player != player
		}
	case "damage_increased":
		event = DamageIncreasedEvent
		condition = func(card Card, event GameEvent) bool {
			return card == event.GetData().(Card)
		}
	case "allied_damage_increased":
		event = DamageIncreasedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return minion != affected && minion.player == affected.player
		}
	case "opponent_damage_increased":
		event = DamageIncreasedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return minion.player != affected.player
		}
	case "minion_state_changed":
		event = StateChangedEvent
		condition = func(card Card, event GameEvent) bool {
			return card == event.GetData().(Card)
		}
	case "allied_minion_state_changed":
		event = StateChangedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return affected != minion && minion.player == affected.player
		}
	case "opponent_minion_state_changed":
		event = StateChangedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return minion.player != affected.player
		}
	case "minion_destroyed":
		event = MinionDestroyedEvent
		condition = func(card Card, event GameEvent) bool {
			return card == event.GetData().(Card)
		}
	case "allied_minion_destroyed":
		event = MinionDestroyedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			destroyed := event.GetData().(*ActiveMinion)
			return destroyed != minion && minion.player == destroyed.player
		}
	case "opponent_minion_destroyed":
		event = MinionDestroyedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			destroyed := event.GetData().(*ActiveMinion)
			return minion.player != destroyed.player
		}
	case "minion_damaged":
		event = MinionDamagedEvent
		condition = func(card Card, event GameEvent) bool {
			payload := event.GetData().(MinionDamagedPayload)
			return card == payload.Defender
		}
	case "allied_minion_damaged":
		event = MinionDamagedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			payload := event.GetData().(MinionDamagedPayload)
			return payload.Defender != card && minion.player == payload.Defender.player
		}
	case "opponent_minion_damaged":
		event = MinionDamagedEvent
		condition = func(card Card, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			payload := event.GetData().(MinionDamagedPayload)
			return minion.player != payload.Defender.player
		}
	default:
		return nil
	}

	return &Trigger{
		Event:     event,
		condition: condition,
	}
}
