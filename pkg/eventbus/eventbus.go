package eventbus

import "sync"

type EventBus[T any] struct {
	subscribers map[string][]chan T
	mx          sync.RWMutex
}

func NewEventBus[T any]() *EventBus[T] {
	return &EventBus[T]{
		subscribers: make(map[string][]chan T),
	}
}

func (eb *EventBus[T]) Subscribe(topic string) <-chan T {
	eb.mx.Lock()
	defer eb.mx.Unlock()

	ch := make(chan T, 100)
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	return ch
}

func (eb *EventBus[T]) Publish(topic string, data T) {
	eb.mx.RLock()
	defer eb.mx.RUnlock()

	for _, ch := range eb.subscribers[topic] {
		go func(ch chan T) {
			ch <- data
		}(ch)
	}
}
