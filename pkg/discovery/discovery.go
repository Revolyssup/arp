package discovery

import (
	"context"

	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Discovery interface {
	Start(ctx context.Context, typ string, eb *eventbus.EventBus[[]*upstream.Node], config map[string]any) error
}
