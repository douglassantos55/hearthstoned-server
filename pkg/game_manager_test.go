package pkg

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func CreateGameEvent(players []*Player) Event {
	return Event{
		Type:    CreateGame,
		Payload: players,
	}
}

func DiscardCardEvent(player *Player, cardId, gameId uuid.UUID) Event {
	return Event{
		Type:   CardDiscarded,
		Player: player,
		Payload: CardDiscardedPayload{
			Card:   cardId,
			GameId: gameId,
		},
	}
}

func TestCreateGame(t *testing.T) {
	p1 := NewPlayer()
	p2 := NewPlayer()

	manager := NewGameManager()

	// process create game event
	manager.Process(CreateGameEvent([]*Player{p1, p2}))

	// expect players to receive starting hand response
	select {
	case response := <-p1.Outgoing:
		if response.Type != StartingHand {
			t.Errorf("Expected %v, got %v", StartingHand, response.Type)
		}

		payload := response.Payload.(StartingHandPayload)
		if payload.Hand.Length() != INITIAL_HAND_LENGTH {
			t.Errorf("Expected %v, got %v", INITIAL_HAND_LENGTH, payload.Hand.Length())
		}
		if payload.GameId == uuid.Nil {
			t.Errorf("Expected game id, got %v", payload.GameId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected start game response")
	}

	select {
	case response := <-p2.Outgoing:
		if response.Type != StartingHand {
			t.Errorf("Expected %v, got %v", StartingHand, response.Type)
		}

		payload := response.Payload.(StartingHandPayload)
		if payload.Hand.Length() != INITIAL_HAND_LENGTH {
			t.Errorf("Expected %v, got %v", INITIAL_HAND_LENGTH, payload.Hand.Length())
		}
		if payload.GameId == uuid.Nil {
			t.Errorf("Expected game id, got %v", payload.GameId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected start game response")
	}

	// expect game to be stored
	if manager.GameCount() != 1 {
		t.Errorf("Expected %v, got %v", 1, manager.GameCount())
	}
}

func TestDiscardStartingHand(t *testing.T) {
	p1 := NewPlayer()
	p2 := NewPlayer()

	manager := NewGameManager()

	// create game
	manager.Process(CreateGameEvent([]*Player{p1, p2}))

	// receive starting hands response
	response := <-p1.Outgoing
	payload := response.Payload.(StartingHandPayload)

	// process discard event on one of the cards
	discarded := payload.Hand.Get(2)
	manager.Process(DiscardCardEvent(p1, discarded.Id, payload.GameId))

	// expect wait other players response
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected wait response")
	case response := <-p1.Outgoing:
		if response.Type != WaitOtherPlayers {
			t.Errorf("Expected %v, got %v", WaitOtherPlayers, response.Type)
		}

		// expect a new card to replace the old one
		newHand := response.Payload.(*Hand)
		if newHand.Length() != INITIAL_HAND_LENGTH {
			t.Errorf("Expected %v, got %v", INITIAL_HAND_LENGTH, newHand.Length())
		}

		if newHand.Find(discarded.Id) != nil {
			t.Error("Expected card to be discarded")
		}
	}

	// expect card to be added back to the deck
	expected := 60 - INITIAL_HAND_LENGTH
	got := manager.games[payload.GameId].decks[p1].cards.Len()
	if got != expected {
		t.Errorf("Expected %v, got %v", expected, got)
	}
}
