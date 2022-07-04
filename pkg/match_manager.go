package pkg

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type MatchManager struct {
	mutex     *sync.Mutex
	timeout   time.Duration
	matches   map[uuid.UUID][]*Socket
	confirmed map[uuid.UUID][]*Socket

	StopTimer chan uuid.UUID
}

func NewMatchManager(timeout time.Duration) *MatchManager {
	return &MatchManager{
		timeout:   timeout,
		mutex:     new(sync.Mutex),
		matches:   make(map[uuid.UUID][]*Socket),
		confirmed: make(map[uuid.UUID][]*Socket),

		StopTimer: make(chan uuid.UUID),
	}
}

func ConfirmMessage(matchId uuid.UUID) Response {
	return Response{
		Type:    ConfirmMatch,
		Payload: matchId,
	}
}

func MatchCanceledMessage(matchId uuid.UUID) Response {
	return Response{
		Type:    MatchCanceled,
		Payload: matchId,
	}
}

func WaitOtherPlayersMessage() Response {
	return Response{
		Type: WaitOtherPlayers,
	}
}

func (m *MatchManager) Process(event Event) *Event {
	switch event.Type {
	case CreateMatch:
		players := event.Payload.([]*Socket)
		if len(players) == NUM_OF_PLAYERS {
			m.CreateMatch(players)
		}
	case MatchConfirmed:
		if matchId, err := uuid.Parse(event.Payload.(string)); err == nil {
			return m.ConfirmMatch(matchId, event.Player)
		}
	case MatchDeclined:
		if matchId, err := uuid.Parse(event.Payload.(string)); err == nil {
			return m.CancelMatch(matchId)
		}
	case Disconnected:
		if matchId, ok := m.FindPlayerMatch(event.Player); ok {
			return m.CancelMatch(matchId)
		}
	}
	return nil
}

func (m *MatchManager) CreateMatch(players []*Socket) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// generate an id
	id := uuid.New()

	// save the match
	m.matches[id] = players

	// return response
	for _, player := range players {
		go player.Send(ConfirmMessage(id))
	}

	go m.StartTimer(id)
}

func (m *MatchManager) StartTimer(matchId uuid.UUID) {
	// start timer for match
	timer := time.NewTimer(m.timeout)

	select {
	// when timer ends, cancel match
	case <-timer.C:
		m.CancelMatch(matchId)
	case <-m.StopTimer:
		if !timer.Stop() {
			<-timer.C
		}
	}
}

func (m *MatchManager) CancelMatch(matchId uuid.UUID) *Event {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var event *Event

	// find match
	if match, ok := m.matches[matchId]; ok {
		// send response to players
		for _, player := range match {
			go player.Send(MatchCanceledMessage(matchId))
		}
		// remove match from map
		delete(m.matches, matchId)
	}

	// return queue event for confirmed players
	if confirmed, ok := m.confirmed[matchId]; ok {
		event = &Event{
			Type:   QueueUp,
			Player: confirmed[0],
		}
		// remove confirmed from map
		delete(m.confirmed, matchId)
	}

	return event
}

func (m *MatchManager) ConfirmMatch(matchId uuid.UUID, player *Socket) *Event {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// find match
	if match, ok := m.matches[matchId]; ok {
		// add player as confirmed
		m.confirmed[matchId] = append(m.confirmed[matchId], player)

		// send wait other players response
		go player.Send(WaitOtherPlayersMessage())

		// if both confirmed
		if len(m.confirmed[matchId]) == len(match) {
			// when all players confirmed, stop timer
			m.StopTimer <- matchId

			// remove match
			delete(m.matches, matchId)
			delete(m.confirmed, matchId)

			// return create game event
			return &Event{Type: CreateGame, Payload: match}
		}
	}
	return nil
}

func (m *MatchManager) FindPlayerMatch(player *Socket) (uuid.UUID, bool) {
	for matchId, players := range m.matches {
		for _, socket := range players {
			if socket == player {
				return matchId, true
			}
		}
	}
	return uuid.Nil, false
}

func (m *MatchManager) MatchCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return len(m.matches)
}
