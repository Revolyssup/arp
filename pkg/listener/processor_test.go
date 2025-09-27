package listener

import (
	"sync"
	"testing"
	"time"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/types"
	"github.com/charmbracelet/log"
)

const TIMEOUT = 2 //seconds
func TestListenerProcessor(t *testing.T) {
	eventBus := eventbus.NewEventBus[config.Dynamic](logger.New(log.InfoLevel))
	processor := NewListenerProcessor(eventBus, logger.New(log.InfoLevel))

	type eventCollector struct {
		mu     sync.RWMutex
		events map[string]config.Dynamic
	}

	collector := &eventCollector{
		events: make(map[string]config.Dynamic),
	}

	recordEvent := func(listenerName string, event config.Dynamic) {
		collector.mu.Lock()
		defer collector.mu.Unlock()
		collector.events[listenerName] = event
	}

	getEvent := func(listenerName string) (config.Dynamic, bool) {
		collector.mu.RLock()
		defer collector.mu.RUnlock()
		event, exists := collector.events[listenerName]
		return event, exists
	}

	// Clear events
	clearEvents := func() {
		collector.mu.Lock()
		defer collector.mu.Unlock()
		collector.events = make(map[string]config.Dynamic)
	}

	// Initial dynamic config
	initialConfig := config.Dynamic{
		Routes: []config.RouteConfig{
			{
				Name:     "route1",
				Listener: "listener1",
				Upstream: &config.UpstreamConfig{Name: "upstream1"},
				Plugins:  []config.PluginConfig{{Name: "plugin1"}},
			},
			{
				Name:     "route2",
				Listener: "listener2",
				Upstream: &config.UpstreamConfig{Name: "upstream2"},
				Plugins:  []config.PluginConfig{{Name: "plugin2"}},
			},
		},
		Upstreams: []config.UpstreamConfig{
			{Name: "upstream1", Nodes: []config.Node{{URL: "http://localhost:8081"}}},
			{Name: "upstream2", Nodes: []config.Node{{URL: "http://localhost:8082"}}},
		},
		Plugins: []config.PluginConfig{
			{Name: "plugin1", Type: "auth", Config: map[string]any{"key": "value1"}},
			{Name: "plugin2", Type: "rate_limit", Config: map[string]any{"limit": 100}},
		},
	}

	eventChan1 := eventBus.Subscribe(types.RouteEventKey("listener1"))
	eventChan2 := eventBus.Subscribe(types.RouteEventKey("listener2"))

	var wg sync.WaitGroup
	wg.Add(2)

	// Listener 1 event handler
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(TIMEOUT * time.Second): //timeout
				return
			case event, ok := <-eventChan1:
				if !ok {
					return
				}
				recordEvent("listener1", event)
			}
		}
	}()

	// Listener 2 event handler
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(TIMEOUT * time.Second): //timeout
				return
			case event, ok := <-eventChan2:
				if !ok {
					return
				}
				recordEvent("listener2", event)
			}
		}
	}()

	// Process the initial config
	processor.Process(initialConfig)

	// Wait a bit for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify events for listener1
	event1, exists := getEvent("listener1")
	if !exists {
		t.Errorf("Expected event for listener1, but none received")
	} else {
		if len(event1.Routes) != 1 || event1.Routes[0].Listener != "listener1" {
			t.Errorf("Unexpected routes for listener1: %v", event1.Routes)
		}
		if len(event1.Upstreams) != 1 || event1.Upstreams[0].Name != "upstream1" {
			t.Errorf("Unexpected upstreams for listener1: %v", event1.Upstreams)
		}
		if len(event1.Plugins) != 1 || event1.Plugins[0].Name != "plugin1" {
			t.Errorf("Unexpected plugins for listener1: %v", event1.Plugins)
		}
	}

	// Verify events for listener2
	event2, exists := getEvent("listener2")
	if !exists {
		t.Errorf("Expected event for listener2, but none received")
	} else {
		if len(event2.Routes) != 1 || event2.Routes[0].Listener != "listener2" {
			t.Errorf("Unexpected routes for listener2: %v", event2.Routes)
		}
		if len(event2.Upstreams) != 1 || event2.Upstreams[0].Name != "upstream2" {
			t.Errorf("Unexpected upstreams for listener2: %v", event2.Upstreams)
		}
		if len(event2.Plugins) != 1 || event2.Plugins[0].Name != "plugin2" {
			t.Errorf("Unexpected plugins for listener2: %v", event2.Plugins)
		}
	}

	clearEvents()

	// Update config - modify listener1 and remove listener2
	updatedConfig := config.Dynamic{
		Routes: []config.RouteConfig{
			{
				Name:     "route1",
				Listener: "listener1",
				Upstream: &config.UpstreamConfig{Name: "upstream1"},
				Plugins:  []config.PluginConfig{{Name: "plugin1"}},
			},
			{
				Name:     "route3",
				Listener: "listener1",
				Upstream: &config.UpstreamConfig{Name: "upstream3"},
				Plugins:  []config.PluginConfig{{Name: "plugin3"}},
			},
		},
		Upstreams: []config.UpstreamConfig{
			{Name: "upstream1", Nodes: []config.Node{{URL: "http://localhost:8081"}}},
			{Name: "upstream3", Nodes: []config.Node{{URL: "http://localhost:8083"}}},
		},
		Plugins: []config.PluginConfig{
			{Name: "plugin1", Type: "auth", Config: map[string]any{"key": "value1"}},
			{Name: "plugin3", Type: "cors", Config: map[string]any{"origin": "*"}},
		},
	}

	// Process the updated config
	processor.Process(updatedConfig)

	// Wait a bit for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify updated event for listener1
	event1, exists = getEvent("listener1")
	if !exists {
		t.Errorf("Expected updated event for listener1, but none received")
	} else {
		if len(event1.Routes) != 2 {
			t.Errorf("Expected 2 routes for listener1, got %d", len(event1.Routes))
		}
		upstreamNames := make(map[string]bool)
		for _, up := range event1.Upstreams {
			upstreamNames[up.Name] = true
		}
		if !upstreamNames["upstream1"] || !upstreamNames["upstream3"] {
			t.Errorf("Unexpected upstreams for listener1: %v", event1.Upstreams)
		}
		pluginNames := make(map[string]bool)
		for _, p := range event1.Plugins {
			pluginNames[p.Name] = true
		}
		if !pluginNames["plugin1"] || !pluginNames["plugin3"] {
			t.Errorf("Unexpected plugins for listener1: %v", event1.Plugins)
		}
	}

	// Verify that listener2 received an empty config event
	event2, exists = getEvent("listener2")
	if !exists {
		t.Errorf("Expected removal event for listener2, but none received")
	} else {
		if len(event2.Routes) != 0 || len(event2.Upstreams) != 0 || len(event2.Plugins) != 0 {
			t.Errorf("Expected empty config for removed listener2, got: %v", event2)
		}
	}

	eventBus.Unsubscribe(types.RouteEventKey("listener1"), eventChan1)
	eventBus.Unsubscribe(types.RouteEventKey("listener2"), eventChan2)
	wg.Wait()
}
