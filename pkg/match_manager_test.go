package pkg

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func CreateMatchEvent(players []*Socket) Event {
	return Event{
		Type:    CreateMatch,
		Payload: players,
	}
}

func MatchConfirmedEvent(player *Socket, matchId uuid.UUID) Event {
	return Event{
		Type:    MatchConfirmed,
		Player:  player,
		Payload: matchId.String(),
	}
}

func MatchDeclinedEvent(player *Socket, matchId uuid.UUID) Event {
	return Event{
		Type:    MatchDeclined,
		Player:  player,
		Payload: matchId.String(),
	}
}

func TestMatchManager(t *testing.T) {
	t.Run("creates match", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewMatchManager(2 * time.Second)

		manager.Process(CreateMatchEvent([]*Socket{p1, p2}))

		if manager.MatchCount() != 1 {
			t.Errorf("Expected %v, got %v", 1, manager.MatchCount())
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected confirm match response")
		case response := <-p1.Outgoing:
			if response.Type != ConfirmMatch {
				t.Errorf("Expected %v, got %v", ConfirmMatch, response.Type)
			}

			if response.Payload == nil {
				t.Error("Expected match id")
			}
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected confirm match response")
		case response := <-p2.Outgoing:
			if response.Type != ConfirmMatch {
				t.Errorf("Expected %v, got %v", ConfirmMatch, response.Type)
			}

			if response.Payload == nil {
				t.Error("Expected match id")
			}
		}
	})

	t.Run("timeout", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewMatchManager(100 * time.Millisecond)

		// create a match
		manager.CreateMatch([]*Socket{p1, p2})

		<-p1.Outgoing // confirm_match
		<-p2.Outgoing // confirm_match

		// wait timer to run out
		time.Sleep(100 * time.Millisecond)

		// expect users to receive match canceled response
		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected match canceled response")
		case response := <-p1.Outgoing:
			if response.Type != MatchCanceled {
				t.Errorf("Expected %v, got %v", MatchCanceled, response.Type)
			}

			if response.Payload == nil {
				t.Error("Expected match id")
			}
		}

		select {
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected match canceled response")
		case response := <-p2.Outgoing:
			if response.Type != MatchCanceled {
				t.Errorf("Expected %v, got %v", MatchCanceled, response.Type)
			}

			if response.Payload == nil {
				t.Error("Expected match id")
			}
		}

		// expect match to be removed
		if manager.MatchCount() != 0 {
			t.Errorf("Expected %v, got %v", 0, manager.MatchCount())
		}
	})

	t.Run("starts game", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()
		manager := NewMatchManager(100 * time.Millisecond)

		// create a match
		manager.CreateMatch([]*Socket{p1, p2})

		response := <-p1.Outgoing
		<-p2.Outgoing

		// both players confirm match
		matchId := response.Payload.(uuid.UUID)
		manager.Process(MatchConfirmedEvent(p1, matchId))
		event := manager.Process(MatchConfirmedEvent(p2, matchId))

		<-p1.Outgoing // wait other players
		<-p2.Outgoing // wait other players

		// expect create game event to be returned
		if event.Type != CreateGame {
			t.Errorf("Expected %v, got %v", CreateGame, event.Type)
		}

		// expect match to be removed
		if manager.MatchCount() != 0 {
			t.Errorf("Expected %v, got %v", 0, manager.MatchCount())
		}

		// expect timer to be stoped
		time.Sleep(100 * time.Millisecond)

		select {
		case <-time.After(100 * time.Millisecond):
		case response := <-p1.Outgoing:
			t.Errorf("Did not expect response, got %v", response)
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case response := <-p2.Outgoing:
			t.Errorf("Did not expect response, got %v", response)
		}
	})

	t.Run("declines match", func(t *testing.T) {
		manager := NewMatchManager(time.Second)

		p1 := NewTestSocket()
		p2 := NewTestSocket()

		// create a match
		manager.CreateMatch([]*Socket{p1, p2})

		response := <-p1.Outgoing // confirm match
		<-p2.Outgoing             // confirm match

		matchId := response.Payload.(uuid.UUID)

		// accept with one player
		manager.Process(MatchConfirmedEvent(p1, matchId))

		<-p1.Outgoing // wait other players

		// decline with another
		event := manager.Process(MatchDeclinedEvent(p2, matchId))

		// expect players to receive match canceled response
		select {
		case response := <-p1.Outgoing:
			if response.Type != MatchCanceled {
				t.Errorf("Expected %v, got %v", MatchCanceled, response.Type)
			}
		case <-time.After(100 * time.Millisecond):
		}

		select {
		case response := <-p2.Outgoing:
			if response.Type != MatchCanceled {
				t.Errorf("Expected %v, got %v", MatchCanceled, response.Type)
			}
		case <-time.After(100 * time.Millisecond):
		}

		// expect queue event for confirmed player
		if event.Type != QueueUp {
			t.Errorf("Expected %v, got %v", QueueUp, event.Type)
		}

		// expect match to be removed
		if manager.MatchCount() != 0 {
			t.Errorf("Expected %v, got %v", 0, manager.MatchCount())
		}
	})

	t.Run("disconnected", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewMatchManager(100 * time.Millisecond)

		// create a match
		manager.CreateMatch([]*Socket{p1, p2})

		// disconnect
		manager.Process(NewDisconnected(p1))

		// check if there are no matches
		if manager.MatchCount() != 0 {
			t.Errorf("Expected no matches, got %v", manager.MatchCount())
		}
	})

	t.Run("confirmed disconnected", func(t *testing.T) {
		p1 := NewTestSocket()
		p2 := NewTestSocket()

		manager := NewMatchManager(100 * time.Millisecond)

		// create a match
		manager.CreateMatch([]*Socket{p1, p2})

		// confirm match
		response := <-p1.Outgoing
		matchId := response.Payload.(uuid.UUID)
		manager.ConfirmMatch(matchId, p2)

		// disconnect
		event := manager.Process(NewDisconnected(p1))

		// check message for other player
		if event.Type != QueueUp {
			t.Errorf("Expected %v event, got %v", QueueUp, event.Type)
		}

		if manager.MatchCount() != 0 {
			t.Errorf("Expected no matches, got %v", manager.MatchCount())
		}
	})
}
