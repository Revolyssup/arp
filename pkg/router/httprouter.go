package router

import (
	"net/http"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/proxy"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type HTTPRouter struct {
	routes   []*Route
	listener string
}

type Route struct {
	matcher  Matcher
	plugins  *plugin.Chain
	upstream *upstream.Upstream
}

func NewHTTPRouter(listener string) *HTTPRouter {
	return &HTTPRouter{
		listener: listener,
	}
}

func (r *HTTPRouter) UpdateRoutes(routeConfigs []config.RouteConfig) error {
	var newRoutes []*Route

	for _, rc := range routeConfigs {
		if rc.Listener != r.listener {
			continue // Skip routes not meant for this listener
		}
		matcher, err := NewCompositeMatcher(rc.Matches)
		if err != nil {
			return err
		}

		upstreamConfig := rc.Upstream
		if upstreamConfig == nil {
			continue // Skip routes with missing upstreams
		}

		// Create upstream
		up, err := upstream.NewUpstream(*upstreamConfig)
		if err != nil {
			return err
		}

		// Create plugin chain
		// pluginChain := plugin.NewChain()
		// for _, pc := range rc.Plugins {
		// 	if p, err := plugin.Get(pc.Name); err == nil {
		// 		pluginChain.Add(p.New(pc.Config))
		// 	}
		// }

		// Sort plugins by priority
		// pluginChain.Sort()

		newRoutes = append(newRoutes, &Route{
			matcher: matcher,
			// plugins:  pluginChain,
			upstream: up,
		})
	}

	r.routes = newRoutes
	return nil
}

// It might be expensive to run each matcher when there are thousands of routes.
// TODO: Optimise route matching
func (r *HTTPRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range r.routes {
		if route.matcher.Match(req) {
			// TODO: Implement Plugins
			// if err := route.plugins.HandleRequest(req); err != nil {
			// 	http.Error(w, err.Error(), http.StatusInternalServerError)
			// 	return
			// }

			// Get upstream node
			node := route.upstream.SelectNode()
			if node == nil {
				http.Error(w, "No available upstream nodes", http.StatusServiceUnavailable)
				return
			}
			proxy := proxy.NewReverseProxy()
			proxy.ServeHTTP(w, req, node.URL)
			return
		}
	}

	http.NotFound(w, req)
}
