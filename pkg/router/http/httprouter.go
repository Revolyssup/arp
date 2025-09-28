package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/proxy"
	route "github.com/Revolyssup/arp/pkg/route"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Router struct {
	pathMatcher     *route.PathMatcher
	methodMatcher   *route.MethodMatcher
	headerMatcher   *route.HeaderMatcher
	upstreamFactory *upstream.Factory
	logger          *logger.Logger
}

func NewRouter(listener string, routerFactory *route.Factory, upstreamFactory *upstream.Factory, parentLogger *logger.Logger) *Router {
	return &Router{
		pathMatcher:     route.NewPathMatcher(),
		methodMatcher:   route.NewMethodMatcher(),
		headerMatcher:   route.NewHeaderMatcher(),
		upstreamFactory: upstreamFactory,
		logger:          parentLogger.WithComponent("router"),
	}
}

func (r *Router) UpdateRoutes(routeConfigs []config.RouteConfig, upstreamConfigs []config.UpstreamConfig, pluginConfigs []config.PluginConfig) error {
	// Clear existing matchers
	r.pathMatcher.Clear()
	r.methodMatcher.Clear()
	r.headerMatcher.Clear()

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
			continue
		}
		if up, exists := upstreamMap[upstreamConfig.Name]; exists {
			upstreamConfig = &up
		}

		up, err := r.upstreamFactory.NewUpstream(*upstreamConfig)
		if err != nil {
			return err
		}

		pluginChain := plugin.NewChain()
		for _, pCfg := range rc.Plugins {
			if pluginMap[pCfg.Name] != nil {
				pCfg = *pluginMap[pCfg.Name]
			}
			if pluginFactory, exists := plugin.Registry.Get(pCfg.Type); exists {
				plugin := pluginFactory()
				r.logger.Infof("Adding plugin %s to route %s", pCfg.Name, rc.Name)
				plugin.SetConfig(pCfg.Config)
				pluginChain.Add(plugin)
			} else {
				r.logger.Warnf("Plugin type %s not found for plugin %s in route %s", pCfg.Type, pCfg.Name, rc.Name)
			}
		}

		route := &route.Route{
			Plugins:  pluginChain,
			Upstream: up,
		}

		// Add route to all matchers
		for _, match := range rc.Matches {
			if match.Path != "" {
				r.pathMatcher.Add(match.Path, route)
			}
			if match.Method != "" {
				r.methodMatcher.Add(strings.ToUpper(match.Method), route)
			}
			if len(match.Headers) > 0 {
				r.headerMatcher.Add(match.Headers, route)
			}
		}
	}

	return nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Step 1: Match by path
	pathRoutes := r.pathMatcher.Match(req.URL.Path)
	if len(pathRoutes) == 0 {
		http.NotFound(w, req)
		return
	}

	// Step 2: Match by method
	methodRoutes := r.methodMatcher.Match(req.Method)

	// Step 3: Find intersection of path and method routes
	candidateRoutes := route.IntersectRoutes(pathRoutes, methodRoutes)
	if len(candidateRoutes) == 0 {
		fmt.Println("Candidate routes empty after method matching")
		http.NotFound(w, req)
		return
	}

	// Step 4: Match by headers if needed
	finalRoutes := r.headerMatcher.Match(req.Header, candidateRoutes)
	if len(finalRoutes) == 0 {
		http.NotFound(w, req)
		return
	}

	// Use the first matching route (you might want to add priority logic here)
	route := finalRoutes[0]

	if err := route.Plugins.HandleRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	node := route.Upstream.SelectNode()
	if node == nil {
		http.Error(w, "No available upstream nodes", http.StatusServiceUnavailable)
		return
	}

	wrappedWriter := route.Plugins.WrapResponseWriter(w)
	proxy := proxy.NewReverseProxy()
	proxy.ServeHTTP(wrappedWriter, req, node.URL)
}
