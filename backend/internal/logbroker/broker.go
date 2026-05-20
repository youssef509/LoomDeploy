package logbroker

import "sync"

type Broker struct {
	mu       sync.Mutex
	channels map[string][]chan string
	history  map[string][]string
}

var Global = &Broker{
	channels: make(map[string][]chan string),
	history:  make(map[string][]string),
}

// Subscribe returns a channel pre-filled with all lines published so far,
// followed by any future lines until Close is called.
func (b *Broker) Subscribe(deployID string) chan string {
	ch := make(chan string, 1024)
	b.mu.Lock()
	for _, line := range b.history[deployID] {
		select {
		case ch <- line:
		default:
		}
	}
	b.channels[deployID] = append(b.channels[deployID], ch)
	b.mu.Unlock()
	return ch
}

func (b *Broker) Unsubscribe(deployID string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.channels[deployID]
	for i, s := range subs {
		if s == ch {
			b.channels[deployID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (b *Broker) Publish(deployID, line string) {
	b.mu.Lock()
	b.history[deployID] = append(b.history[deployID], line)
	subs := b.channels[deployID]
	b.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (b *Broker) Close(deployID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.channels[deployID] {
		close(ch)
	}
	delete(b.channels, deployID)
	delete(b.history, deployID)
}
