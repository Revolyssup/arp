package listener

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/router"
)

type Listener struct {
	config config.ListenerConfig
	router *router.HTTPRouter
	server *http.Server
}

func NewListener(cfg config.ListenerConfig, discoveryManager *discovery.DiscoveryManager, eventBus *eventbus.EventBus[config.Dynamic]) *Listener {
	l := &Listener{
		config: cfg,
		router: router.NewHTTPRouter(cfg.Name, discoveryManager),
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
	log.Print("Updating routes for listener ", l.config.Name)
	l.router.UpdateRoutes(routes, upstreams, plugins)
}

func (l *Listener) Stop() error {
	return l.server.Shutdown(context.Background())
}
