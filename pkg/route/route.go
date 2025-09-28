package route

import (
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Route struct {
	Plugins  *plugin.Chain
	Upstream *upstream.Upstream
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func IntersectRoutes(r1, r2 []*Route) []*Route {
	if len(r1) == 0 {
		return r2
	}
	if len(r2) == 0 {
		return r1
	}
	set := make(map[*Route]bool)
	for _, route := range r1 {
		set[route] = true
	}

	var result []*Route
	for _, route := range r2 {
		if set[route] {
			result = append(result, route)
		}
	}
	return result
}
