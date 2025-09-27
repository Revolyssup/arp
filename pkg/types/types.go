package types

import (
	"net/url"

	"github.com/Revolyssup/arp/pkg/eventbus"
)

//TODO: Do I need this types.go?

func RouteEventKey(listenerName string) string {
	return "routes_" + listenerName
}

func StreamRouteEventKey(listenerName string) string {
	return "stream_routes_" + listenerName
}

type Node struct {
	ServiceName string
	URL         *url.URL
}

type Discovery interface {
	Start(typ string, eb *eventbus.EventBus[[]*Node], config map[string]any) error
}
