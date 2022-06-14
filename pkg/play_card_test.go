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
	played := payload.Hand.Get(0).(*Minion)
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
		card := response.Payload.(*Minion)
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
		card := response.Payload.(*Minion)
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
	played := payload.Hand.Get(0).(*Minion)
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
	played := payload.Hand.Get(0).(*Minion)
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

func TestMagicCard(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	ability := GainManaAbility(2)
	card := NewSpell(1, ability)

	player := game.players[p1]
	player.hand.Add(card)

	game.PlayCard(card.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	if player.GetTotalMana() != 3 {
		t.Errorf("Expected %v mana, got %v", 3, player.GetTotalMana())
	}
}

func TestTriggeredSpell(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	ability := GainManaAbility(1)
	spell := Trigerable(TurnStartedEvent, NewSpell(1, ability))

	// play a magic card with triggered spell
	player := game.players[p1]
	player.hand.Add(spell)

	game.PlayCard(spell.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	game.EndTurn() // p1 end turn

	<-p1.Outgoing
	<-p2.Outgoing

	game.EndTurn() // p2 end turn

	<-p1.Outgoing
	<-p2.Outgoing

	// expect spell to cast
	if player.GetMana() != 3 {
		t.Errorf("Expected %v mana, got %v", 3, player.GetMana())
	}
}

func TestTriggeredSpellOnce(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	ability := GainManaAbility(1)
	spell := Trigerable(TurnStartedEvent, NewSpell(1, ability))

	// play a magic card with triggered spell
	player := game.players[p1]
	player.hand.Add(spell)
	game.PlayCard(spell.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	game.EndTurn() // p1 end turn

	<-p1.Outgoing
	<-p2.Outgoing

	game.EndTurn() // p2 end turn

	<-p1.Outgoing
	<-p2.Outgoing

	game.EndTurn() // p1 end turn

	<-p1.Outgoing
	<-p2.Outgoing

	game.EndTurn() // p2 end turn

	<-p1.Outgoing
	<-p2.Outgoing

	// expect spell to cast
	if player.GetMana() != 4 {
		t.Errorf("Expected %v mana, got %v", 4, player.GetMana())
	}
}

func TestMagicFullBoard(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()
	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	// fill board
	player := game.players[p1]
	for i := 0; i < 7; i++ {
		player.board.Place(NewCard(i, i, i))
	}

	// play magic card
	spell := NewSpell(1, GainManaAbility(1))
	player.hand.Add(spell)

	// expect it to cast normally
	game.PlayCard(spell.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	if player.GetTotalMana() != 2 {
		t.Errorf("expected %v mana, got %v", 2, player.GetTotalMana())
	}
}

func TestMinionAbility(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()
	game := NewGame([]*Socket{p1, p2}, time.Second)

	// create a minion
	minion := NewCard(1, 1, 1)

	// give it an ability
	minion.SetAbility(GainDamageAbility(1))

	// play the minion
	player := game.players[p1]
	player.hand.Add(minion)

	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	game.PlayCard(minion.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	// expect ability to be cast
	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}

func TestTriggerableMinionAbility(t *testing.T) {
	p1 := NewSocket()
	p2 := NewSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)

	// create a minion
	minion := NewCard(1, 1, 1)

	// give it an ability
    minion.SetTrigger(TurnStartedEvent)
	minion.SetAbility(GainDamageAbility(1))

	// play the minion
	player := game.players[p1]
	player.hand.Add(minion)

	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	game.PlayCard(minion.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	// expect ability to be cast
	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}
