package demo

import (
	"net/url"
	"time"

	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/types"
)

func New(cfg map[string]any, log *logger.Logger) types.Discovery {
	return &DemoDiscovery{
		eb:  eventbus.NewEventBus[[]*types.Node](log.WithComponent("demo_discovery")),
		cfg: cfg,
		log: log.WithComponent("demo_discovery"),
	}
}

type DemoDiscovery struct {
	eb  *eventbus.EventBus[[]*types.Node]
	cfg map[string]any
	log *logger.Logger
}

const (
	DemoServiceAddress = "localhost:9090"
)

func (d *DemoDiscovery) Start(name string, eb *eventbus.EventBus[[]*types.Node], cfg map[string]any) error {
	d.log.Infof("Starting demo discovery with config: %v", cfg)
	// For demo purposes, we will just publish a static list of nodes every 10 seconds.
	nodes := []*types.Node{
		{
			ServiceName: "header",
			URL:         &url.URL{Scheme: "http", Host: DemoServiceAddress, Path: "/headers"},
		},
		{
			ServiceName: "ip",
			URL:         &url.URL{Scheme: "http", Host: DemoServiceAddress, Path: "/ip"},
		},
	}
	t := 10 * time.Second
	if interval, ok := cfg["interval"].(string); ok {
		if dur, err := time.ParseDuration(interval); err == nil {
			t = dur
		}
	}
	go func() {
		// simulate pushing updated nodes every t seconds with same servername and url
		eb.Publish(types.ServiceDiscoveryEventKey(name, "header"), []*types.Node{nodes[0]})
		eb.Publish(types.ServiceDiscoveryEventKey(name, "ip"), []*types.Node{nodes[1]})
		time.Sleep(t)
	}()
	return nil
}
