package types

import (
	"net/url"

	"github.com/Revolyssup/arp/pkg/eventbus"
)

type Node struct {
	ServiceName string
	URL         *url.URL
	Weight      int
}

// TODO: Refactor configs to have types
// TODO: Add stopping mechanism

type Discovery interface {
	Start(typ string, eb *eventbus.EventBus[[]*Node], config map[string]any) error
}
