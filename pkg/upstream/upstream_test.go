package upstream

import (
	"testing"
	"time"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/logger"
)

var discoveryManager, _ = discovery.NewDiscoveryManager([]config.DiscoveryConfig{
	{
		Type: "demo",
		Config: map[string]any{
			"interval": "1s",
		},
	},
}, logger.New(logger.LevelInfo))

func TestUpstream(t *testing.T) {

	upstreamFactory := NewFactory(discoveryManager)

	upConf := config.UpstreamConfig{
		Name: "test-upstream",
		Nodes: []config.Node{
			{URL: "http://localhost:8080"},
			{URL: "http://localhost:8081"},
		},
		Type: "round_robin",
	}

	up, err := upstreamFactory.NewUpstream(upConf)
	if err != nil {
		t.Fatalf("Failed to create upstream: %v", err)
	}

	if up == nil {
		t.Fatal("Expected upstream to be non-nil")
	}

	if len(up.nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(up.nodes))
	}

	// Test round-robin selection
	firstNode := up.SelectNode()
	secondNode := up.SelectNode()
	thirdNode := up.SelectNode()

	if firstNode != up.nodes[0] {
		t.Errorf("Expected first node to be %v, got %v", up.nodes[0], firstNode)
	}
	if secondNode != up.nodes[1] {
		t.Errorf("Expected second node to be %v, got %v", up.nodes[1], secondNode)
	}
	if thirdNode != up.nodes[0] {
		t.Errorf("Expected third node to be %v, got %v", up.nodes[0], thirdNode)
	}
}

func TestUpstreamWithDiscovery(t *testing.T) {

	upstreamFactory := NewFactory(discoveryManager)

	upConf := config.UpstreamConfig{
		Name: "test-upstream-discovery",
		Type: "round_robin",
		Discovery: config.DiscoveryRef{
			Type: "demo",
		},
	}

	up, err := upstreamFactory.NewUpstream(upConf)
	if err != nil {
		t.Fatalf("Failed to create upstream with discovery: %v", err)
	}

	if up == nil {
		t.Fatal("Expected upstream to be non-nil")
	}
	time.Sleep(1 * time.Second) // wait for discovery to populate nodes

	if len(up.nodes) == 0 {
		t.Fatalf("Expected nodes to be populated by discovery, got 0")
	}
	//first try
	firstNode := up.SelectNode()
	time.Sleep(1 * time.Second)
	secondNode := up.SelectNode()
	if firstNode == secondNode {
		t.Errorf("Expected different nodes from discovery, got same node %v", firstNode)
	}
}
