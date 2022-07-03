package pkg

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const INITIAL_HAND_LENGTH = 3

type Timer struct {
	mutex    *sync.Mutex
	timer    *time.Timer
	start    time.Time
	duration time.Duration
}

func NewTimer() *Timer {
	timer := time.NewTimer(time.Second)
	timer.Stop()

	return &Timer{
		timer: timer,
		mutex: new(sync.Mutex),
	}
}

func (t *Timer) Start(duration time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.start = time.Now()
	t.duration = duration
	t.timer.Reset(duration)
}

func (t *Timer) Stop() {
	if !t.timer.Stop() {
		<-t.timer.C
	}
}

func (t *Timer) Done() <-chan time.Time {
	return t.timer.C
}

func (t *Timer) Left() time.Duration {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	ellapsed := time.Since(t.start).Nanoseconds()
	return time.Duration(t.duration.Nanoseconds() - ellapsed)
}

type Game struct {
	Id          uuid.UUID
	StopTimer   chan bool
	Reconnected chan *Socket

	disconnected int
	timer        *Timer
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
		Id:          uuid.New(),
		StopTimer:   make(chan bool),
		Reconnected: make(chan *Socket),

		timer:        NewTimer(),
		turnDuration: turnDuration,
		disconnected: -1,
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
	g.timer.Start(duration)

	select {
	case <-g.StopTimer:
		g.timer.Stop()
	case <-g.timer.Done():
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

func (g *Game) OtherPlayers(current *Socket) []*Player {
	players := []*Player{}
	for _, socket := range g.sockets {
		if socket != current {
			players = append(players, g.players[socket])
		}
	}
	return players
}

func (g *Game) HasPlayer(player *Socket) bool {
	_, exists := g.players[player]
	return exists
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
	played, err := current.PlayCard(card)
	if err != nil {
		go current.Send(Response{
			Type:    Error,
			Payload: err.Error(),
		})
		return
	}

	// remove card from hand now
	current.Discard(cardId)

	// link card and player
	played.SetPlayer(current)

	// dispatch card played event
	go g.dispatcher.Dispatch(NewCardPlayedEvent(played))
}

func (g *Game) HandleAbilities(event GameEvent) bool {
	if card, ok := event.GetData().(ActiveCard); ok {
		g.mutex.Lock()
		defer g.mutex.Unlock()

		if card.HasAbility() {
			ability := card.GetAbility()
			if ability.trigger != nil {
				go g.dispatcher.Subscribe(ability.trigger.event, func(event GameEvent) bool {
					if ability.trigger.condition == nil || ability.trigger.condition(card, event) {
						if event := card.CastAbility(); event != nil {
							go g.dispatcher.Dispatch(event)
						}
					}

					// until it dies
					minion, isMinion := card.(*ActiveMinion)
					return !isMinion || minion.GetHealth() <= 0
				})
			} else {
				if event := card.CastAbility(); event != nil {
					go g.dispatcher.Dispatch(event)
				}
			}
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
	if loser != nil {
		go loser.Send(Response{
			Type: Loss,
		})
	}
}

func (g *Game) Disconnect(player *Socket, duration time.Duration) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for idx, socket := range g.sockets {
		if socket == player {
			g.disconnected = idx
			break
		}
	}

	// start a 30s timer
	timer := time.NewTimer(duration)

	go func() {
		select {
		case <-g.Reconnected:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			var winner *Player
			loser := g.players[player]
			for _, player := range g.players {
				if player != loser {
					winner = player
				}
			}
			g.GameOver(winner, nil)
		}
	}()
}

func (g *Game) Reconnect(socket *Socket) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// stop timer
	g.Reconnected <- socket

	// grab reference to disconnected socket
	disconnected := g.sockets[g.disconnected]

	// replace disconnect player with new player
	g.sockets[g.disconnected] = socket
	g.players[socket] = g.players[disconnected]

	// send the new player game data
	go socket.Send(Response{
		Type: "reconnected",
		Payload: map[string]interface{}{
			"Playing":   g.current == g.disconnected,
			"Time":      g.timer.Left(),
			"Player":    g.players[socket],
			"Opponents": g.OtherPlayers(socket),
		},
	})

	delete(g.players, disconnected)
	g.disconnected = -1
}

// Searches for a minion on all players board, except current player
func (g *Game) FindMinion(minionId uuid.UUID) (*ActiveMinion, *Player) {
	current := g.sockets[g.current]
	for _, player := range g.OtherPlayers(current) {
		if minion, ok := player.board.GetMinion(minionId); ok {
			return minion, player
		}
	}
	return nil, nil
}
