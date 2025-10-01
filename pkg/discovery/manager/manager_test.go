package manager

import (
	"testing"
	"time"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/upstream"
)

var discoveryManager, _ = NewDiscoveryManager(logger.New(logger.LevelInfo))
var conf = []config.DiscoveryConfig{
	{
		Type: "demo",
		Config: map[string]any{
			"interval": "1s",
		},
	},
}

func TestUpstreamWithDemoDiscovery(t *testing.T) {

	upstreamFactory := upstream.NewFactory()
	ipupConf := config.UpstreamConfig{
		Name:    "test-upstream-discovery",
		Type:    "round_robin",
		Service: "ip",
		Discovery: config.DiscoveryRef{
			Type: "demo",
		},
	}

	headerupConf := config.UpstreamConfig{
		Name:    "test-upstream-discovery-header",
		Type:    "round_robin",
		Service: "header",
		Discovery: config.DiscoveryRef{
			Type: "demo",
		},
	}

	headerup, err := upstreamFactory.NewUpstream(headerupConf)
	if err != nil {
		t.Fatalf("Failed to create upstream with discovery: %v", err)
	}
	discoveryManager.InitDiscovery(t.Context(), conf)

	if headerup == nil {
		t.Fatal("Expected upstream to be non-nil")
	}

	up, err := upstreamFactory.NewUpstream(ipupConf)
	if err != nil {
		t.Fatalf("Failed to create upstream with discovery: %v", err)
	}

	if up == nil {
		t.Fatal("Expected upstream to be non-nil")
	}
	discoveryManager.InitDiscovery(t.Context(), conf)
	time.Sleep(1 * time.Second) // wait for discovery to populate nodes
	//first try
	firstNode := up.SelectNode()
	//assert returned service
	if firstNode.ServiceName != "ip" {
		t.Errorf("Expected first node service name to be 'ip', got %v", firstNode.ServiceName)
	}
	time.Sleep(1 * time.Second)
	secondNode := headerup.SelectNode()
	if secondNode.ServiceName != "header" {
		t.Errorf("Expected second node service name to be 'header', got %v", secondNode.ServiceName)
	}
	if firstNode == secondNode {
		t.Errorf("Expected different nodes from different service name, got same node %v", firstNode)
	}
}
