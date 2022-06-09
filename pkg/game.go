package pkg

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const INITIAL_HAND_LENGTH = 3

type Game struct {
	Id uuid.UUID

	current *Player
	mutex   *sync.Mutex
	hands   map[*Socket]*Hand
	players map[*Socket]*Player
}

func StartingHandMessage(gameId uuid.UUID, hand *Hand) Response {
	return Response{
		Type: StartingHand,
		Payload: StartingHandPayload{
			Hand:   hand,
			GameId: gameId,
		},
	}
}

func TurnMessage(responseType ResponseType, gameId uuid.UUID, player *Player) Response {
	return Response{
		Type: responseType,
		Payload: TurnPayload{
			GameId: gameId,
			Mana:   player.GetMana(),
		},
	}
}

func NewGame(sockets []*Socket) *Game {
	var current *Player
	players := make(map[*Socket]*Player)

	for i, socket := range sockets {
		player := NewPlayer(socket)
		players[socket] = player
		if i == 0 {
			current = player
		}
	}

	return &Game{
		Id: uuid.New(),

		current: current,
		players: players,
		mutex:   new(sync.Mutex),
		hands:   make(map[*Socket]*Hand),
	}
}

func (g *Game) ChooseStartingHand(duration time.Duration) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for _, player := range g.players {
		// draw a starting hand for each player
		player.DrawCards(INITIAL_HAND_LENGTH)

		// return starting hand responses to each player
		go player.Send(StartingHandMessage(g.Id, player.GetHand()))

		// start timer
		go g.StartTimer(duration)
	}
}

func (g *Game) StartTimer(duration time.Duration) {
	// start a timer
	timer := time.NewTimer(duration)

	// when it stops, start turns
	select {
	case <-timer.C:
		// change game state to avoid discarding again?
		// start turn
		g.StartTurn()
	}
}

func (g *Game) StartTurn() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// update current player?
	current := g.NextPlayer()

	// draw a new card
	current.DrawCards(1)

	// gain mana
	current.GainMana(1)

	// start timer

	// send start turn to current player
	go current.Send(TurnMessage(StartTurn, g.Id, current))

	// send wait turn to other players
	for _, player := range g.OtherPlayers() {
		go player.Send(TurnMessage(WaitTurn, g.Id, current))
	}
}

func (g *Game) NextPlayer() *Player {
	return g.current
}

func (g *Game) OtherPlayers() []*Player {
	players := []*Player{}
	for _, player := range g.players {
		if player != g.current {
			players = append(players, player)
		}
	}
	return players
}

func (g *Game) Discard(cardId uuid.UUID, socket *Socket) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if player, ok := g.players[socket]; ok {
		// remove card from player's hand
		discarded := player.Discard(cardId)

		// add a new card to hand
		player.DrawCards(1)

		// add card back to player's deck
		player.AddToDeck(discarded)

		// return wait other players response
		go player.Send(Response{
			Type:    WaitOtherPlayers,
			Payload: player.GetHand(),
		})
	}
}
