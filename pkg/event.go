package pkg

type Event struct {
	Type    EventType
	Player  *Player
	Payload interface{}
}

type EventType string

const (
	QueueUp     EventType = "queue"
	Dequeue     EventType = "dequeue"
	CreateMatch EventType = "create_match"
)

type Response struct {
	Type ResponseType
}

type ResponseType string

const (
	Success      ResponseType = "success"
	WaitForMatch ResponseType = "wait_for_match"
)
