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
		ability.SetTrigger(CreateTrigger(data.Ability.Trigger))
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
		effect:  effect,
		trigger: CreateTrigger(data.Trigger),
	}, nil
}

func CreateTrigger(identifier string) *Trigger {
	var event GameEventType
	var description string
	var condition func(card ActiveCard, event GameEvent) bool

	switch identifier {
	case "card_played":
		event = CardPlayedEvent
		description = "When you play a card"
		condition = func(card ActiveCard, event GameEvent) bool {
			played := event.GetData().(ActiveCard)
			return card.GetPlayer() == played.GetPlayer()
		}
	case "opponent_card_played":
		event = CardPlayedEvent
		description = "When your opponent plays a card"
		condition = func(card ActiveCard, event GameEvent) bool {
			played := event.GetData().(*ActiveMinion)
			return card.GetPlayer() != played.GetPlayer()
		}
	case "turn_started":
		event = TurnStartedEvent
		description = "At the start of your turn"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			data := event.GetData().(map[string]interface{})
			return minion.player == data["Player"].(*Player)
		}
	case "opponent_turn_started":
		event = TurnStartedEvent
		description = "At the start of your opponent's turn"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			data := event.GetData().(map[string]interface{})
			return minion.player != data["Player"].(*Player)
		}
	case "mana_gained":
		event = ManaGainedEvent
		description = "When you gain mana"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			player := event.GetData().(*Player)
			return minion.player == player
		}
	case "opponent_mana_gained":
		event = ManaGainedEvent
		description = "When your opponent gains mana"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			player := event.GetData().(*Player)
			return minion.player != player
		}
	case "damage_increased":
		event = DamageIncreasedEvent
		description = "When this minion gain damage"
		condition = func(card ActiveCard, event GameEvent) bool {
			return card == event.GetData().(Card)
		}
	case "allied_damage_increased":
		event = DamageIncreasedEvent
		description = "When an allied minion gains damage"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return minion != affected && minion.player == affected.player
		}
	case "opponent_damage_increased":
		event = DamageIncreasedEvent
		description = "When an opponent minion gains damage"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return minion.player != affected.player
		}
	case "minion_state_changed":
		event = StateChangedEvent
		description = "When this minion changes state"
		condition = func(card ActiveCard, event GameEvent) bool {
			return card == event.GetData().(Card)
		}
	case "allied_minion_state_changed":
		event = StateChangedEvent
		description = "When an allied minion changes state"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return affected != minion && minion.player == affected.player
		}
	case "opponent_minion_state_changed":
		event = StateChangedEvent
		description = "When an opponent minion changes state"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			affected := event.GetData().(*ActiveMinion)
			return minion.player != affected.player
		}
	case "minion_destroyed":
		event = MinionDestroyedEvent
		description = "When this minion is destroyed"
		condition = func(card ActiveCard, event GameEvent) bool {
			return card == event.GetData().(Card)
		}
	case "allied_minion_destroyed":
		event = MinionDestroyedEvent
		description = "When an allied minion is destroyed"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			destroyed := event.GetData().(*ActiveMinion)
			return destroyed != minion && minion.player == destroyed.player
		}
	case "opponent_minion_destroyed":
		event = MinionDestroyedEvent
		description = "When an opponent minion is destroyed"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			destroyed := event.GetData().(*ActiveMinion)
			return minion.player != destroyed.player
		}
	case "minion_damaged":
		event = MinionDamagedEvent
		description = "When this minion takes damage"
		condition = func(card ActiveCard, event GameEvent) bool {
			payload := event.GetData().(MinionDamagedPayload)
			return card == payload.Defender
		}
	case "allied_minion_damaged":
		event = MinionDamagedEvent
		description = "When an allied minion takes damage"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			payload := event.GetData().(MinionDamagedPayload)
			return payload.Defender != card && minion.player == payload.Defender.player
		}
	case "opponent_minion_damaged":
		event = MinionDamagedEvent
		description = "When an opponent minion takes damage"
		condition = func(card ActiveCard, event GameEvent) bool {
			minion := card.(*ActiveMinion)
			payload := event.GetData().(MinionDamagedPayload)
			return minion.player != payload.Defender.player
		}
	default:
		return nil
	}

	return &Trigger{
		event:       event,
		condition:   condition,
		Description: description,
	}
}
