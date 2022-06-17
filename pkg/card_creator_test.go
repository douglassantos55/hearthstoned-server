package pkg

import "testing"

func TestCreateMinion(t *testing.T) {
	card, _ := CreateCard(`{
        "type": "minion",
        "name": "Black Magician of Doom",
        "mana": 2,
        "damage": 5,
        "health": 2
    }`)

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
	card, _ := CreateCard(`{
        "type": "spell",
        "name": "Unlucky",
        "mana": 2,
        "ability": {
            "type": "gain_mana",
            "params": { "amount": 4 }
        }
    }`)

	if _, ok := card.(*TriggerableSpell); ok {
		t.Error("Should not be a triggerable spell")
	}

	spell, ok := card.(*Spell)
	if !ok {
		t.Error("Should be a normal spell")
	}

	if spell.GetMana() != 2 {
		t.Errorf("expected %v mana, got %v", 2, spell.GetMana())
	}

	player := NewPlayer(NewSocket())
	spell.Execute(player)

	if player.GetTotalMana() != 4 {
		t.Errorf("Expected %v total mana, got %v", 4, player.GetTotalMana())
	}
}

func TestCreateMinionWithAbility(t *testing.T) {
	card, _ := CreateCard(`{
        "type": "minion",
        "name": "Crazy Shirtless Dude",
        "mana": 1,
        "damage": 1,
        "health": 1,
        "ability": {
            "type": "gain_damage",
            "params": { "amount": 1 },
            "trigger": "turn_started"
        }
    }`)

	minion := card.(*Minion)

	if minion.Trigger == nil {
		t.Error("minion ability should have a trigger")
	}

	if minion.Trigger.Event != TurnStartedEvent {
		t.Errorf("Expected %v event, got %v", TurnStartedEvent, minion.Trigger.Event)
	}

	minion.CastAbility()

	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}

func TestSpellWithTrigger(t *testing.T) {
	card, err := CreateCard(`{
        "type": "spell",
        "name": "Unlucky",
        "mana": 2,
        "ability": {
            "type": "gain_mana",
            "params": { "amount": 4 },
            "trigger": "turn_started"
        }
    }`)

	if err != nil {
		t.Error(err)
	}

	if _, ok := card.(*Spell); ok {
		t.Error("Should not be a normal spell")
	}

	spell, ok := card.(*TriggerableSpell)
	if !ok {
		t.Error("Should be a triggerable spell")
	}

	if spell.Trigger == nil {
		t.Error("Should have a trigger")
	}
}

func TestMinionWithCondition(t *testing.T) {
	card, err := CreateCard(`{
        "type": "minion",
        "name": "Crazy Shirtless Dude",
        "mana": 1,
        "damage": 1,
        "health": 1,
        "ability": {
            "type": "gain_damage",
            "params": { "amount": 1 },
            "trigger": "minion_destroyed",
            "condition": "allied"
        }
    }`)

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

	if minion.Trigger == nil {
		t.Error("Should have a trigger")
	}

	if minion.Trigger.Condition == nil {
		t.Error("Expected trigger condition")
	}

	dispatcher := NewGameDispatcher()
	dispatcher.Subscribe(minion.Trigger.Event, func(event GameEvent) bool {
		if minion.Trigger.Condition(minion, event) {
			minion.CastAbility()
		}
		return true
	})

	player := NewPlayer(NewSocket())
	deadMinion := NewMinion(NewCard(1, 1, 2), player)

	player.PlayCard(minion)
	dispatcher.Dispatch(NewDestroyedEvent(deadMinion))

	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}

func TestTurnStartedCondition(t *testing.T) {
	card, err := CreateCard(`{
        "type": "minion",
        "name": "Crazy Shirtless Dude",
        "mana": 1,
        "damage": 1,
        "health": 1,
        "ability": {
            "type": "gain_damage",
            "params": { "amount": 1 },
            "trigger": "turn_started",
            "condition": "self"
        }
    }`)

	if err != nil {
		t.Error(err)
	}

	player := NewPlayer(NewSocket())
	minion := NewMinion(card.(*Minion), player)

	dispatcher := NewGameDispatcher()
	dispatcher.Subscribe(TurnStartedEvent, func(event GameEvent) bool {
		if minion.Trigger.Condition(minion, event) {
			minion.CastAbility()
		}
		return true
	})

	dispatcher.Dispatch(NewTurnStartedEvent(player))

	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}
