package pkg

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Type    EventType   `json:"type"`
	Player  *Socket     `json:"player"`
	Payload interface{} `json:"payload"`
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
	Type    ResponseType `json:"type"`
	Payload interface{}  `json:"payload"`
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
	GameId   uuid.UUID     `json:"game_id"`
	Duration time.Duration `json:"duration"`
	Hand     []Card        `json:"hand"`
}

type CardDiscardedPayload struct {
	GameId string
	Cards  []string
}

type TurnPayload struct {
	GameId      uuid.UUID     `json:"game_id"`
	Mana        int           `json:"mana"`
	Duration    time.Duration `json:"duration"`
	CardsInHand int           `json:"cards_in_hand"`
	Cards       []Card        `json:"cards"`
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
