package listener

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery/manager"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/route"
	httprouter "github.com/Revolyssup/arp/pkg/router/http"
	"github.com/Revolyssup/arp/pkg/types"
	"github.com/Revolyssup/arp/pkg/upstream"
	"github.com/Revolyssup/arp/pkg/utils"
)

type Listener struct {
	config config.ListenerConfig
	router *httprouter.Router
	server *http.Server
	logger *logger.Logger
}

func NewListener(cfg config.ListenerConfig, discoveryManager *manager.DiscoveryManager, eventBus *eventbus.EventBus[config.Dynamic], routerFactory *route.Factory, upstreamFactory *upstream.Factory, logger *logger.Logger) *Listener {
	l := &Listener{
		config: cfg,
		router: httprouter.NewRouter(cfg.Name, routerFactory, upstreamFactory, discoveryManager, logger),
		logger: logger,
	}
	l.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: l.router,
	}
	utils.GoWithRecover(func() {
		for dynCfg := range eventBus.Subscribe(types.RouteEventKey(cfg.Name)) {
			l.updateRoutes(dynCfg.Routes, dynCfg.Upstreams, dynCfg.Plugins)
		}
		for dynCfg := range eventBus.Subscribe(types.StreamRouteEventKey(cfg.Name)) {
			l.updateStreamRoutes(dynCfg.StreamRoute, dynCfg.Upstreams, dynCfg.Plugins)
		}
	}, func(a any) {
		l.logger.Errorf("panic in route update listener for listener %s: %v", cfg.Name, a)
	})
	return l
}

func (l *Listener) Start() error {
	if l.config.TLS != nil {
		return l.server.ListenAndServeTLS(l.config.TLS.CertFile, l.config.TLS.KeyFile)
	}
	return l.server.ListenAndServe()
}

// TODO: Refactor the updation logic from this ugly mess of passing each config type separately.
func (l *Listener) updateRoutes(routes []config.RouteConfig, upstreams []config.UpstreamConfig, plugins []config.PluginConfig) {
	l.logger.Infof("Updating routes for listener %s", l.config.Name)
	l.router.UpdateRoutes(routes, upstreams, plugins)
}

func (l *Listener) updateStreamRoutes(streamRoutes []config.StreamRouteConfig, upstreams []config.UpstreamConfig, plugins []config.PluginConfig) {
	l.logger.Infof("Updating stream routes for listener %s", l.config.Name)
	// l.router.UpdateStreamRoutes(streamRoutes, upstreams, plugins)
}
func (l *Listener) Stop(ctx context.Context) error {
	return l.server.Shutdown(ctx)
}
