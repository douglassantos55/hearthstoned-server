package pkg

import (
	"testing"
	"time"
)

func TestAttackEvent(t *testing.T) {
	manager := NewGameManager()

	p1 := NewSocket()
	p2 := NewSocket()

	game := manager.CreateGame([]*Socket{p1, p2})
	game.StartTurn()

	<-p2.Outgoing             // wait turn
	response := <-p1.Outgoing // start turn
	payload := response.Payload.(TurnPayload)

	attacker := payload.Cards[0]
	attacker.Mana = 1
	attacker.Damage = 1
	attacker.Health = 1

	game.PlayCard(attacker.Id, p1)
	<-p2.Outgoing // card played

	game.EndTurn()
	<-p1.Outgoing // wait turn

	response = <-p2.Outgoing // start turn
	payload = response.Payload.(TurnPayload)

	defender := payload.Cards[0]
	defender.Mana = 1
	defender.Damage = 1
	defender.Health = 2

	game.PlayCard(defender.Id, p2)
	<-p1.Outgoing // card played

	game.EndTurn()

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	manager.Process(Event{
		Type:   Attack,
		Player: p1,
		Payload: CombatPayload{
			GameId:   game.Id.String(),
			Attacker: attacker.Id.String(),
			Defender: defender.Id.String(),
		},
	})

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected damage taken response")
	case response := <-p1.Outgoing:
		if response.Type != MinionDamageTaken {
			t.Errorf("Expected %v, got %v", MinionDamageTaken, response.Type)
		}
		minion := response.Payload.(*Card)
		if minion.Id != defender.Id {
			t.Errorf("Expected %v, got %v", defender.Id, minion.Id)
		}
		if minion.Health != 1 {
			t.Errorf("Expecetd %v health, got %v", 1, minion.Health)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected damage taken response")
	case response := <-p2.Outgoing:
		if response.Type != MinionDamageTaken {
			t.Errorf("Expected %v, got %v", MinionDamageTaken, response.Type)
		}
		minion := response.Payload.(*Card)
		if minion.Id != defender.Id {
			t.Errorf("Expected %v, got %v", defender.Id, minion.Id)
		}
		if minion.Health != 1 {
			t.Errorf("Expecetd %v health, got %v", 1, minion.Health)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected minion destroyed response")
	case response := <-p1.Outgoing:
		if response.Type != MinionDestroyed {
			t.Errorf("Expected %v, got %v", MinionDestroyed, response.Type)
		}
		minion := response.Payload.(*Card)
		if minion.Id != attacker.Id {
			t.Errorf("Expected %v, got %v", attacker.Id, minion.Id)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected minion destroyed response")
	case response := <-p2.Outgoing:
		if response.Type != MinionDestroyed {
			t.Errorf("Expected %v, got %v", MinionDestroyed, response.Type)
		}
		minion := response.Payload.(*Card)
		if minion.Id != attacker.Id {
			t.Errorf("Expected %v, got %v", attacker.Id, minion.Id)
		}
	}
}
