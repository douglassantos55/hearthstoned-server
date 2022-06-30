package pkg

import "testing"

func TestGainManaEvent(t *testing.T) {
	socket := NewTestSocket()
	player := NewPlayer(socket)
	dispatcher := NewGameDispatcher()
	dispatcher.Subscribe(ManaGainedEvent, player.NotifyManaChanges)

	dispatcher.Dispatch(ManaGained{Player: player})

	select {
	case response := <-socket.Outgoing:
		if response.Type != ManaChanged {
			t.Errorf("Expected %v type, got %v", ManaChanged, response.Type)
		}
		if response.Payload.(*Player) != player {
			t.Error("wrong player")
		}
	}
}

func TestAttributeChangedEvent(t *testing.T) {
	socket := NewTestSocket()
	player := NewPlayer(socket)
	minion := NewMinion(NewCard("test", 1, 1, 1))
	minion.SetPlayer(player)
	dispatcher := NewGameDispatcher()

	dispatcher.Subscribe(DamageIncreasedEvent, player.NotifyAttributeChanges)
	dispatcher.Dispatch(&DamageIncreased{Minion: minion})

	select {
	case response := <-socket.Outgoing:
		if response.Type != AttributeChanged {
			t.Errorf("Expected %v type, got %v", AttributeChanged, response.Type)
		}
		if response.Payload.(*ActiveMinion) != minion {
			t.Error("wrong minion")
		}
	}
}
