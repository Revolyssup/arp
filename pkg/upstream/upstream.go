package upstream

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/Revolyssup/arp/pkg/config"
)

const LoadBalancerRoundRobin = "round_robin"

type Upstream struct {
	name   string
	lbType string
	nodes  []*Node
	mu     sync.RWMutex
	// For load balancing
	currentIndex int
	// For service discovery
	discovery Discovery
}

type Node struct {
	URL    *url.URL
	Weight int
}

func NewUpstream(upsConf config.UpstreamConfig) (*Upstream, error) {
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
		u.nodes = append(u.nodes, &Node{
			URL:    parsedURL,
			Weight: nodeConfig.Weight,
		})
	}

	// Initialize service discovery if configured
	if upsConf.Discovery.Type != "" {
		discovery, err := NewDiscovery(config.DiscoveryRef(upsConf.Discovery))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize discovery: %v", err)
		}
		u.discovery = discovery
		go func() {
			for nodes := range discovery.Nodes() {
				u.updateNodes(nodes)
			}
		}()
	}

	return u, nil
}

func (u *Upstream) SelectNode() *Node {
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

func (u *Upstream) updateNodes(nodes []*Node) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.nodes = nodes
	u.currentIndex = 0
}
