package tcp

import (
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/streamroute"
	"github.com/Revolyssup/arp/pkg/upstream"
)

// TODO: Implement L4 routing logic
type Router struct {
	routes             []*streamroute.Route
	streamRouteFactory *streamroute.Factory
	upstreamFactory    *upstream.Factory
	logger             *logger.Logger
}

func NewRouter(streamRouteFactory *streamroute.Factory, upstreamFactory *upstream.Factory, parentLogger *logger.Logger) *Router {
	return &Router{
		streamRouteFactory: streamRouteFactory,
		upstreamFactory:    upstreamFactory,
		logger:             parentLogger.WithComponent("tcprouter"),
	}
}
