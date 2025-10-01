package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/discovery/demo"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/types"
	"github.com/Revolyssup/arp/pkg/upstream"
	"github.com/Revolyssup/arp/pkg/utils"
)

// Manages all instantiated discovereres and based on the config, gives an event bus to client to subscribe on.
type DiscoveryManager struct {
	discoverers map[string]discovery.Discovery
	eb          *eventbus.EventBus[[]*upstream.Node]
	log         *logger.Logger
}

func NewDiscoveryManager(parentLogger *logger.Logger) (*DiscoveryManager, error) {
	discoveryLogger := parentLogger.WithComponent("discovery_manager")
	mgr := &DiscoveryManager{
		discoverers: make(map[string]discovery.Discovery),
		eb:          eventbus.NewEventBus[[]*upstream.Node](discoveryLogger),
		log:         discoveryLogger,
	}
	return mgr, nil
}

func (d *DiscoveryManager) InitDiscovery(ctx context.Context, cfg []config.DiscoveryConfig) error {
	// Start all discoverers from cfg
	//TODO: Refactor this into registration based data flow.
	for _, dcfg := range cfg {
		switch dcfg.Type {
		case "demo":
			d.discoverers["demo"] = demo.New(dcfg.Config, d.log)
		default:
			return fmt.Errorf("unsupported discovery type: %s", dcfg.Type)
		}
		d.log.Infof("Starting discovery with config: %v", dcfg)
		if err := d.discoverers[dcfg.Type].Start(ctx, dcfg.Type, d.eb, dcfg.Config); err != nil {
			return fmt.Errorf("failed to start discovery %s: %w", dcfg.Type, err)
		}
	}
	return nil
}

func (d *DiscoveryManager) getDiscovery(config config.DiscoveryRef, serviceName string) (<-chan []*upstream.Node, error) {
	d.log.Warnf("discoverers %v", d)
	if _, exists := d.discoverers[config.Type]; exists {
		return d.eb.Subscribe(types.ServiceDiscoveryEventKey(config.Type, serviceName)), nil
	}
	return nil, fmt.Errorf("unsupported discovery type: %s", config.Type)
}

func (d *DiscoveryManager) StartDiscovery(ups *upstream.Upstream, discoveryManager *DiscoveryManager, discoveryConf config.DiscoveryRef, serviceName string) chan error {
	errChan := make(chan error, 1)
	nodesEvent, err := discoveryManager.getDiscovery(discoveryConf, serviceName)
	if err != nil {
		errChan <- fmt.Errorf("failed to initialize discovery: %v", err)
		return errChan
	}
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()
	wg.Add(1)
	utils.GoWithRecover(func() {
		defer wg.Done()
		for nodes := range nodesEvent {
			ups.UpdateNodes(nodes)
		}
	}, func(a any) {
		errChan <- fmt.Errorf("panic in node update listener for upstream %s: %v", ups.Name(), a)
	})
	go func() {
		wg.Wait()
		close(errChan)
	}()
	return errChan
}
