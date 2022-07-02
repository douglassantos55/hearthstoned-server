package pkg

import (
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

type GameManager struct {
	games      map[uuid.UUID]*Game
	disconnect time.Duration
}

func NewGameManager(duration time.Duration) *GameManager {
	return &GameManager{
		disconnect: duration,
		games:      make(map[uuid.UUID]*Game),
	}
}

func (g *GameManager) Process(event Event) *Event {
	switch event.Type {
	case CreateGame:
		players := event.Payload.([]*Socket)
		game := g.CreateGame(players)
		game.ChooseStartingHand(30 * time.Second)
	case CardDiscarded:
		var payload CardDiscardedPayload

		if err := mapstructure.Decode(event.Payload, &payload); err == nil {
			if gameId, err := uuid.Parse(payload.GameId); err == nil {
				cards := []uuid.UUID{}
				for _, cardId := range payload.Cards {
					if uuid, err := uuid.Parse(cardId); err == nil {
						cards = append(cards, uuid)
					}
				}
				if game, ok := g.games[gameId]; ok {
					game.Discard(cards, event.Player)
				}
			}
		}
	case EndTurn:
		if gameId, err := uuid.Parse(event.Payload.(string)); err == nil {
			if game, ok := g.games[gameId]; ok {
				game.EndTurn()
			}
		}
	case PlayCard:
		var payload PlayCardPayload
		if err := mapstructure.Decode(event.Payload, &payload); err == nil {
			if gameId, err := uuid.Parse(payload.GameId); err == nil {
				if game, ok := g.games[gameId]; ok {
					if cardId, err := uuid.Parse(payload.CardId); err == nil {
						game.PlayCard(cardId, event.Player)
					}
				}
			}
		}
	case Attack:
		var payload CombatPayload
		if err := mapstructure.Decode(event.Payload, &payload); err == nil {
			if gameId, err := uuid.Parse(payload.GameId); err == nil {
				if attacker, err := uuid.Parse(payload.Attacker); err == nil {
					if defender, err := uuid.Parse(payload.Defender); err == nil {
						if game, ok := g.games[gameId]; ok {
							game.Attack(attacker, defender, event.Player)
						}
					}
				}
			}
		}
	case AttackPlayer:
		var payload CombatPayload
		if err := mapstructure.Decode(event.Payload, &payload); err == nil {
			if gameId, err := uuid.Parse(payload.GameId); err == nil {
				if attacker, err := uuid.Parse(payload.Attacker); err == nil {
					if defender, err := uuid.Parse(payload.Defender); err == nil {
						if game, ok := g.games[gameId]; ok {
							if game.AttackPlayer(attacker, defender, event.Player) {
								delete(g.games, gameId)
							}
						}
					}
				}
			}
		}
	case Disconnected:
		// check if disconnected player is playing
		if game := g.FindPlayerGame(event.Player); game != nil {
			game.Disconnect(event.Player, g.disconnect)

		}
	case Reconnected:
		// find game
		if gameId, err := uuid.Parse(event.Payload.(string)); err == nil {
			if game, ok := g.games[gameId]; ok {
				game.Reconnect(event.Player)
			}
		}
	}
	return nil
}

func (g *GameManager) CreateGame(players []*Socket) *Game {
	game := NewGame(players, 75*time.Second)
	g.games[game.Id] = game

	return game
}

func (g *GameManager) FindPlayerGame(player *Socket) *Game {
	for _, game := range g.games {
		if game.HasPlayer(player) {
			return game
		}
	}
	return nil
}

func (g *GameManager) GameCount() int {
	return len(g.games)
}
