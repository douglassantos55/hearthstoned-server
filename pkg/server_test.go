package pkg

import (
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type DisconnectHandler struct {
	Count int
	mutex *sync.Mutex
}

func NewDisconnectHandler() *DisconnectHandler {
	return &DisconnectHandler{
		mutex: new(sync.Mutex),
	}
}

func (d *DisconnectHandler) GetCount() int {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.Count
}

func (d *DisconnectHandler) Process(event Event) *Event {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if event.Type == Disconnected {
		d.Count++
	}
	return nil
}

func TestServer(t *testing.T) {
	t.Run("disconnect", func(t *testing.T) {
		server := NewServer()
		handler := NewDisconnectHandler()

		server.RegisterHandler(handler)
		go server.Listen("0.0.0.0:8888")

		time.Sleep(time.Millisecond)

		// connect
		socket, _, err := websocket.DefaultDialer.Dial("ws://0.0.0.0:8888", nil)
		if err != nil {
			t.Error(err)
		}

		// disconnect
		socket.Close()

		time.Sleep(time.Millisecond)

		if handler.GetCount() != 1 {
			t.Errorf("Expected %v count, got %v", 1, handler.GetCount())
		}
	})
}
