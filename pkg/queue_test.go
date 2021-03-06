package pkg

import "testing"

func TestAddToQueue(t *testing.T) {
	queue := NewQueue()

	queue.Queue(NewTestSocket())
	queue.Queue(NewTestSocket())
	queue.Queue(NewTestSocket())

	if queue.Length() != 3 {
		t.Errorf("Expected %v, got %v", 3, queue.Length())
	}
	if len(queue.players) != 3 {
		t.Errorf("Expected %v, got %v", 3, len(queue.players))
	}
}

func TestRemoveFromQueue(t *testing.T) {
	p1 := NewTestSocket()
	queue := NewQueue()

	queue.Queue(p1)
	queue.Queue(NewTestSocket())
	queue.Queue(NewTestSocket())

	got := queue.Dequeue()

	if got != p1 {
		t.Errorf("Expected %v, got %v", p1, got)
	}
	if queue.Length() != 2 {
		t.Errorf("Expected %v, got %v", 2, queue.Length())
	}
	if len(queue.players) != 2 {
		t.Errorf("Expected %v, got %v", 2, len(queue.players))
	}
}

func TestAddDuplicates(t *testing.T) {
	queue := NewQueue()
	player := NewTestSocket()

	queue.Queue(player)
	queue.Queue(player)
	queue.Queue(player)

	if queue.Length() != 1 {
		t.Errorf("Expected %v, got %v", 1, queue.Length())
	}
	if len(queue.players) != 1 {
		t.Errorf("Expected %v, got %v", 1, len(queue.players))
	}
}

func TestRemoveParticularPlayer(t *testing.T) {
	queue := NewQueue()
	p2 := NewTestSocket()

	queue.Queue(NewTestSocket())
	queue.Queue(p2)
	queue.Queue(NewTestSocket())

	queue.Remove(p2)
	if queue.Length() != 2 {
		t.Errorf("Expected %v, got %v", 2, queue.Length())
	}
	if len(queue.players) != 2 {
		t.Errorf("Expected %v, got %v", 2, len(queue.players))
	}

	player := queue.Dequeue()
	for player != nil {
		if player == p2 {
			t.Errorf("Did not expect %v", p2)
		}
		player = queue.Dequeue()
	}
}
