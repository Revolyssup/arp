package eventbus

import (
	"sync"

	"github.com/Revolyssup/arp/pkg/logger"
)

/*
This is mostly stateless EventBus with one modification:
Any new subscriber will get the last published event for a topic immediately after subscribing.
*/
//TODO: Can this cache be improved(ttl or something) or is it too simple to concern myself with?
type EventBus[T any] struct {
	subscribers map[string][]chan T
	cache       map[string]T // Cache for last published value per topic
	mx          sync.RWMutex
	log         *logger.Logger
}

func NewEventBus[T any](parentLogger *logger.Logger) *EventBus[T] {
	return &EventBus[T]{
		subscribers: make(map[string][]chan T),
		cache:       make(map[string]T),
		log:         parentLogger.WithComponent("eventbus"),
	}
}

func (eb *EventBus[T]) Subscribe(topic string) <-chan T {
	eb.mx.Lock()
	defer eb.mx.Unlock()
	ch := make(chan T, 100)
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	// Send the last cached value if it exists
	if cached, exists := eb.cache[topic]; exists {
		go func() {
			ch <- cached
		}()
	}

	return ch
}

func (eb *EventBus[T]) Unsubscribe(topic string, ch <-chan T) {
	eb.mx.Lock()
	defer eb.mx.Unlock()

	subscribers, exists := eb.subscribers[topic]
	if !exists {
		return
	}

	// Find and remove the channel
	for i, subscriber := range subscribers {
		if subscriber == ch {
			eb.subscribers[topic] = append(eb.subscribers[topic][:i], eb.subscribers[topic][i+1:]...)
			close(subscriber)
			return
		}
	}
}

func (eb *EventBus[T]) Publish(topic string, data T) {
	eb.mx.Lock()
	defer eb.mx.Unlock()
	// Update cache first
	eb.cache[topic] = data

	for _, ch := range eb.subscribers[topic] {
		select {
		case ch <- data:
			// Successfully sent
		default:
			// Channel is full, log warning but don't block
			eb.log.Infof("WARNING: Channel full for topic %s, dropping message", topic)
		}
	}
}
