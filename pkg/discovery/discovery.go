package discovery

import (
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Discovery interface {
	Start(typ string, eb *eventbus.EventBus[[]*upstream.Node], config map[string]any) error
}
