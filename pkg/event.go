package pkg

import "github.com/google/uuid"

type Event struct {
	Type    EventType
	Player  *Socket
	Payload interface{}
}

type EventHandler interface {
	Process(event Event) *Event
}

type EventType string

const (
	QueueUp        EventType = "queue"
	Dequeue        EventType = "dequeue"
	CreateMatch    EventType = "create_match"
	MatchConfirmed EventType = "match_confirmed"
	MatchDeclined  EventType = "match_declined"
	CreateGame     EventType = "create_game"
	CardDiscarded  EventType = "card_discarded"
	EndTurn        EventType = "end_turn"
	PlayCard       EventType = "play_card"
	Attack         EventType = "attack"
)

type Response struct {
	Type    ResponseType
	Payload interface{}
}

type ResponseType string

const (
	Error             ResponseType = "error"
	Success           ResponseType = "success"
	WaitForMatch      ResponseType = "wait_for_match"
	ConfirmMatch      ResponseType = "confirm_match"
	MatchCanceled     ResponseType = "match_canceled"
	WaitOtherPlayers  ResponseType = "wait_other_players"
	StartingHand      ResponseType = "starting_hand"
	StartTurn         ResponseType = "start_turn"
	WaitTurn          ResponseType = "wait_turn"
	CardPlayed        ResponseType = "card_played"
	MinionDamageTaken ResponseType = "minion_taken_damage"
	MinionDestroyed   ResponseType = "minion_destroyed"
	ManaChanged       ResponseType = "mana_changed"
	AttributeChanged  ResponseType = "attribute_changed"
)

type StartingHandPayload struct {
	GameId uuid.UUID
	Hand   *Hand
}

type CardDiscardedPayload struct {
	GameId string
	Cards  []string
}

type TurnPayload struct {
	GameId      uuid.UUID
	Mana        int
	CardsInHand int
	Cards       []Card
}

type PlayCardPayload struct {
	GameId string
	CardId string
}

type CombatPayload struct {
	GameId   string
	Attacker string
	Defender string
}
