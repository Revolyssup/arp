package listener

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/route"
	httprouter "github.com/Revolyssup/arp/pkg/router/http"
	"github.com/Revolyssup/arp/pkg/upstream"
)

type Listener struct {
	config config.ListenerConfig
	router *httprouter.Router
	server *http.Server
	logger *logger.Logger
}

func NewListener(cfg config.ListenerConfig, discoveryManager *discovery.DiscoveryManager, eventBus *eventbus.EventBus[config.Dynamic], routerFactory *route.Factory, upstreamFactory *upstream.Factory, logger *logger.Logger) *Listener {
	l := &Listener{
		config: cfg,
		router: httprouter.NewRouter(cfg.Name, routerFactory, upstreamFactory, logger),
		logger: logger,
	}
	l.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: l.router,
	}
	go func() {
		for dynCfg := range eventBus.Subscribe(cfg.Name) {
			l.updateRoutes(dynCfg.Routes, dynCfg.Upstreams, dynCfg.Plugins)
		}
	}()
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

func (l *Listener) Stop(ctx context.Context) error {
	return l.server.Shutdown(ctx)
}
