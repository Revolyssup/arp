package discovery

import (
	"fmt"
	"sync"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery/demo"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/types"
	"github.com/Revolyssup/arp/pkg/upstream"
	"github.com/Revolyssup/arp/pkg/utils"
)

func (d *DiscoveryManager) InitDiscovery(ups *upstream.Upstream, discoveryManager *DiscoveryManager, discoveryConf config.DiscoveryRef, serviceName string) chan error {
	errChan := make(chan error, 1)
	nodesEvent, err := discoveryManager.GetDiscovery(discoveryConf, serviceName)
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

// Manages all instantiated discovereres and based on the config, gives an event bus to client to subscribe on.
type DiscoveryManager struct {
	discoverers map[string]types.Discovery
	eb          *eventbus.EventBus[[]*types.Node]
	log         *logger.Logger
}

func NewDiscoveryManager(cfg []config.DiscoveryConfig, parentLogger *logger.Logger) (*DiscoveryManager, error) {
	discoveryLogger := parentLogger.WithComponent("discovery_manager")
	mgr := &DiscoveryManager{
		discoverers: make(map[string]types.Discovery),
		eb:          eventbus.NewEventBus[[]*types.Node](discoveryLogger),
		log:         discoveryLogger,
	}
	// Start all discoverers from cfg
	//TODO: Refactor this into registration based data flow.
	for _, dcfg := range cfg {
		switch dcfg.Type {
		case "demo":
			mgr.discoverers["demo"] = demo.New(dcfg.Config, parentLogger)
		default:
			return nil, fmt.Errorf("unsupported discovery type: %s", dcfg.Type)
		}
		mgr.log.Infof("Starting discovery with config: %v", dcfg)
		if err := mgr.discoverers[dcfg.Type].Start(dcfg.Type, mgr.eb, dcfg.Config); err != nil {
			return nil, fmt.Errorf("failed to start discovery %s: %w", dcfg.Type, err)
		}
	}
	return mgr, nil
}

func (d *DiscoveryManager) GetDiscovery(config config.DiscoveryRef, serviceName string) (<-chan []*types.Node, error) {
	d.log.Warnf("discoverers %v", d)
	if _, exists := d.discoverers[config.Type]; exists {
		return d.eb.Subscribe(types.ServiceDiscoveryEventKey(config.Type, serviceName)), nil
	}
	return nil, fmt.Errorf("unsupported discovery type: %s", config.Type)
}
