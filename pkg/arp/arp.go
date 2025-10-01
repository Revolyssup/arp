package arp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/discovery/manager"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/listener"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/route"
	"github.com/Revolyssup/arp/pkg/upstream"
	"github.com/Revolyssup/arp/pkg/utils"
	"github.com/Revolyssup/arp/pkg/watcher"
	"gopkg.in/yaml.v3"
)

// ASCII Art for ARP banner
const arpBanner = `
    █████╗ ██████╗ ██████╗ 
   ██╔══██╗██╔══██╗██╔══██╗
   ███████║██████╔╝██████╔╝
   ██╔══██║██╔══██╗██╔═══╝ 
   ██║  ██║██║  ██║██║     
   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝      

 Another Reverse Proxy
         starting...
`

func printBanner() {
	// Print directly to stdout to avoid logger prefixes for the banner
	fmt.Print("\033[1;36m") // Cyan color
	fmt.Print(arpBanner)
	fmt.Print("\033[0m") // Reset color
	fmt.Println()
}

// ARP represents the main application instance
type ARP struct {
	config     *config.Static
	log        *logger.Logger
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
	level := logger.SetLogLevel(staticConfig.LogLevel)
	log := logger.New(level).WithComponent("arp")

	return &ARP{
		config: staticConfig,
		log:    log,
	}, nil
}

func (a *ARP) Run(ctx context.Context) error {
	printBanner()
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	// Initialize components
	if err := a.init(); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	if err := a.start(ctx); err != nil {
		return fmt.Errorf("failed to start components: %w", err)
	}

	a.log.Infof("ARP server started successfully with %d listeners", len(a.listeners))

	// Wait for shutdown signal
	a.waitForShutdown(ctx)

	// Perform graceful shutdown
	a.shutdown()

	return nil
}

func (a *ARP) init() error {
	configBus := eventbus.NewEventBus[config.Dynamic](a.log.WithComponent("config_bus"))

	discoveryManager, err := manager.NewDiscoveryManager(a.config.DiscoveryConfigs, a.log)
	if err != nil {
		return fmt.Errorf("failed to initialize discovery manager: %w", err)
	}
	routeFactory := route.NewFactory()
	upstreamFactory := upstream.NewFactory()

	a.listeners = make(map[string]*listener.Listener)
	for _, lc := range a.config.Listeners {
		l := listener.NewListener(lc, discoveryManager, configBus, routeFactory, upstreamFactory, a.log.WithComponent("listener_"+lc.Name))
		a.listeners[lc.Name] = l
	}

	dynamicValidator := config.NewDynamicValidator()
	listenerProcessor := listener.NewListenerProcessor(configBus, dynamicValidator, a.log.WithComponent("listener_processor"))
	a.watcher = watcher.NewWatcher(a.config.Providers, listenerProcessor, a.log.WithComponent("watcher"))

	return nil
}

func (a *ARP) start(ctx context.Context) error {
	a.wg.Add(1)
	// Start configuration watcher
	utils.GoWithRecover(func() {
		defer a.wg.Done()
		a.watcher.Watch(ctx)
		a.log.Info("Configuration watcher stopped")
	}, func(err any) {
		a.log.Errorf("panic in configuration watcher: %v", err)
	})

	// Start listeners
	for name, l := range a.listeners {
		a.wg.Add(1)
		// for compatibility with older go versions
		name := name
		l := l
		utils.GoWithRecover(func() {
			defer a.wg.Done()
			a.log.Infof("Starting listener: %s", name)

			if err := l.Start(); err != nil && err != http.ErrServerClosed {
				a.log.Errorf("Listener %s failed: %v", name, err)
			} else {
				a.log.Infof("Listener %s stopped", name)
			}
		}, func(err any) {
			a.log.Errorf("panic in listener %s: %v", name, err)
		})
	}

	return nil
}

// waitForShutdown waits for a shutdown signal
func (a *ARP) waitForShutdown(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case sig := <-sigChan:
		a.log.Infof("Received signal: %v", sig)
	case <-ctx.Done():
		a.log.Infof("Context cancelled: %v", ctx.Err())
	}
}

// shutdown performs graceful shutdown of all components
func (a *ARP) shutdown() {
	a.log.Info("Initiating graceful shutdown...")

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
		// for compatibility with older go versions
		name := name
		l := l
		utils.GoWithRecover(func() {
			defer wg.Done()
			a.log.Infof("Stopping listener: %s", name)
			if err := l.Stop(shutdownCtx); err != nil {
				a.log.Errorf("Error stopping listener %s: %v", name, err)
			}
		}, func(err any) {
			a.log.Errorf("panic while stopping listener %s: %v", name, err)
		})
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
		a.log.Info("Shutdown completed successfully")
	case <-shutdownCtx.Done():
		a.log.Warn("Shutdown timeout exceeded, forcing exit")
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

	validator := config.NewStaticValidator()
	if err := validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}
	return &cfg, nil
}
