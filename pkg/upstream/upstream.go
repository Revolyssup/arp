package upstream

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/types"
)

const LoadBalancerRoundRobin = "round_robin"

type Upstream struct {
	name   string
	lbType string
	nodes  []*types.Node
	mu     sync.RWMutex
	// For load balancing
	currentIndex int
}

type Factory struct {
	discoveryManager *discovery.DiscoveryManager
}

func NewFactory(discoveryManager *discovery.DiscoveryManager) *Factory {
	return &Factory{
		discoveryManager: discoveryManager,
	}
}

func (f *Factory) NewUpstream(upsConf config.UpstreamConfig) (*Upstream, error) {
	return newUpstream(upsConf, f.discoveryManager)
}

// TODO: Implement garbage collection for upstream related nodeevents and healthcheck
func newUpstream(upsConf config.UpstreamConfig, discoveryManager *discovery.DiscoveryManager) (*Upstream, error) {
	u := &Upstream{
		name:   upsConf.Name,
		lbType: upsConf.Type,
	}
	if u.lbType == "" {
		u.lbType = LoadBalancerRoundRobin
	}
	// Parse node URLs
	for _, nodeConfig := range upsConf.Nodes {
		parsedURL, err := url.Parse(nodeConfig.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid node URL %s: %v", nodeConfig.URL, err)
		}
		u.nodes = append(u.nodes, &types.Node{
			URL: parsedURL,
		})
	}

	// Initialize service discovery if configured
	if upsConf.Discovery.Type != "" && discoveryManager != nil {
		nodesEvent, err := discoveryManager.GetDiscovery(upsConf.Discovery, upsConf.Service)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize discovery: %v", err)
		}
		go func() {
			for nodes := range nodesEvent {
				u.updateNodes(nodes)
			}
		}()
	}

	return u, nil
}

func (u *Upstream) SelectNode() *types.Node {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if len(u.nodes) == 0 {
		return nil
	}

	if u.lbType == LoadBalancerRoundRobin {
		node := u.nodes[u.currentIndex]
		u.currentIndex = (u.currentIndex + 1) % len(u.nodes)
		return node
	}
	return nil // Unsupported load balancer type
}

func (u *Upstream) updateNodes(nodes []*types.Node) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.nodes = nodes
	u.currentIndex = 0
}
