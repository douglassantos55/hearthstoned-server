package pkg

import "github.com/google/uuid"

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
		players := event.Payload.([]*Player)
		g.CreateGame(players)
	case CardDiscarded:
		payload := event.Payload.(CardDiscardedPayload)
		game, ok := g.games[payload.GameId]

		if ok {
			game.Discard(payload.Card, event.Player)
		}
	}
	return nil
}

func (g *GameManager) CreateGame(players []*Player) {
	// create a game
	game := NewGame(players)
	// store it
	g.games[game.Id] = game
	// start it
	game.Start()
}

func (g *GameManager) GameCount() int {
	return len(g.games)
}
