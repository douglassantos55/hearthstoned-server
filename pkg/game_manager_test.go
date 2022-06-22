package pkg

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func CreateGameEvent(players []*Socket) Event {
	return Event{
		Type:    CreateGame,
		Payload: players,
	}
}

func DiscardCardsEvent(player *Socket, cardIds []string, gameId string) Event {
	return Event{
		Type:   CardDiscarded,
		Player: player,
		Payload: CardDiscardedPayload{
			Cards:  cardIds,
			GameId: gameId,
		},
	}
}

func TestCreateGame(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	manager := NewGameManager()

	// process create game event
	manager.Process(CreateGameEvent([]*Socket{p1, p2}))

	// expect players to receive starting hand response
	select {
	case response := <-p1.Outgoing:
		if response.Type != StartingHand {
			t.Errorf("Expected %v, got %v", StartingHand, response.Type)
		}

		payload := response.Payload.(StartingHandPayload)
		if len(payload.Hand) != INITIAL_HAND_LENGTH {
			t.Errorf("Expected %v, got %v", INITIAL_HAND_LENGTH, len(payload.Hand))
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
		if len(payload.Hand) != INITIAL_HAND_LENGTH {
			t.Errorf("Expected %v, got %v", INITIAL_HAND_LENGTH, len(payload.Hand))
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
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	manager := NewGameManager()

	// create game
	manager.Process(CreateGameEvent([]*Socket{p1, p2}))

	// receive starting hands response
	response := <-p1.Outgoing
	payload := response.Payload.(StartingHandPayload)

	// process discard event on one of the cards
	discarded := payload.Hand[2]
	manager.Process(DiscardCardsEvent(
		p1,
		[]string{discarded.GetId().String()},
		payload.GameId.String(),
	))

	// expect wait other players response
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected wait response")
	case response := <-p1.Outgoing:
		if response.Type != WaitOtherPlayers {
			t.Errorf("Expected %v, got %v", WaitOtherPlayers, response.Type)
		}

		// expect a new card to replace the old one
		newHand := response.Payload.([]Card)
		if len(newHand) != INITIAL_HAND_LENGTH {
			t.Errorf("Expected %v, got %v", INITIAL_HAND_LENGTH, len(newHand))
		}

		for _, card := range newHand {
			if card.GetId() == discarded.GetId() {
				t.Error("Expected card to be discarded")
			}
		}
	}

	// expect card to be added back to the deck
	expected := 60 - INITIAL_HAND_LENGTH
	got := manager.games[payload.GameId].players[p1].deck.cards.Len()
	if got != expected {
		t.Errorf("Expected %v, got %v", expected, got)
	}
}

func TestDiscardTimeout(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	manager := NewGameManager()

	// create and start game
	game := manager.CreateGame([]*Socket{p1, p2})
	game.ChooseStartingHand(100 * time.Millisecond)

	<-p1.Outgoing // starting hand
	<-p2.Outgoing // starting hand

	// wait for timeout
	time.Sleep(110 * time.Millisecond)

	// expect turn to start
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected turn to start")
	case response := <-p1.Outgoing:
		if response.Type != StartTurn {
			t.Errorf("Expected %v, got %v", StartTurn, response.Type)
		}
		payload := response.Payload.(TurnPayload)
		if payload.Mana != 1 {
			t.Errorf("Expected %v, got %v", 1, payload.Mana)
		}
		if payload.CardsInHand != 4 {
			t.Errorf("expected %v, got %v", 4, payload.CardsInHand)
		}
		if len(payload.Cards) != 4 {
			t.Errorf("Expected %v cards, got %v", 4, len(payload.Cards))
		}
		if payload.OpponentId != uuid.Nil {
			t.Errorf("Expected no opponent, got %v", payload.OpponentId)
		}
	}

	// expect other player to receive wait turn
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected wait turn")
	case response := <-p2.Outgoing:
		if response.Type != WaitTurn {
			t.Errorf("Expected %v, got %v", WaitTurn, response.Type)
		}
		payload := response.Payload.(TurnPayload)
		if payload.Mana != 1 {
			t.Errorf("Expected %v, got %v", 1, payload.Mana)
		}
		if payload.CardsInHand != 4 {
			t.Errorf("expected %v, got %v", 4, payload.CardsInHand)
		}
		if len(payload.Cards) != 0 {
			t.Errorf("Expected no cards, got %+v", payload.Cards)
		}
		if payload.OpponentId == uuid.Nil {
			t.Errorf("Expected opponent, got %v", payload.OpponentId)
		}
	}
}

func TestTurnTimer(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := NewGame([]*Socket{p1, p2}, 100*time.Millisecond)
	game.StartTurn()

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	time.Sleep(100 * time.Millisecond)

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected wait turn")
	case response := <-p1.Outgoing:
		if response.Type != WaitTurn {
			t.Errorf("Expected %v, got %v", WaitTurn, response.Type)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected start turn")
	case response := <-p2.Outgoing:
		if response.Type != StartTurn {
			t.Errorf("Expected %v, got %v", StartTurn, response.Type)
		}
	}
}

func TestStartTurnWhenBothReady(t *testing.T) {
	manager := NewGameManager()

	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := manager.CreateGame([]*Socket{p1, p2})
	game.ChooseStartingHand(100 * time.Millisecond)

	<-p1.Outgoing // starting hand
	<-p2.Outgoing // starting hand

	// ready players up without discarding
	manager.Process(DiscardCardsEvent(p1, []string{}, game.Id.String()))
	manager.Process(DiscardCardsEvent(p2, []string{}, game.Id.String()))

	<-p1.Outgoing // wait other players
	<-p2.Outgoing // wait other players

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected start turn")
	case response := <-p1.Outgoing:
		if response.Type != StartTurn {
			t.Errorf("Expecetd %v, got %v", StartTurn, response.Type)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected wait turn")
	case response := <-p2.Outgoing:
		if response.Type != WaitTurn {
			t.Errorf("Expecetd %v, got %v", WaitTurn, response.Type)
		}
	}

	// expect timer to be stopped
	time.Sleep(100 * time.Millisecond)

	select {
	case <-time.After(100 * time.Millisecond):
	case <-p1.Outgoing:
		t.Error("Should not receive response")
	}

	select {
	case <-time.After(100 * time.Millisecond):
	case <-p2.Outgoing:
		t.Error("Should not receive response")
	}
}

func TestPassTurn(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	game.EndTurn()

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("expected wait turn")
	case response := <-p1.Outgoing:
		if response.Type != WaitTurn {
			t.Errorf("Expected %v, got %v", WaitTurn, response.Type)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("expected start turn")
	case response := <-p2.Outgoing:
		if response.Type != StartTurn {
			t.Errorf("Expected %v, got %v", StartTurn, response.Type)
		}
	}
}

func TestRefillsManaOnTurnStart(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	response := <-p1.Outgoing
	payload := response.Payload.(TurnPayload)

	// spend some mana
	payload.Cards[0].(*Minion).Mana = 1
	game.PlayCard(payload.Cards[0].GetId(), p1)

	// end turn
	game.EndTurn()

	// end other player turn
	game.EndTurn()

	// expect mana to be refilled
	player := game.players[game.sockets[game.current]]
	if player.GetMana() != 2 {
		t.Errorf("Expected %v mana, got %v", 2, player.GetMana())
	}
}
