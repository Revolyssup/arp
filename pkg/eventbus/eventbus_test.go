package eventbus

import (
	"sync"
	"testing"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/charmbracelet/log"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus[string](logger.New(log.InfoLevel))

	topic := "test_topic"
	subscriber := bus.Subscribe(topic)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		msg := <-subscriber
		if msg != "hello" {
			t.Errorf("expected 'hello', got '%s'", msg)
		}
	}()

	bus.Publish(topic, "hello")
	wg.Wait()
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewEventBus[int](logger.New(log.InfoLevel))

	topic := "numbers"
	subscriber1 := bus.Subscribe(topic)
	subscriber2 := bus.Subscribe(topic)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		num := <-subscriber1
		if num != 42 {
			t.Errorf("subscriber1 expected 42, got %d", num)
		}
	}()

	go func() {
		defer wg.Done()
		num := <-subscriber2
		if num != 42 {
			t.Errorf("subscriber2 expected 42, got %d", num)
		}
	}()

	bus.Publish(topic, 42)
	wg.Wait()
}

// Subscribers should get the last published event immediately after subscribing.
func TestSubscribersComeAfterPublishers(t *testing.T) {
	bus := NewEventBus[string](logger.New(log.InfoLevel))

	topic := "late_topic"
	bus.Publish(topic, "cached_message")
	bus.Publish(topic, "cached_message2")
	bus.Publish(topic, "cached_message3")
	subscriber := bus.Subscribe(topic)

	msg := <-subscriber
	if msg != "cached_message3" {
		t.Errorf("expected 'cached_message3', got '%s'", msg)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewEventBus[string](logger.New(log.InfoLevel))

	topic := "unsub_topic"
	subscriber := bus.Subscribe(topic)
	bus.Unsubscribe(topic, subscriber)

	select {
	case msg, ok := <-subscriber:
		if !ok {
			return // Channel closed as expected
		}
		t.Errorf("expected no message after unsubscribe, but got '%s'", msg)
	default:
		// No message received, as expected
	}

}
