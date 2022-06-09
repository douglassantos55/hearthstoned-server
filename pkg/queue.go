package pkg

import "container/list"

type Queue struct {
	head    *list.List
	players map[*Socket]*list.Element
}

func NewQueue() *Queue {
	return &Queue{
		head:    list.New(),
		players: make(map[*Socket]*list.Element),
	}
}

func (q *Queue) Queue(player *Socket) {
	_, ok := q.players[player]
	if !ok {
		element := q.head.PushBack(player)
		q.players[player] = element
	}
}

func (q *Queue) Dequeue() *Socket {
	element := q.head.Front()
	if element == nil {
		return nil
	}
	player := q.head.Remove(element).(*Socket)
	delete(q.players, player)
	return player
}

func (q *Queue) Remove(player *Socket) {
	element, ok := q.players[player]
	if ok {
		q.head.Remove(element)
		delete(q.players, player)
	}
}

func (q *Queue) Length() int {
	return q.head.Len()
}
