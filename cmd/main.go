package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/listener"
	"github.com/Revolyssup/arp/pkg/watcher"
	"gopkg.in/yaml.v3"
)

var filename = "./static.yaml"

func init() {
	configPath := os.Getenv("ARP_CONFIG")
	if configPath != "" {
		filename = configPath
	}
}
func main() {
	staticConfig, err := loadStaticConfig(filename)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	configBus := eventbus.NewEventBus[config.Dynamic]()

	listeners := make(map[string]*listener.Listener)
	for _, lc := range staticConfig.Listeners {
		l := listener.NewListener(lc, configBus)
		listeners[lc.Name] = l
	}

	cfgWatcher := watcher.NewWatcher(staticConfig.Providers, configBus)
	go cfgWatcher.Watch()

	for name, l := range listeners {
		go func(name string, l *listener.Listener) {
			if err := l.Start(); err != nil {
				log.Printf("Listener %s failed: %v", name, err)
			}
		}(name, l)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	for _, l := range listeners {
		l.Stop()
	}
}

func loadStaticConfig(filename string) (*config.Static, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg config.Static
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
