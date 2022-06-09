package pkg

import "github.com/google/uuid"

type Socket struct {
	Id uuid.UUID

	Outgoing chan Response // messages to client
}

func NewSocket() *Socket {
	return &Socket{
		Id: uuid.New(),

		Outgoing: make(chan Response),
	}
}

func (p *Socket) Send(message Response) {
	p.Outgoing <- message
}
