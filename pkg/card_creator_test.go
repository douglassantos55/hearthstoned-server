package pkg

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCreateMinion(t *testing.T) {
	card, _ := CreateCard(CardData{
		Type:   "minion",
		Name:   "Black Magician of Doom",
		Mana:   2,
		Damage: 5,
		Health: 2,
	})

	minion := card.(*Minion)

	if minion.GetMana() != 2 {
		t.Errorf("Expected %v mana, got %v", 2, minion.GetMana())
	}

	if minion.Damage != 5 {
		t.Errorf("Expected %v damage, got %v", 5, minion.Damage)
	}

	if minion.Health != 2 {
		t.Errorf("Expected %v health, got %v", 2, minion.Health)
	}
}

func TestCreateSpell(t *testing.T) {
	card, _ := CreateCard(CardData{
		Type: "spell",
		Name: "Unlucky",
		Mana: 2,
		Ability: AbilityData{
			Type:   "gain_mana",
			Params: map[string]interface{}{"amount": 4.0},
		},
	})

	ability := card.GetAbility()
	if ability.trigger != nil {
		t.Error("Should not be a triggerable spell")
	}

	spell, ok := card.(*Spell)
	if !ok {
		t.Error("Should be a normal spell")
	}

	if spell.GetMana() != 2 {
		t.Errorf("expected %v mana, got %v", 2, spell.GetMana())
	}

	player := NewPlayer(NewTestSocket())
	spell.Execute(player)

	if player.GetTotalMana() != 4 {
		t.Errorf("Expected %v total mana, got %v", 4, player.GetTotalMana())
	}
}

func TestCreateMinionWithAbility(t *testing.T) {
	card, _ := CreateCard(CardData{
		Type:   "minion",
		Name:   "Crazy Shirtless Dude",
		Mana:   1,
		Damage: 1,
		Health: 1,
		Ability: AbilityData{
			Type:    "gain_damage",
			Params:  map[string]interface{}{"amount": 1.0},
			Trigger: "turn_started",
		},
	})

	minion := card.(*Minion)

	if minion.Ability.trigger == nil {
		t.Error("minion ability should have a trigger")
	}

	if minion.Ability.trigger.event != TurnStartedEvent {
		t.Errorf("Expected %v event, got %v", TurnStartedEvent, minion.Ability.trigger.event)
	}

	active := NewMinion(minion)
	active.CastAbility()

	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}

func TestSpellWithTrigger(t *testing.T) {
	card, err := CreateCard(CardData{
		Type: "spell",
		Name: "Unlucky",
		Mana: 2,
		Ability: AbilityData{
			Type:    "gain_mana",
			Params:  map[string]interface{}{"amount": 4.0},
			Trigger: "turn_started",
		},
	})

	if err != nil {
		t.Error(err)
	}

	ability := card.GetAbility()
	if ability.trigger == nil {
		t.Error("Should not be a normal spell")
	}

	if ability.trigger == nil {
		t.Error("Should be a triggerable spell")
	}

	if ability.trigger == nil {
		t.Error("Should have a trigger")
	}
}

func TestMinionWithCondition(t *testing.T) {
	card, err := CreateCard(CardData{
		Type:   "minion",
		Name:   "Crazy Shirtless Dude",
		Mana:   1,
		Damage: 1,
		Health: 1,
		Ability: AbilityData{
			Type:    "gain_damage",
			Params:  map[string]interface{}{"amount": 1.0},
			Trigger: "allied_minion_destroyed",
		},
	})

	if err != nil {
		t.Error(err)
	}

	minion, ok := card.(*Minion)
	if !ok {
		t.Error("Expected a minion card")
	}

	if minion.Ability == nil {
		t.Error("Should have an ability")
	}

	if minion.Ability.trigger == nil {
		t.Error("Should have a trigger")
	}

	if minion.Ability.trigger.condition == nil {
		t.Error("Expected trigger condition")
	}

	player := NewPlayer(NewTestSocket())

	active := NewMinion(minion)
	active.SetPlayer(player)

	deadMinion := NewMinion(NewCard("", 1, 1, 2))
	deadMinion.SetPlayer(player)

	dispatcher := NewGameDispatcher()
	dispatcher.Subscribe(active.Ability.trigger.event, func(event GameEvent) bool {
		if active.Ability.trigger.condition(active, event) {
			active.CastAbility()
		}
		return true
	})

	dispatcher.Dispatch(NewDestroyedEvent(deadMinion))

	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}

func TestTurnStartedCondition(t *testing.T) {
	card, err := CreateCard(CardData{
		Type:   "minion",
		Name:   "Crazy Shirtless Dude",
		Mana:   1,
		Damage: 1,
		Health: 1,
		Ability: AbilityData{
			Type:    "gain_damage",
			Params:  map[string]interface{}{"amount": 1.0},
			Trigger: "turn_started",
		},
	})

	if err != nil {
		t.Error(err)
	}

	player := NewPlayer(NewTestSocket())
	minion := NewMinion(card.(*Minion))
	minion.SetPlayer(player)

	dispatcher := NewGameDispatcher()
	dispatcher.Subscribe(TurnStartedEvent, func(event GameEvent) bool {
		if minion.Ability.trigger.condition(minion, event) {
			minion.CastAbility()
		}
		return true
	})

	dispatcher.Dispatch(NewTurnStartedEvent(player, time.Second))

	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}

func TestJSONWithAbility(t *testing.T) {
	card, _ := CreateCard(CardData{
		Type:   "minion",
		Name:   "Crazy Shirtless Dude",
		Mana:   1,
		Damage: 1,
		Health: 1,
		Ability: AbilityData{
			Type:    "gain_damage",
			Params:  map[string]interface{}{"amount": 1.0},
			Trigger: "turn_started",
		},
	})

	_, err := json.Marshal(card)
	if err != nil {
		t.Error(err)
	}
}

func TestTriggers(t *testing.T) {
	t.Run("spell played", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "card_played",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		ability := &Ability{effect: GainDamageEffect(2)}
		spell := NewSpell("Buff mimana", 1, ability)
		activeSpell := spell.Activate()
		activeSpell.SetPlayer(player)
		dispatcher.Dispatch(NewCardPlayedEvent(activeSpell))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("card played", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "card_played",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		played := NewMinion(NewCard("foo", 1, 1, 1))
		played.SetPlayer(player)
		dispatcher.Dispatch(NewCardPlayedEvent(played))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent card played", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_card_played",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		otherPlayer := NewPlayer(NewTestSocket())
		played := NewMinion(NewCard("foo", 1, 1, 1))
		played.SetPlayer(otherPlayer)

		dispatcher.Dispatch(NewCardPlayedEvent(played))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent turn started", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_turn_started",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		otherPlayer := NewPlayer(NewTestSocket())
		dispatcher.Dispatch(NewTurnStartedEvent(otherPlayer, time.Second))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("mana gained", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "mana_gained",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(ManaGained{player})

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent mana gained", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_mana_gained",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		otherPlayer := NewPlayer(NewTestSocket())
		dispatcher.Dispatch(ManaGained{otherPlayer})

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("damage increased", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "damage_increased",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(DamageIncreased{minion})

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("allied damage increased", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "allied_damage_increased",
			},
		})

		player := NewPlayer(NewTestSocket())

		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		allied := NewMinion(NewCard("allied", 1, 1, 1))
		allied.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(DamageIncreased{allied})

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent damage increased", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_damage_increased",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		opponent := NewPlayer(NewTestSocket())
		enemy := NewMinion(NewCard("allied", 1, 1, 1))
		enemy.SetPlayer(opponent)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(DamageIncreased{enemy})

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("minion state changed", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "minion_state_changed",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewStateChangedEvent(minion))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("allied minion state changed", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "allied_minion_state_changed",
			},
		})

		player := NewPlayer(NewTestSocket())

		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		allied := NewMinion(NewCard("allied", 1, 1, 1))
		allied.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewStateChangedEvent(allied))
		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent minion state changed", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_minion_state_changed",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		opponent := NewPlayer(NewTestSocket())
		enemy := NewMinion(NewCard("enemy", 1, 1, 1))
		enemy.SetPlayer(opponent)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewStateChangedEvent(enemy))
		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("minion destroyed", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "minion_destroyed",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewDestroyedEvent(minion))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("allied minion destroyed", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "allied_minion_destroyed",
			},
		})

		player := NewPlayer(NewTestSocket())

		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		allied := NewMinion(NewCard("allied", 1, 1, 1))
		allied.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return false
		})

		dispatcher.Dispatch(NewDestroyedEvent(minion))
		if minion.GetDamage() != 2 {
			t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
		}

		dispatcher.Dispatch(NewDestroyedEvent(allied))
		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent minion destroyed", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_minion_destroyed",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		opponent := NewPlayer(NewTestSocket())
		enemy := NewMinion(NewCard("enemy", 1, 1, 1))
		enemy.SetPlayer(opponent)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewDestroyedEvent(enemy))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("minion damaged", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "minion_damaged",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		opponent := NewPlayer(NewTestSocket())
		attacker := NewMinion(NewCard("attacker", 1, 1, 1))
		attacker.SetPlayer(opponent)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewDamageEvent(attacker, minion))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("allied minion damaged", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "allied_minion_damaged",
			},
		})

		player := NewPlayer(NewTestSocket())

		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		allied := NewMinion(NewCard("allied", 1, 1, 1))
		allied.SetPlayer(player)

		opponent := NewPlayer(NewTestSocket())
		attacker := NewMinion(NewCard("attacker", 1, 1, 1))
		attacker.SetPlayer(opponent)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return false
		})

		dispatcher.Dispatch(NewDamageEvent(attacker, minion))
		if minion.GetDamage() != 2 {
			t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
		}

		dispatcher.Dispatch(NewDamageEvent(attacker, allied))
		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("opponent minion damaged", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "opponent_minion_damaged",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		opponent := NewPlayer(NewTestSocket())
		attacker := NewMinion(NewCard("attacker", 1, 1, 1))
		attacker.SetPlayer(opponent)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(NewDamageEvent(minion, attacker))

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})

	t.Run("cards drawn", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type:   "minion",
			Name:   "Crazy Shirtless Dude",
			Mana:   1,
			Damage: 2,
			Health: 1,
			Ability: AbilityData{
				Type:    "gain_damage",
				Params:  map[string]interface{}{"amount": 2.0},
				Trigger: "cards_drawn",
			},
		})

		player := NewPlayer(NewTestSocket())
		minion := NewMinion(card.(*Minion))
		minion.SetPlayer(player)

		dispatcher := NewGameDispatcher()
		dispatcher.Subscribe(minion.Ability.trigger.event, func(event GameEvent) bool {
			if minion.Ability.trigger.condition(minion, event) {
				minion.CastAbility()
			}
			return true
		})

		dispatcher.Dispatch(CardsDrawn{player, []Card{}})

		if minion.GetDamage() != 4 {
			t.Errorf("Expected %v damage, got %v", 4, minion.GetDamage())
		}
	})
}

func TestSpells(t *testing.T) {
	t.Run("draw cards", func(t *testing.T) {
		card, _ := CreateCard(CardData{
			Type: "spell",
			Name: "Crazy Shirtless Dude",
			Mana: 1,
			Ability: AbilityData{
				Type:   "draw_card",
				Params: map[string]interface{}{"amount": 4.0},
			},
		})

		player := NewPlayer(NewTestSocket())
		spell := card.(*Spell).Activate()
		spell.SetPlayer(player)
		spell.CastAbility()

		if player.hand.Length() != 4 {
			t.Errorf("Expected %v cards in hand, got %v", 4, player.hand.Length())
		}
	})
}
