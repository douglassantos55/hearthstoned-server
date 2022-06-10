package pkg

import (
	"time"

	"github.com/google/uuid"
)

type GameManager struct {
	games map[uuid.UUID]*Game
}

func NewGameManager() *GameManager {
	return &GameManager{
		games: make(map[uuid.UUID]*Game),
	}
}

func (g *GameManager) Process(event Event) *Event {
	switch event.Type {
	case CreateGame:
		players := event.Payload.([]*Socket)
		game := g.CreateGame(players)
		game.ChooseStartingHand(30 * time.Second)
	case CardDiscarded:
		payload := event.Payload.(CardDiscardedPayload)

		if game, ok := g.games[payload.GameId]; ok {
			game.Discard(payload.Cards, event.Player)
		}
	case EndTurn:
		gameId := event.Payload.(uuid.UUID)
		if game, ok := g.games[gameId]; ok {
			game.EndTurn()
		}
	}
	return nil
}

func (g *GameManager) CreateGame(players []*Socket) *Game {
	game := NewGame(players, 75*time.Second)
	g.games[game.Id] = game

	return game
}

func (g *GameManager) GameCount() int {
	return len(g.games)
}
