package router

import (
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
	pluginChain     []*plugin.Chain
	pathMatcher     *route.PathMatcher
	methodMatcher   *route.MethodMatcher
	headerMatcher   *route.HeaderMatcher
	upstreamFactory *upstream.Factory
	logger          *logger.Logger
}

func NewRouter(listener string, routerFactory *route.Factory, upstreamFactory *upstream.Factory, parentLogger *logger.Logger) *Router {
	return &Router{
		pathMatcher:     route.NewPathMatcher(parentLogger),
		methodMatcher:   route.NewMethodMatcher(),
		headerMatcher:   route.NewHeaderMatcher(),
		upstreamFactory: upstreamFactory,
		pluginChain:     []*plugin.Chain{},
		logger:          parentLogger.WithComponent("router"),
	}
}

func (r *Router) UpdateRoutes(routeConfigs []config.RouteConfig, upstreamConfigs []config.UpstreamConfig, pluginConfigs []config.PluginConfig) error {
	// Clear existing matchers
	r.pathMatcher.Clear()
	r.methodMatcher.Clear()
	r.headerMatcher.Clear()
	//cleanup plugin
	for _, p := range r.pluginChain {
		p.Destroy()
	}
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
				plugin := pluginFactory(r.logger)
				r.logger.Infof("Adding plugin %s to route %s", pCfg.Name, rc.Name)
				err := plugin.ValidateAndSetConfig(pCfg.Config)
				if err != nil {
					r.logger.Warnf("Invalid config for plugin %s in route %s: %v", pCfg.Name, rc.Name, err)
					continue
				}
				pluginChain.Add(plugin)
			} else {
				r.logger.Warnf("Plugin type %s not found for plugin %s in route %s", pCfg.Type, pCfg.Name, rc.Name)
			}
		}
		r.pluginChain = append(r.pluginChain, pluginChain)

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
			} else {
				r.methodMatcher.Add("GET", route)
				r.methodMatcher.Add("POST", route)
				r.methodMatcher.Add("PUT", route)
				r.methodMatcher.Add("DELETE", route)
				r.methodMatcher.Add("PATCH", route)
				r.methodMatcher.Add("HEAD", route)
				r.methodMatcher.Add("OPTIONS", route)
			}
			if len(match.Headers) > 0 {
				r.headerMatcher.Add(match.Headers, route)
			} else {
				r.headerMatcher.Add(nil, route) // Match all headers
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
	if len(methodRoutes) == 0 {
		http.NotFound(w, req)
		return
	}
	// Step 3: Find intersection of path and method routes
	candidateRoutes := route.IntersectRoutes(pathRoutes, methodRoutes)
	if len(candidateRoutes) == 0 {
		http.NotFound(w, req)
		return
	}

	// Step 4: Match by headers if needed
	finalRoutes := r.headerMatcher.Match(req.Header, candidateRoutes)
	if len(finalRoutes) == 0 {
		http.NotFound(w, req)
		return
	}
	// For simplicity, pick the first matched route
	route := finalRoutes[0]

	finished, err := route.Plugins.HandleRequest(req, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if finished {
		return
	}
	node := route.Upstream.SelectNode()
	if node == nil {
		http.Error(w, "No available upstream nodes", http.StatusServiceUnavailable)
		return
	}

	wrappedWriter := route.Plugins.HandleResponse(req, w)
	proxy := proxy.NewReverseProxy()
	proxy.ServeHTTP(wrappedWriter, req, node.URL)
}
