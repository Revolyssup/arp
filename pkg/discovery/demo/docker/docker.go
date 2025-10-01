package docker

import (
	"context"
	"fmt"

	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/upstream"
	"github.com/moby/moby/client"
)

type DockerDiscovery struct {
	cli    *client.Client
	cfg    map[string]any
	ctx    context.Context
	cancel context.CancelFunc
}

func New(cfg map[string]any) (discovery.Discovery, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}

	return &DockerDiscovery{
		cli:    cli,
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// TODO: Implement actual discovery logic
func (d *DockerDiscovery) Start(name string, eb *eventbus.EventBus[[]*upstream.Node], config map[string]any) error {
	return nil
}
