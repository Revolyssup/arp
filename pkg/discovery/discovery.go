package discovery

import (
	"fmt"
	"log"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery/demo"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/types"
)

// Manages all instantiated discovereres and based on the config, gives an event bus to client to subscribe on.
type DiscoveryManager struct {
	discoverers map[string]types.Discovery
	eb          *eventbus.EventBus[[]*types.Node]
}

func NewDiscoveryManager(cfg []config.DiscoveryConfig) *DiscoveryManager {
	mgr := &DiscoveryManager{
		discoverers: make(map[string]types.Discovery),
		eb:          eventbus.NewEventBus[[]*types.Node](),
	}
	// Start all discoverers from cfg
	//TODO: Refactor this into registration based data flow.
	for _, dcfg := range cfg {
		switch dcfg.Type {
		case "demo":
			mgr.discoverers["demo"] = demo.New(dcfg.Config)
		default:
			log.Printf("Unsupported discovery type: %s", dcfg.Type)
			continue
		}
		log.Printf("Starting discovery with config: %v", dcfg)
		if err := mgr.discoverers[dcfg.Type].Start(dcfg.Type, mgr.eb, dcfg.Config); err != nil {
			log.Printf("failed to start discovery %s: %v", dcfg.Type, err)
		}
	}
	log.Printf("RETURNED MANAGER FROM NEWDISCOVERYMANAGER %v", *mgr)
	return mgr
}

func (d *DiscoveryManager) NewDiscovery(config config.DiscoveryRef) (<-chan []*types.Node, error) {
	log.Printf("discoverers %v", d)
	if _, exists := d.discoverers[config.Type]; exists {
		return d.eb.Subscribe(config.Type), nil
	}
	return nil, fmt.Errorf("unsupported discovery type: %s", config.Type)
}
