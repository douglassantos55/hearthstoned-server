package pkg

import "github.com/google/uuid"

type Event struct {
	Type    EventType
	Player  *Player
	Payload interface{}
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
)

type Response struct {
	Type    ResponseType
	Payload interface{}
}

type ResponseType string

const (
	Success          ResponseType = "success"
	WaitForMatch     ResponseType = "wait_for_match"
	ConfirmMatch     ResponseType = "confirm_match"
	MatchCanceled    ResponseType = "match_canceled"
	WaitOtherPlayers ResponseType = "wait_other_players"
	StartingHand     ResponseType = "starting_hand"
)

type StartingHandPayload struct {
	GameId uuid.UUID
	Hand   *Hand
}

type CardDiscardedPayload struct {
	GameId uuid.UUID
	Card   uuid.UUID
}
