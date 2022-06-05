package pkg

import (
	"sync"
)

const NUM_OF_PLAYERS = 2

type QueueManager struct {
	queue *Queue
	mutex *sync.Mutex
}

func WaitForMatchMessage() Response {
	return Response{
		Type: WaitForMatch,
	}
}

func NewQueueManager() *QueueManager {
	return &QueueManager{
		queue: NewQueue(),
		mutex: new(sync.Mutex),
	}
}

func (q *QueueManager) Process(event Event) *Event {
	switch event.Type {
	case QueueUp:
		q.AddToQueue(event.Player)

		if q.InQueueCount() == NUM_OF_PLAYERS {
			event := q.PrepareMatch()
			return &event
		}
	case Dequeue:
		q.RemoveFromQueue(event.Player)
	}
	return nil
}

func (q *QueueManager) AddToQueue(player *Player) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.queue.Queue(player)
	go player.Send(WaitForMatchMessage())
}

func (q *QueueManager) RemoveFromQueue(player *Player) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.queue.Remove(player)
	go player.Send(Response{Type: Success})
}

func (q *QueueManager) PrepareMatch() Event {
	players := make([]*Player, NUM_OF_PLAYERS)
	for i := 0; i < NUM_OF_PLAYERS; i++ {
		players[i] = q.queue.Dequeue()
	}
	return Event{
		Type:    CreateMatch,
		Payload: players,
	}
}

func (q *QueueManager) InQueueCount() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return q.queue.Length()
}
