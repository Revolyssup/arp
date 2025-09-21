package eventbus

import (
	"sync"
	"testing"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus[string]()

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
	bus := NewEventBus[int]()

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
