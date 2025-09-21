package upstream

import (
	"fmt"

	"github.com/Revolyssup/arp/pkg/config"
)

type Discovery interface {
	Nodes() <-chan []*Node
}

func NewDiscovery(config config.DiscoveryRef) (Discovery, error) {
	switch config.Type {
	// case "dns":
	// 	return NewDNSDiscovery(config)
	// case "kubernetes":
	// 	return NewKubernetesDiscovery(config)
	default:
		return nil, fmt.Errorf("unsupported discovery type: %s", config.Type)
	}
}
