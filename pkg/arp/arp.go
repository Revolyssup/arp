package arp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/listener"
	"github.com/Revolyssup/arp/pkg/route"
	"github.com/Revolyssup/arp/pkg/upstream"
	"github.com/Revolyssup/arp/pkg/watcher"
	"gopkg.in/yaml.v3"
)

// ARP represents the main application instance
type ARP struct {
	config     *config.Static
	listeners  map[string]*listener.Listener
	watcher    *watcher.Watcher
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

// NewARP creates a new ARP instance with the given configuration file
func NewARP(configFile string) (*ARP, error) {
	staticConfig, err := loadStaticConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &ARP{
		config: staticConfig,
	}, nil
}

func (a *ARP) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// Initialize components
	if err := a.init(); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	if err := a.start(ctx); err != nil {
		return fmt.Errorf("failed to start components: %w", err)
	}

	log.Printf("ARP server started successfully with %d listeners", len(a.listeners))

	// Wait for shutdown signal
	a.waitForShutdown(ctx)

	// Perform graceful shutdown
	a.shutdown()

	return nil
}

func (a *ARP) init() error {
	configBus := eventbus.NewEventBus[config.Dynamic]()

	discoveryManager := discovery.NewDiscoveryManager(a.config.DiscoveryConfigs)

	routeFactory := route.NewFactory()
	upstreamFactory := upstream.NewFactory(discoveryManager)

	a.listeners = make(map[string]*listener.Listener)
	for _, lc := range a.config.Listeners {
		l := listener.NewListener(lc, discoveryManager, configBus, routeFactory, upstreamFactory)
		a.listeners[lc.Name] = l
	}

	listenerProcessor := listener.NewListenerProcessor(configBus)
	a.watcher = watcher.NewWatcher(a.config.Providers, listenerProcessor)

	return nil
}

func (a *ARP) start(ctx context.Context) error {
	a.wg.Add(1)
	// Start configuration watcher
	go func() {
		defer a.wg.Done()
		a.watcher.Watch(ctx)
		log.Println("Configuration watcher stopped")
	}()

	// Start listeners
	for name, l := range a.listeners {
		a.wg.Add(1)
		go func(name string, l *listener.Listener) {
			defer a.wg.Done()
			log.Printf("Starting listener: %s", name)

			if err := l.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("Listener %s failed: %v", name, err)
			} else {
				log.Printf("Listener %s stopped", name)
			}
		}(name, l)
	}

	return nil
}

// waitForShutdown waits for a shutdown signal
func (a *ARP) waitForShutdown(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case <-ctx.Done():
		log.Printf("Context cancelled: %v", ctx.Err())
	}
}

// shutdown performs graceful shutdown of all components
func (a *ARP) shutdown() {
	log.Println("Initiating graceful shutdown...")

	// Cancel the main context to signal all components to stop
	if a.cancelFunc != nil {
		a.cancelFunc()
	}

	// Stop all listeners
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for name, l := range a.listeners {
		wg.Add(1)
		go func(name string, l *listener.Listener) {
			defer wg.Done()
			log.Printf("Stopping listener: %s", name)
			if err := l.Stop(shutdownCtx); err != nil {
				log.Printf("Error stopping listener %s: %v", name, err)
			}
		}(name, l)
	}

	// Wait for all listeners to stop
	done := make(chan struct{})
	go func() {
		wg.Wait()
		a.wg.Wait()
		close(done)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case <-done:
		log.Println("Shutdown completed successfully")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing exit")
	}
}

// loadStaticConfig loads the static configuration from file
func loadStaticConfig(filename string) (*config.Static, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Static
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return &cfg, nil
}
