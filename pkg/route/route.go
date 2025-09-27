package route

import (
	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Route struct {
	Matcher  Matcher
	Plugins  *plugin.Chain
	Upstream *upstream.Upstream
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}
func (f *Factory) NewRoute(matches []config.Match, plugins *plugin.Chain, up *upstream.Upstream) *Route {
	matcher, err := NewCompositeMatcher(matches)
	if err != nil {
		return nil
	}
	return &Route{
		Matcher:  matcher,
		Plugins:  plugins,
		Upstream: up,
	}
}
