package discovery

import (
	"fmt"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery/demo"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/types"
)

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

func (d *DiscoveryManager) GetDiscovery(config config.DiscoveryRef) (<-chan []*types.Node, error) {
	d.log.Warnf("discoverers %v", d)
	if _, exists := d.discoverers[config.Type]; exists {
		return d.eb.Subscribe(config.Type), nil
	}
	return nil, fmt.Errorf("unsupported discovery type: %s", config.Type)
}
