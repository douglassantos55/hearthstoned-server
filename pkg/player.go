package pkg

import "github.com/google/uuid"

type Player struct {
	Id uuid.UUID

	Outgoing chan Response // messages to client
}

func NewPlayer() *Player {
	return &Player{
		Id: uuid.New(),

		Outgoing: make(chan Response),
	}
}

func (p *Player) Send(message Response) {
	p.Outgoing <- message
}
