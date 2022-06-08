package pkg

import "github.com/google/uuid"

const INITIAL_HAND_LENGTH = 3

type Game struct {
	Id uuid.UUID

	players []*Player
	decks   map[*Player]*Deck
	hands   map[*Player]*Hand
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

func NewGame(players []*Player) *Game {
	decks := map[*Player]*Deck{}

	for _, player := range players {
		// create a deck for each player
		decks[player] = NewDeck()
	}

	return &Game{
		Id: uuid.New(),

		decks:   decks,
		players: players,
		hands:   make(map[*Player]*Hand),
	}
}

func (g *Game) Start() {
	for _, player := range g.players {
		// draw a starting hand for each player
		cards := g.decks[player].Draw(INITIAL_HAND_LENGTH)

		// save hand
		hand := NewHand(cards)
		g.hands[player] = hand

		// return starting hand responses to each player
		go player.Send(StartingHandMessage(g.Id, hand))
	}
}

func (g *Game) Discard(cardId uuid.UUID, player *Player) {
	// find card
	hand := g.hands[player]
	card := hand.Find(cardId)

	if card != nil {
		deck := g.decks[player]

		// remove card from player's hand
		hand.Remove(card)

		// add a new card to hand
		hand.Add(deck.Pop())

		// add card back to player's deck
		deck.Push(card)

		// return wait other players response
		go player.Send(Response{
			Type:    WaitOtherPlayers,
			Payload: hand,
		})
	}
}
