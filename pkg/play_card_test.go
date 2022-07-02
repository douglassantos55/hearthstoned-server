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
	manager := NewGameManager(time.Second)

	// create game
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := manager.CreateGame([]*Socket{p1, p2})

	// get starting hand
	game.ChooseStartingHand(time.Millisecond)

	<-p2.Outgoing             // starting hand
	response := <-p1.Outgoing // starting hand
	payload := response.Payload.(StartingHandPayload)

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	// make sure it can get played
	played := payload.Hand[0].(*Minion)
	played.Mana = 1

	// play a card
	manager.Process(PlayCardEvent(p1, game.Id, played.Id))

	player := game.players[p1]
	if player.hand.Length() != 3 {
		t.Errorf("Expected %v cards in hand, got %v", 3, player.hand.Length())
	}

	select {
	case <-time.After(100 * time.Millisecond):
		t.Error("expected card played")
	case response := <-p1.Outgoing:
		if response.Type != CardPlayed {
			t.Errorf("expected %v, got %v", CardPlayed, response.Type)
		}
		card := response.Payload.(*ActiveMinion)
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
		card := response.Payload.(*ActiveMinion)
		if card.Id != played.Id {
			t.Errorf("expected %v, got %v", played.Id, card.Id)
		}
	}
}

func TestNotEnoughMana(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

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
	played := payload.Hand[0].(*Minion)
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

	card := game.players[p1].hand.Find(played.GetId())
	if card == nil {
		t.Error("Expected card in hand")
	}
}

func TestCardNotFound(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	// create game
	game := NewGame([]*Socket{p1, p2}, time.Second)

	// get starting hands
	game.ChooseStartingHand(time.Millisecond)

	response := <-p1.Outgoing // starting hand

	response = <-p2.Outgoing
	payload := response.Payload.(StartingHandPayload)

	<-p1.Outgoing // start turn
	<-p2.Outgoing // wait turn

	// play a nonexisting card for player
	played := payload.Hand[0].(*Minion)
	played.Mana = 1
	game.PlayCard(played.GetId(), p1)

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
	player := NewPlayer(NewTestSocket())

	player.GainMana(1)
	player.RefillMana()

	card := NewCard("", 1, 1, 1)
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
		board.Place(NewMinion(NewCard("", i, i, i)))
	}

	// try to play another card
	err := board.Place(NewMinion(NewCard("", 1, 1, 1)))

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
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	ability := &Ability{effect: GainManaEffect(2)}
	card := NewSpell("", 1, ability)

	player := game.players[p1]
	player.GainMana(10)
	player.hand.Add(card)

	player.GetMana()
	game.PlayCard(card.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	if player.GetMana() != 2 {
		t.Errorf("Expected %v mana, got %v", 2, player.GetMana())
	}
}

func TestTriggeredSpell(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	ability := &Ability{
		effect: &DrawCard{amount: 1},
		trigger: &Trigger{
			event: TurnStartedEvent,
		},
	}
	spell := NewSpell("", 1, ability)

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
	if player.hand.Length() != 3 {
		t.Errorf("Expected %v cards in hand, got %v", 3, player.hand.Length())
	}
}

func TestMagicFullBoard(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()
	game := NewGame([]*Socket{p1, p2}, time.Second)
	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	// fill board
	player := game.players[p1]
	player.GainMana(10)

	for i := 0; i < 7; i++ {
		minion := NewMinion(NewCard("", i, i, i))
		minion.SetPlayer(player)
		player.board.Place(minion)
	}

	// play magic card
	spell := NewSpell("", 1, &Ability{effect: GainManaEffect(1)})
	player.hand.Add(spell)

	// expect it to cast normally
	game.PlayCard(spell.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	if player.GetMana() != 1 {
		t.Errorf("expected %v mana, got %v", 1, player.GetMana())
	}
}

func TestMinionAbility(t *testing.T) {
	p1 := NewTestSocket()
	p2 := NewTestSocket()
	game := NewGame([]*Socket{p1, p2}, time.Second)

	// create a minion
	minion := NewCard("", 1, 1, 1)

	// give it an ability
	minion.SetAbility(&Ability{
		effect: GainDamageEffect(1),
	})

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
	p1 := NewTestSocket()
	p2 := NewTestSocket()

	game := NewGame([]*Socket{p1, p2}, time.Second)

	// create a minion
	minion := NewCard("", 1, 1, 1)

	// give it an ability
	trigger := &Trigger{
		event: TurnStartedEvent,
		condition: func(card ActiveCard, event GameEvent) bool {
			data := event.GetData().(map[string]interface{})
			player := data["Player"].(*Player)
			minion := card.(*ActiveMinion)
			return player == minion.player
		},
	}

	minion.SetAbility(&Ability{
		trigger: trigger,
		effect:  GainDamageEffect(1),
	})

	game.StartTurn()

	<-p1.Outgoing
	<-p2.Outgoing

	// play the minion
	player := game.players[p1]
	player.hand.Add(minion)

	game.PlayCard(minion.GetId(), p1)

	<-p1.Outgoing
	<-p2.Outgoing

	if minion.GetDamage() != 1 {
		t.Errorf("Expected %v damage, got %v", 1, minion.GetDamage())
	}

	game.EndTurn()

	<-p1.Outgoing // turn
	<-p2.Outgoing // turn

	game.EndTurn()

	<-p1.Outgoing // turn
	<-p2.Outgoing // turn

	<-p1.Outgoing // state changed
	<-p2.Outgoing // state changed

	<-p1.Outgoing // damage gained
	<-p2.Outgoing // damage gained

	// expect ability to be cast
	if minion.GetDamage() != 2 {
		t.Errorf("Expected %v damage, got %v", 2, minion.GetDamage())
	}
}
