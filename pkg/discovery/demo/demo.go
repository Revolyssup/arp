package demo

import (
	"context"
	"net/url"
	"time"

	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/types"
	"github.com/Revolyssup/arp/pkg/upstream"
)

func New(cfg map[string]any, log *logger.Logger) discovery.Discovery {
	return &DemoDiscovery{
		eb:  eventbus.NewEventBus[[]*upstream.Node](log.WithComponent("demo_discovery")),
		cfg: cfg,
		log: log.WithComponent("demo_discovery"),
	}
}

type DemoDiscovery struct {
	eb  *eventbus.EventBus[[]*upstream.Node]
	cfg map[string]any
	log *logger.Logger
}

const (
	DemoServiceAddress = "localhost:9090"
)

func (d *DemoDiscovery) Start(ctx context.Context, name string, eb *eventbus.EventBus[[]*upstream.Node], cfg map[string]any) error {
	d.log.Infof("Starting demo discovery with config: %v", cfg)
	// For demo purposes, we will just publish a static list of nodes every 10 seconds.
	nodes := []*upstream.Node{
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
		for {
			select {
			case <-ctx.Done():
				d.log.Info("Demo discovery stopped")
				return
			case <-time.Tick(t):
				d.log.Infof("Demo discovery publishing nodes for service %s: %v", name, nodes)
				eb.Publish(types.ServiceDiscoveryEventKey(name, "header"), []*upstream.Node{nodes[0]})
				eb.Publish(types.ServiceDiscoveryEventKey(name, "ip"), []*upstream.Node{nodes[1]})
			}
		}
	}()
	return nil
}
