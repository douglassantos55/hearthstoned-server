package pkg

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func PlayCardEvent(player *Socket, gameId, cardId uuid.UUID) Event {
	return Event{
		Type:   PlayCard,
		Player: player,
		Payload: PlayCardPayload{
			CardId: cardId.String(),
			GameId: gameId.String(),
		},
	}
}

func TestPlayCard(t *testing.T) {
	manager := NewGameManager()

	// create game
	p1 := NewSocket()
	p2 := NewSocket()

	game := manager.CreateGame([]*Socket{p1, p2})

	// get starting hand
	game.ChooseStartingHand(time.Millisecond)

	<-p2.Outgoing             // starting hand
	response := <-p1.Outgoing // starting hand
	payload := response.Payload.(StartingHandPayload)

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	// make sure it can get played
	played := payload.Hand.Get(0)
	played.Mana = 1

	// play a card
	manager.Process(PlayCardEvent(p1, game.Id, played.Id))

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("expected card played")
	case response := <-p1.Outgoing:
		if response.Type != CardPlayed {
			t.Errorf("expected %v, got %v", CardPlayed, response.Type)
		}
		card := response.Payload.(*Card)
		if card.Id != played.Id {
			t.Errorf("expected %v, got %v", played.Id, card.Id)
		}
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("expected card played")
	case response := <-p2.Outgoing:
		if response.Type != CardPlayed {
			t.Errorf("expected %v, got %v", CardPlayed, response.Type)
		}
		card := response.Payload.(*Card)
		if card.Id != played.Id {
			t.Errorf("expected %v, got %v", played.Id, card.Id)
		}
	}
}

func TestNotEnoughMana(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()

	// create game
	game := NewGame([]*Socket{p1, p2}, time.Second)

	// get starting hands
	game.ChooseStartingHand(time.Millisecond)

	<-p2.Outgoing // starting hand
	response := <-p1.Outgoing
	payload := response.Payload.(StartingHandPayload)

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	// play a card with mana > 1
	played := payload.Hand.Get(0)
	played.Mana = 5
	game.PlayCard(played.Id, p1)

	// expect error response
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected error message")
	case response := <-p1.Outgoing:
		if response.Type != Error {
			t.Errorf("expecetd %v, got %v", Error, response.Type)
		}
		payload := response.Payload.(string)
		if payload != "Not enough mana" {
			t.Errorf("Expected not enough mana, got %v", payload)
		}
	}
}

func TestCardNotFound(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()

	// create game
	game := NewGame([]*Socket{p1, p2}, time.Second)

	// get starting hands
	game.ChooseStartingHand(time.Millisecond)

	<-p1.Outgoing // starting hand
	response := <-p2.Outgoing
	payload := response.Payload.(StartingHandPayload)

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	// play a nonexisting card for player
	played := payload.Hand.Get(0)
	played.Mana = 1
	game.PlayCard(played.Id, p1)

	// expect error response
	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected error message")
	case response := <-p1.Outgoing:
		if response.Type != Error {
			t.Errorf("expecetd %v, got %v", Error, response.Type)
		}
		payload := response.Payload.(string)
		if payload != "Card not found in hand" {
			t.Errorf("Expected '%v', got '%v'", "Card not found in hand", payload)
		}
	}
}

func TestPlacesOnBoard(t *testing.T) {
	player := NewPlayer(NewSocket())

	player.GainMana(1)
	player.RefillMana()

	card := NewCard(1, 1, 1)
	player.PlayCard(card)

	if player.CardsOnBoardCount() != 1 {
		t.Errorf("Expected %v cards on board, got %v", 1, player.CardsOnBoardCount())
	}

	if player.GetMana() != 0 {
		t.Errorf("Expected %v mana, got %v", 0, player.GetMana())
	}

	// expect minion to start exhausted
	state := player.board.minions[card.Id].GetState()
	if !reflect.DeepEqual(state, Exhausted{}) {
		t.Error("expected minion to start exhausted")
	}
}

func TestFullBoard(t *testing.T) {
	// fill the board
	board := NewBoard()
	for i := 0; i < MAX_MINIONS; i++ {
		board.Place(NewCard(i, i, i))
	}

	// try to play another card
	err := board.Place(NewCard(1, 1, 1))

	// expect error
	if err == nil {
		t.Error("expected error, got nil")
	} else {
		if err.Error() != "Cannot place minion, board is full" {
			t.Errorf("expected '%v', got '%v'", "Cannot place minion, board is full", err.Error())
		}
	}
}
