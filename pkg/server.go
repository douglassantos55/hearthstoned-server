package pkg

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func NewDisconnected(player *Socket) Event {
	return Event{
		Type:   Disconnected,
		Player: player,
	}
}

type Server struct {
	handlers []EventHandler
	upgrader websocket.Upgrader
}

func NewServer() *Server {
	return &Server{
		handlers: make([]EventHandler, 0),
		upgrader: websocket.Upgrader{},
	}
}

func (s *Server) Listen(addr string) {
	http.HandleFunc("/", s.HandleConnection)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *Server) RegisterHandler(handler EventHandler) {
	s.handlers = append(s.handlers, handler)
}

func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
	s.upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("Could not upgrade connection")
		return
	}

	socket := NewSocket(conn)

	go func() {
		defer conn.Close()

		for {
			select {
			case event := <-socket.Incoming:
				event.Player = socket
				s.ProcessEvent(event)
			case <-socket.Disconnect:
				s.ProcessEvent(NewDisconnected(socket))
				return
			}
		}
	}()
}

func (s *Server) ProcessEvent(event Event) {
	for _, handler := range s.handlers {
		if recast := handler.Process(event); recast != nil {
			s.ProcessEvent(*recast)
		}
	}
}
