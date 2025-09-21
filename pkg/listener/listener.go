package listener

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/router"
)

type Listener struct {
	config config.ListenerConfig
	router *router.HTTPRouter
	server *http.Server
}

func NewListener(cfg config.ListenerConfig, eventBus *eventbus.EventBus[config.Dynamic]) *Listener {
	l := &Listener{
		config: cfg,
		router: router.NewHTTPRouter(cfg.Name),
	}
	l.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: l.router,
	}
	go func() {
		for dynCfg := range eventBus.Subscribe(cfg.Name) {
			l.updateRoutes(dynCfg.Routes)
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

func (l *Listener) updateRoutes(routes []config.RouteConfig) {
	log.Print("Updating routes for listener ", l.config.Name)
	l.router.UpdateRoutes(routes)
}

func (l *Listener) Stop() error {
	return l.server.Shutdown(context.Background())
}
