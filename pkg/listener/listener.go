package listener

import (
	"context"
	"crypto/tls"
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
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	var handler http.Handler = l.router
	if cfg.HTTP2 && cfg.TLS == nil {
		handler = h2c.NewHandler(l.router, &http2.Server{})
	}

	l.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: handler,
	}
	if cfg.TLS != nil {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			logger.Errorf("Failed to load TLS certificate: %v", err)
		}

		l.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		}
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
