package pkg

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const INITIAL_HAND_LENGTH = 3

type Game struct {
	Id           uuid.UUID
	StopTimer    chan bool
	turnDuration time.Duration
	current      int
	sockets      []*Socket
	ready        []*Player
	mutex        *sync.Mutex
	players      map[*Socket]*Player
	dispatcher   Dispatcher
}

func StartingHandMessage(gameId uuid.UUID, duration time.Duration, hand *Hand) Response {
	return Response{
		Type: StartingHand,
		Payload: StartingHandPayload{
			Duration: duration,
			Hand:     hand.GetCards(),
			GameId:   gameId,
		},
	}
}

func MinionDestroyedMessage(card *ActiveMinion) Response {
	return Response{
		Type:    MinionDestroyed,
		Payload: card,
	}
}

func NewGame(sockets []*Socket, turnDuration time.Duration) *Game {
	dispatcher := NewGameDispatcher()

	players := make(map[*Socket]*Player)
	for _, socket := range sockets {
		player := NewPlayer(socket)
		players[socket] = player

		dispatcher.Subscribe(MinionDamagedEvent, player.NotifyDamage)
		dispatcher.Subscribe(MinionDestroyedEvent, player.NotifyDestroyed)
		dispatcher.Subscribe(CardPlayedEvent, player.NotifyCardPlayed)
		dispatcher.Subscribe(TurnStartedEvent, player.NotifyTurnStarted)
		dispatcher.Subscribe(ManaGainedEvent, player.NotifyManaChanges)
		dispatcher.Subscribe(DamageIncreasedEvent, player.NotifyAttributeChanges)
		dispatcher.Subscribe(PlayerDamagedEvent, player.NotifyPlayerDamage)
		dispatcher.Subscribe(StateChangedEvent, player.NotifyAttributeChanges)
	}

	game := &Game{
		Id:           uuid.New(),
		StopTimer:    make(chan bool),
		turnDuration: turnDuration,
		current:      -1,
		sockets:      sockets,
		players:      players,
		mutex:        new(sync.Mutex),
		dispatcher:   dispatcher,
	}

	dispatcher.Subscribe(CardPlayedEvent, game.HandleAbilities)

	return game
}

func (g *Game) ChooseStartingHand(duration time.Duration) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for _, player := range g.players {
		// draw a starting hand for each player
		player.DrawCards(INITIAL_HAND_LENGTH)

		// return starting hand responses to each player
		go player.Send(StartingHandMessage(g.Id, duration, player.GetHand()))
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
		g.StartTurn()
	}
}

func (g *Game) StartTurn() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	current := g.NextPlayer()

	current.GainMana(1)
	current.RefillMana()

	for _, minion := range current.board.ActivateAll() {
		go g.dispatcher.Dispatch(NewStateChangedEvent(minion))
	}

	current.DrawCards(1)

	go g.StartTimer(g.turnDuration)

	go g.dispatcher.Dispatch(NewTurnStartedEvent(current, g.turnDuration))
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
			Payload: player.GetHand().GetCards(),
		})
	}

	g.mutex.Unlock()

	// if both players are ready, start turns
	if len(g.ready) == len(g.players) {
		g.StopTimer <- true
		g.StartTurn()
	}
}

func (g *Game) EndTurn() {
	g.StopTimer <- true
	g.StartTurn()
}

func (g *Game) PlayCard(cardId uuid.UUID, socket *Socket) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.sockets[g.current] != socket {
		return
	}

	current := g.players[socket]

	// check if card exists on player's hand
	card := current.hand.Find(cardId)
	if card == nil {
		go current.Send(Response{
			Type:    Error,
			Payload: "Card not found in hand",
		})
		return
	}

	// check if player has enough mana to play card
	if current.GetMana() < card.GetMana() {
		go current.Send(Response{
			Type:    Error,
			Payload: "Not enough mana",
		})
		return
	}

	// play card
	minion, err := current.PlayCard(card)
	if err != nil {
		go current.Send(Response{
			Type:    Error,
			Payload: err.Error(),
		})
		return
	}

	// remove card from hand now
	current.Discard(cardId)

	// dispatch card played event
	go g.dispatcher.Dispatch(NewCardPlayedEvent(minion))
}

func (g *Game) HandleAbilities(event GameEvent) bool {
	if card, ok := event.GetData().(Card); ok {
		g.mutex.Lock()
		defer g.mutex.Unlock()

		current := g.players[g.sockets[g.current]]

		if minion, ok := card.(*ActiveMinion); ok {
			if minion.HasAbility() {
				if minion.Trigger != nil {
					go g.dispatcher.Subscribe(minion.Trigger.Event, func(event GameEvent) bool {
						if minion.Trigger.condition == nil || minion.Trigger.condition(minion, event) {
							if event := minion.CastAbility(); event != nil {
								go g.dispatcher.Dispatch(event)
							}
						}
						// until it dies
						return minion.GetHealth() <= 0
					})
				} else {
					if event := minion.CastAbility(); event != nil {
						go g.dispatcher.Dispatch(event)
					}
				}
			}
		} else if spell, ok := card.(*Spell); ok {
			spell.Execute(current)
		} else if spell, ok := card.(*TriggerableSpell); ok {
			go g.dispatcher.Subscribe(spell.Trigger.Event, func(event GameEvent) bool {
				if spell.Trigger.condition == nil || spell.Trigger.condition(spell, event) {
					if event := spell.Execute(current); event != nil {
						go g.dispatcher.Dispatch(event)
					}
				}
				return true
			})
		}
	}
	return false
}

func (g *Game) Attack(attackerId, defenderId uuid.UUID, socket *Socket) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	current := g.players[socket]

	// check if attacker exists in attacking player's board
	if attacker, ok := current.board.GetMinion(attackerId); ok {
		if attacker.CanAttack() {
			// check if defender exists in defending player's board
			if defender, player := g.FindMinion(defenderId); defender != nil {
				// deal damage to defender
				survived := defender.RemoveHealth(attacker.GetDamage())

				// send damage taken message to players
				g.dispatcher.Dispatch(NewDamageEvent(attacker, defender))

				if survived {
					// if it survives, counter-attack
					if defender.CanCounterAttack() {
						attackerSurvived := attacker.RemoveHealth(defender.GetDamage())

						// send damage taken message to players
						g.dispatcher.Dispatch(NewDamageEvent(defender, attacker))

						if !attackerSurvived {
							// remove from its board
							current.board.Remove(attacker)

							// send minion destroyed message to players
							g.dispatcher.Dispatch(NewDestroyedEvent(attacker))
						} else {
							// after attacking, minion gets exhausted
							attacker.SetState(Exhausted{})

							g.dispatcher.Dispatch(NewStateChangedEvent(attacker))
						}
					}
				} else {
					// remove from its board
					player.board.Remove(defender)

					// send minion destroyed message to players
					g.dispatcher.Dispatch(NewDestroyedEvent(defender))

					// after attacking, minion gets exhausted
					attacker.SetState(Exhausted{})

					g.dispatcher.Dispatch(NewStateChangedEvent(attacker))
				}
			}
		}
	}
}

func (g *Game) AttackPlayer(attackerId, playerId uuid.UUID, socket *Socket) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// validate player
	current := g.players[socket]
	if current.Id == playerId {
		go current.Send(Response{
			Type:    Error,
			Payload: "You cannot attack yourself...",
		})
		return false
	}

	// get player
	var player *Player
	for _, p := range g.players {
		if p.Id == playerId {
			player = p
			break
		}
	}
	if player == nil {
		go current.Send(Response{
			Type:    Error,
			Payload: "Player not found",
		})
		return false
	}

	// check if player has minions on board
	if player.board.MinionsCount() == 0 {
		// get minion
		attacker, ok := current.board.GetMinion(attackerId)
		if !ok {
			go current.Send(Response{
				Type:    Error,
				Payload: "Minion not found on board",
			})
			return false
		}

		if attacker.CanAttack() {
			// reduce player's health
			player.ReduceHealth(attacker.GetDamage())

			// send player damage event
			g.dispatcher.Dispatch(NewPlayerDamagedEvent(player, attacker))

			// after attacking, minion gets exhausted
			attacker.SetState(Exhausted{})

			g.dispatcher.Dispatch(NewStateChangedEvent(attacker))

			// check if dead
			if player.GetHealth() <= 0 {
				g.GameOver(current, player)
				return true
			}
		}
	}

	return false
}

func (g *Game) GameOver(winner, loser *Player) {
	g.StopTimer <- true

	// winner gets win message
	go winner.Send(Response{
		Type: Win,
	})

	// loser gets loss message
	go loser.Send(Response{
		Type: Loss,
	})
}

// Searches for a minion on all players board, except current player
func (g *Game) FindMinion(minionId uuid.UUID) (*ActiveMinion, *Player) {
	for _, player := range g.OtherPlayers() {
		if minion, ok := player.board.GetMinion(minionId); ok {
			return minion, player
		}
	}
	return nil, nil
}
