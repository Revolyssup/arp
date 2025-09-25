package demo

import (
	"log"
	"net/url"
	"time"

	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/types"
)

func New(cfg map[string]any) types.Discovery {
	return &DemoDiscovery{
		eb:  eventbus.NewEventBus[[]*types.Node](),
		cfg: cfg,
	}
}

type DemoDiscovery struct {
	eb  *eventbus.EventBus[[]*types.Node]
	cfg map[string]any
}

func (d *DemoDiscovery) Start(name string, eb *eventbus.EventBus[[]*types.Node], cfg map[string]any) error {
	log.Printf("Starting demo discovery with config: %v", cfg)
	// For demo purposes, we will just publish a static list of nodes every 10 seconds.
	nodes := []*types.Node{
		{
			URL:    &url.URL{Scheme: "http", Host: "httpbin.org", Path: "/headers"},
			Weight: 1,
		},
		{
			URL:    &url.URL{Scheme: "http", Host: "httpbin.org", Path: "/ip"},
			Weight: 1,
		},
	}
	t := 10 * time.Second
	if interval, ok := cfg["interval"].(string); ok {
		if dur, err := time.ParseDuration(interval); err == nil {
			t = dur
		}
	}
	go func() {
		//simulating pushing different nodes every interval
		for i := 0; ; i++ {
			log.Printf("Publishing nodes for demo discovery: %v for name %s", nodes[i%len(nodes)], name)
			eb.Publish(name, []*types.Node{nodes[i%len(nodes)]})
			time.Sleep(t)
		}
	}()
	return nil
}
