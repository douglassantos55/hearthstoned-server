package pkg

import (
	"testing"
	"time"
)

func QueueUpEvent(player *Socket) Event {
	return Event{
		Type:   QueueUp,
		Player: player,
	}
}

func DequeueEvent(player *Socket) Event {
	return Event{
		Type:   Dequeue,
		Player: player,
	}
}

func TestQueueManager(t *testing.T) {
	t.Run("queues player", func(t *testing.T) {
		player := NewTestSocket()
		manager := NewQueueManager()

		// process a queue up event
		manager.Process(QueueUpEvent(player))

		// expect manager to send wait message to player
		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Did not receive wait message")
		case response := <-player.Outgoing:
			if response.Type != WaitForMatch {
				t.Errorf("Expected %v, got %v", WaitForMatch, response.Type)
			}
		}

		// expect manager to queue player
		got := manager.InQueueCount()
		if got != 1 {
			t.Errorf("Expected %v, got %v", 1, got)
		}
	})

	t.Run("ignores other events", func(t *testing.T) {
		player := NewTestSocket()
		manager := NewQueueManager()

		// process an invalid event type
		go manager.Process(Event{Type: "play_sound", Player: player})

		// expect player to not receive anything
		select {
		case <-time.After(500 * time.Millisecond):
		case <-player.Outgoing:
			t.Error("Did not expect player to receive response")
		}

		// expect queue to remain empty
		if manager.InQueueCount() != 0 {
			t.Error("Queue should be empty")
		}
	})

	t.Run("match found", func(t *testing.T) {
		manager := NewQueueManager()

		// queue two players
		manager.Process(QueueUpEvent(NewTestSocket()))
		event := manager.Process(QueueUpEvent(NewTestSocket()))

		// expect create match event from manager
		if event.Type != CreateMatch {
			t.Errorf("Expected %v, got %v", CreateMatch, event.Type)
		}

		// expected create match event to have players
		players := event.Payload.([]*Socket)
		if len(players) != NUM_OF_PLAYERS {
			t.Errorf("Expected %v, got %v", NUM_OF_PLAYERS, len(players))
		}

		// expect queue to be empty
		count := manager.InQueueCount()
		if count != 0 {
			t.Errorf("Expected %v, got %v", 0, count)
		}
	})

	t.Run("dequeues players", func(t *testing.T) {
		player := NewTestSocket()
		manager := NewQueueManager()

		// queue a player
		manager.AddToQueue(player)

		<-player.Outgoing // skip WaitForMatch response

		// process dequeue event for that player
		manager.Process(DequeueEvent(player))

		// expect player to not be on queue anymore
		count := manager.InQueueCount()
		if count != 0 {
			t.Errorf("Expected %v, got %v", 0, count)
		}

		// expect player to receive confirmation of dequeue
		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected confirmation of dequeue")
		case response := <-player.Outgoing:
			if response.Type != Success {
				t.Errorf("Expected %v, got %v", Success, response.Type)
			}
		}
	})

	t.Run("disconnected", func(t *testing.T) {
		player := NewTestSocket()
		manager := NewQueueManager()

		//queue
		manager.AddToQueue(player)

		// disconnect
		manager.Process(NewDisconnected(player))

		// chekc if socket is removed from queue
		if manager.InQueueCount() != 0 {
			t.Errorf("Expected empty queue, got %v", manager.InQueueCount())
		}
	})
}
