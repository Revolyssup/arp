package streamroute

import (
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/upstream"
)

// TODO: Implement L4 routing logic
type Route struct {
	Plugins  *plugin.Chain
	Upstream *upstream.Upstream
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}
func (f *Factory) NewRoute(plugins *plugin.Chain, up *upstream.Upstream) *Route {
	return &Route{
		Plugins:  plugins,
		Upstream: up,
	}
}
