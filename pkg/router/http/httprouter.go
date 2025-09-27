package router

import (
	"net/http"

	"log"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/proxy"
	route "github.com/Revolyssup/arp/pkg/route"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Router struct {
	routes          []*route.Route
	routerFactory   *route.Factory
	upstreamFactory *upstream.Factory
}

func NewRouter(listener string, routerFactory *route.Factory, upstreamFactory *upstream.Factory) *Router {
	return &Router{
		routerFactory:   routerFactory,
		upstreamFactory: upstreamFactory,
	}
}

func (r *Router) UpdateRoutes(routeConfigs []config.RouteConfig, upstreamConfigs []config.UpstreamConfig, pluginConfigs []config.PluginConfig) error {
	var newRoutes []*route.Route
	upstreamMap := make(map[string]config.UpstreamConfig)
	for _, up := range upstreamConfigs {
		upstreamMap[up.Name] = up
	}

	pluginMap := make(map[string]*config.PluginConfig)
	for _, p := range pluginConfigs {
		pluginMap[p.Name] = &p
	}
	for _, rc := range routeConfigs {
		upstreamConfig := rc.Upstream
		if upstreamConfig == nil {
			continue // Skip routes with missing upstreams
		}
		//If upstream configuration exists, then it will override the upstream passed in route.
		if up, exists := upstreamMap[upstreamConfig.Name]; exists {
			upstreamConfig = &up
		}
		// Create upstream
		up, err := r.upstreamFactory.NewUpstream(*upstreamConfig)
		if err != nil {
			return err
		}

		// Create plugin chain
		pluginChain := plugin.NewChain()
		for _, pCfg := range rc.Plugins {
			if pluginMap[pCfg.Name] != nil {
				pCfg = *pluginMap[pCfg.Name]
			}
			if pluginFactory, exists := plugin.Registry.Get(pCfg.Type); exists {
				plugin := pluginFactory()
				log.Printf("Adding plugin %s to route %s", pCfg.Name, rc.Name)
				plugin.SetConfig(pCfg.Config)
				pluginChain.Add(plugin)
			} else {
				log.Printf("Plugin type %s not found for plugin %s in route %s", pCfg.Type, pCfg.Name, rc.Name)
			}
		}
		route := r.routerFactory.NewRoute(rc.Matches, pluginChain, up)
		newRoutes = append(newRoutes, route)
	}

	r.routes = newRoutes
	return nil
}

// It might be expensive to run each matcher when there are thousands of routes.
// TODO: Optimise route matching
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range r.routes {
		if route.Matcher.Match(req) {
			if err := route.Plugins.HandleRequest(req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Get upstream node
			node := route.Upstream.SelectNode()
			if node == nil {
				http.Error(w, "No available upstream nodes", http.StatusServiceUnavailable)
				return
			}
			wrappedWriter := route.Plugins.WrapResponseWriter(w)
			proxy := proxy.NewReverseProxy()
			proxy.ServeHTTP(wrappedWriter, req, node.URL)
			return
		}
	}

	http.NotFound(w, req)
}
