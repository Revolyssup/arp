package router

import (
	"net/http"

	"log"

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

func (r *HTTPRouter) UpdateRoutes(routeConfigs []config.RouteConfig, upstreamConfigs []config.UpstreamConfig, pluginConfigs []config.PluginConfig) error {
	var newRoutes []*Route
	upstreamMap := make(map[string]config.UpstreamConfig)
	for _, up := range upstreamConfigs {
		upstreamMap[up.Name] = up
	}

	pluginMap := make(map[string]*config.PluginConfig)
	for _, p := range pluginConfigs {
		pluginMap[p.Name] = &p
	}
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
		//If upstream configuration exists, then it will override the upstream passed in route.
		if up, exists := upstreamMap[upstreamConfig.Name]; exists {
			upstreamConfig = &up
		}
		// Create upstream
		up, err := upstream.NewUpstream(*upstreamConfig)
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
		newRoutes = append(newRoutes, &Route{
			matcher:  matcher,
			plugins:  pluginChain,
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
			if err := route.plugins.HandleRequest(req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Get upstream node
			node := route.upstream.SelectNode()
			if node == nil {
				http.Error(w, "No available upstream nodes", http.StatusServiceUnavailable)
				return
			}
			wrappedWriter := route.plugins.WrapResponseWriter(w)
			proxy := proxy.NewReverseProxy()
			proxy.ServeHTTP(wrappedWriter, req, node.URL)
			return
		}
	}

	http.NotFound(w, req)
}
