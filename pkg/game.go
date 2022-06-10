package pkg

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const INITIAL_HAND_LENGTH = 3

type Game struct {
	Id uuid.UUID

	StopTimer    chan bool
	turnDuration time.Duration

	current int
	sockets []*Socket
	ready   []*Player
	mutex   *sync.Mutex
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
			GameId:      gameId,
			Mana:        player.GetMana(),
			CardsInHand: player.GetHand().Length(),
		},
	}
}

func NewGame(sockets []*Socket, turnDuration time.Duration) *Game {
	players := make(map[*Socket]*Player)
	for _, socket := range sockets {
		player := NewPlayer(socket)
		players[socket] = player
	}

	return &Game{
		Id: uuid.New(),

		StopTimer:    make(chan bool),
		turnDuration: turnDuration,

		current: -1,
		sockets: sockets,
		players: players,
		mutex:   new(sync.Mutex),
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
	}

	// start timer
	go g.StartTimer(duration)
}

func (g *Game) StartTimer(duration time.Duration) {
	timer := time.NewTimer(duration)

	select {
	case <-g.StopTimer:
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		// change game state to avoid discarding again?
		g.StartTurn(duration)
	}
}

func (g *Game) StartTurn(duration time.Duration) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	current := g.NextPlayer()

	current.GainMana(1)
	cards := current.DrawCards(1)

	go g.StartTimer(duration)

	go current.Send(Response{
		Type: StartTurn,
		Payload: TurnPayload{
			Mana:        current.GetMana(),
			Cards:       cards,
			GameId:      g.Id,
			CardsInHand: current.GetHand().Length(),
		},
	})

	for _, player := range g.OtherPlayers() {
		go player.Send(Response{
			Type: WaitTurn,
			Payload: TurnPayload{
				Mana:        current.GetMana(),
				GameId:      g.Id,
				CardsInHand: current.GetHand().Length(),
			},
		})
	}
}

func (g *Game) NextPlayer() *Player {
	g.current++
	if g.current >= len(g.sockets) {
		g.current = 0
	}
	return g.players[g.sockets[g.current]]
}

func (g *Game) OtherPlayers() []*Player {
	players := []*Player{}
	for _, socket := range g.sockets {
		if socket != g.sockets[g.current] {
			players = append(players, g.players[socket])
		}
	}
	return players
}

func (g *Game) Discard(cardIds []uuid.UUID, socket *Socket) {
	g.mutex.Lock()

	if player, ok := g.players[socket]; ok {
		for _, cardId := range cardIds {
			// remove card from player's hand
			discarded := player.Discard(cardId)

			// add a new card to hand
			player.DrawCards(1)

			// add card back to player's deck
			player.AddToDeck(discarded)
		}

		// mark player as ready
		g.ready = append(g.ready, player)

		// return wait other players response
		go player.Send(Response{
			Type:    WaitOtherPlayers,
			Payload: player.GetHand(),
		})
	}

	g.mutex.Unlock()

	// if both players are ready, start turns
	if len(g.ready) == len(g.players) {
		g.StopTimer <- true
		g.StartTurn(g.turnDuration)
	}
}

func (g *Game) EndTurn() {
	g.StopTimer <- true
	g.StartTurn(g.turnDuration)
}
