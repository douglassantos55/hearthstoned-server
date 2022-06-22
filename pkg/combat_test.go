package pkg

import (
	"testing"
	"time"
)

func TestCombat(t *testing.T) {
	t.Run("attack minion", func(t *testing.T) {
		manager := NewGameManager()

		p1 := NewTestSocket()
		p2 := NewTestSocket()

		game := manager.CreateGame([]*Socket{p1, p2})
		game.StartTurn()

		<-p2.Outgoing             // wait turn
		response := <-p1.Outgoing // start turn
		payload := response.Payload.(TurnPayload)

		attacker := payload.Cards[0].(*Minion)
		attacker.Mana = 1
		attacker.Damage = 1
		attacker.Health = 1

		game.PlayCard(attacker.Id, p1)

		<-p1.Outgoing // card played
		<-p2.Outgoing // card played

		game.EndTurn()
		<-p1.Outgoing // wait turn

		response = <-p2.Outgoing // start turn
		payload = response.Payload.(TurnPayload)

		defender := payload.Cards[0].(*Minion)
		defender.Mana = 1
		defender.Damage = 1
		defender.Health = 2

		game.PlayCard(defender.Id, p2)
		<-p1.Outgoing // card played
		<-p2.Outgoing // card played

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
			minion := response.Payload.(*Minion)
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
			minion := response.Payload.(*Minion)
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
			minion := response.Payload.(*Minion)
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
			minion := response.Payload.(*Minion)
			if minion.Id != attacker.Id {
				t.Errorf("Expected %v, got %v", attacker.Id, minion.Id)
			}
		}
	})

	t.Run("attack player", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewGameManager()
		game := manager.CreateGame([]*Socket{p1, p2})

		player := game.players[p2]

		attacker := NewCard("", 1, 3, 1)
		_, err := game.players[p1].PlayCard(attacker)

		if err != nil {
			t.Error("Should not received this error")
		}

		game.StartTurn()

		manager.Process(Event{
			Type:   AttackPlayer,
			Player: p1,
			Payload: CombatPayload{
				GameId:   game.Id.String(),
				Attacker: attacker.Id.String(),
				Defender: player.Id.String(),
			},
		})

		if player.Health != MAX_HEALTH-3 {
			t.Errorf("Expected %v health, got %v", MAX_HEALTH-3, player.Health)
		}
	})
}
