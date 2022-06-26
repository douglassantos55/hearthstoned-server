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

		<-p1.Outgoing // attribute changed
		<-p2.Outgoing // attribute changed

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
			payload := response.Payload.(MinionDamagedPayload)
			if payload.Defender.Id != defender.Id {
				t.Errorf("Expected %v defender, got %v", defender.Id, payload.Defender.Id)
			}
			if payload.Defender.Health != 1 {
				t.Errorf("Expecetd %v health, got %v", 1, payload.Defender.Health)
			}
			if payload.Attacker.Id != attacker.Id {
				t.Errorf("Expected %v attacker, got %v", attacker.Id, payload.Attacker.Id)
			}
		}

		select {
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected damage taken response")
		case response := <-p2.Outgoing:
			if response.Type != MinionDamageTaken {
				t.Errorf("Expected %v, got %v", MinionDamageTaken, response.Type)
			}
			payload := response.Payload.(MinionDamagedPayload)
			if payload.Defender.Id != defender.Id {
				t.Errorf("Expected %v defender, got %v", defender.Id, payload.Defender.Id)
			}
			if payload.Defender.Health != 1 {
				t.Errorf("Expecetd %v health, got %v", 1, payload.Defender.Health)
			}
			if payload.Attacker.Id != attacker.Id {
				t.Errorf("Expected %v attacker, got %v", attacker.Id, payload.Attacker.Id)
			}
		}

		select {
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected minion destroyed response")
		case response := <-p1.Outgoing:
			if response.Type != MinionDestroyed {
				t.Errorf("Expected %v, got %v", MinionDestroyed, response.Type)
			}
			minion := response.Payload.(*ActiveMinion)
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
			minion := response.Payload.(*ActiveMinion)
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

		<-p1.Outgoing // start turn
		<-p2.Outgoing // wait turn

		manager.Process(Event{
			Type:   AttackPlayer,
			Player: p1,
			Payload: CombatPayload{
				GameId:   game.Id.String(),
				Attacker: attacker.Id.String(),
				Defender: player.Id.String(),
			},
		})

		<-p1.Outgoing // start turn
		<-p2.Outgoing // wait turn

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected response")
		case response := <-p1.Outgoing:
			if response.Type != PlayerDamageTaken {
				t.Errorf("Expected %v type, got %v", PlayerDamageTaken, response.Type)
			}
			payload := response.Payload.(PlayerDamagedPayload)
			if payload.Player.Id != player.Id {
				t.Errorf("Expected %v player, got %v", player.Id, payload.Player.Id)
			}
			if payload.Attacker.Id != attacker.Id {
				t.Errorf("Expected %v attacker, got %v", attacker.Id, payload.Attacker.Id)
			}
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected response")
		case response := <-p2.Outgoing:
			if response.Type != PlayerDamageTaken {
				t.Errorf("Expected %v type, got %v", PlayerDamageTaken, response.Type)
			}
			payload := response.Payload.(PlayerDamagedPayload)
			if payload.Player.Id != player.Id {
				t.Errorf("Expected %v player, got %v", player.Id, payload.Player.Id)
			}
			if payload.Attacker.Id != attacker.Id {
				t.Errorf("Expected %v attacker, got %v", attacker.Id, payload.Attacker.Id)
			}
		}

		if player.Health != MAX_HEALTH-3 {
			t.Errorf("Expected %v health, got %v", MAX_HEALTH-3, player.Health)
		}
	})

	t.Run("attack player with minions on board", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewGameManager()
		game := manager.CreateGame([]*Socket{p1, p2})

		player := game.players[p2]

		attacker := NewCard("", 1, 3, 1)
		game.players[p1].PlayCard(attacker)

		defender := NewCard("", 1, 1, 3)
		game.players[p2].PlayCard(defender)

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

		if player.Health != MAX_HEALTH {
			t.Errorf("Expected %v health, got %v", MAX_HEALTH, player.Health)
		}
	})

	t.Run("game over", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewGameManager()
		game := manager.CreateGame([]*Socket{p1, p2})

		player := game.players[p2]

		attacker := NewCard("", 1, 30, 1)
		_, err := game.players[p1].PlayCard(attacker)

		if err != nil {
			t.Error("Should not received this error")
		}

		game.StartTurn()

		<-p1.Outgoing // turn
		<-p2.Outgoing // turn

		<-p1.Outgoing // attribute changed
		<-p2.Outgoing // attribute changed

		manager.Process(Event{
			Type:   AttackPlayer,
			Player: p1,
			Payload: CombatPayload{
				GameId:   game.Id.String(),
				Attacker: attacker.Id.String(),
				Defender: player.Id.String(),
			},
		})

		<-p1.Outgoing // damage taken
		<-p2.Outgoing // damage taken

		<-p1.Outgoing // attribute changed
		<-p2.Outgoing // attribute changed

		if player.Health != 0 {
			t.Errorf("Expected %v health, got %v", 0, player.Health)
		}

		if _, ok := manager.games[game.Id]; ok {
			t.Errorf("Expected game to be removed, got %v", manager.games)
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected game over response")
		case response := <-p1.Outgoing:
			if response.Type != Win {
				t.Errorf("Expected %v, got %v", Win, response.Type)
			}
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected game over response")
		case response := <-p2.Outgoing:
			if response.Type != Loss {
				t.Errorf("Expected %v, got %v", Loss, response.Type)
			}
		}
	})
}
