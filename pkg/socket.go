package pkg

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Socket struct {
	Id uuid.UUID

	Outgoing   chan Response // messages to client
	Incoming   chan Event    // messages from client
	Disconnect chan bool

	socket *websocket.Conn
}

func NewSocket(conn *websocket.Conn) *Socket {
	socket := &Socket{
		Id: uuid.New(),

		Incoming:   make(chan Event),
		Outgoing:   make(chan Response),
		Disconnect: make(chan bool),

		socket: conn,
	}

	go socket.Read()
	go socket.Write()

	return socket
}

func NewTestSocket() *Socket {
	return &Socket{
		Id: uuid.New(),

		Incoming:   make(chan Event),
		Outgoing:   make(chan Response),
		Disconnect: make(chan bool),
	}
}

func (p *Socket) Send(message Response) {
	p.Outgoing <- message
}

func (s *Socket) Read() {
	for {
		var event Event
		err := s.socket.ReadJSON(&event)
		if err != nil {
			break
		}
		s.Incoming <- event
	}
	s.Disconnect <- true
}

func (s *Socket) Write() {
	for {
		select {
		case msg := <-s.Outgoing:
			s.socket.WriteJSON(msg)
		}
	}
}
