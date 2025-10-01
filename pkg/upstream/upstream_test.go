package upstream

import (
	"testing"

	"github.com/Revolyssup/arp/pkg/config"
)

func TestUpstream(t *testing.T) {

	upstreamFactory := NewFactory()

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
