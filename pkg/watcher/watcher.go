package watcher

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/provider"
	"github.com/Revolyssup/arp/pkg/provider/file"
)

type Watcher struct {
	eventBus       *eventbus.EventBus[config.Dynamic]
	receiveChan    chan config.Dynamic
	listenerHashes map[string]string // Maps listener name to config hash
}

func NewWatcher(providers []config.ProviderConfig, eventBus *eventbus.EventBus[config.Dynamic]) *Watcher {
	if len(providers) == 0 {
		return nil
	}
	watcher := &Watcher{
		eventBus:       eventBus,
		receiveChan:    make(chan config.Dynamic, 10), // Buffered channel
		listenerHashes: make(map[string]string),
	}
	for _, pCfg := range providers {
		var p provider.Provider
		switch pCfg.Type {
		case "file":
			p = file.NewFileProvider(pCfg)
		default:
			continue // Unsupported provider type
		}
		go p.Provide(watcher.receiveChan)
	}
	return watcher
}

func (w *Watcher) Watch() {
	for dynCfg := range w.receiveChan {
		w.processConfig(dynCfg)
	}
}

// processConfig helps to send each provider config for the listener that it's specifically subscribed to.
func (w *Watcher) processConfig(dynCfg config.Dynamic) {
	// Group routes by listener
	listenerRoutes := make(map[string][]config.RouteConfig)
	for _, route := range dynCfg.Routes {
		listenerRoutes[route.Listener] = append(listenerRoutes[route.Listener], route)
	}

	// Group upstreams by routes that reference them
	upstreamMap := make(map[string]config.UpstreamConfig)
	for _, upstream := range dynCfg.Upstreams {
		upstreamMap[upstream.Name] = upstream
	}
	for listenerName, routes := range listenerRoutes {
		upstreamConfigs := make([]config.UpstreamConfig, 0, len(upstreamMap))
		for _, route := range routes {
			if route.Upstream != nil && route.Upstream.Name != "" {
				if up, exists := upstreamMap[route.Upstream.Name]; exists {
					upstreamConfigs = append(upstreamConfigs, up)
				}
			}
		}
		listenerConfig := config.Dynamic{
			Routes:    routes,
			Upstreams: upstreamConfigs,
		}
		hash, err := w.calculateHash(listenerConfig)
		if err != nil {
			log.Printf("Error calculating hash for listener %s: %v", listenerName, err)
			continue
		}

		if prevHash, exists := w.listenerHashes[listenerName]; !exists || prevHash != hash {
			// Config has changed, publish to event bus
			w.eventBus.Publish(listenerName, listenerConfig)
			w.listenerHashes[listenerName] = hash
			log.Printf("Published updated config for listener: %s", listenerName)
		}
	}

	// Check for listeners that have been removed
	for listenerName := range w.listenerHashes {
		if _, exists := listenerRoutes[listenerName]; !exists {
			// Listener has been removed, publish empty config
			w.eventBus.Publish(listenerName, config.Dynamic{})
			delete(w.listenerHashes, listenerName)
			log.Printf("Published empty config for removed listener: %s", listenerName)
		}
	}
}

func (w *Watcher) calculateHash(cfg config.Dynamic) (string, error) {
	// Convert config to JSON for hashing
	configBytes, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	// Calculate MD5 hash
	hash := md5.Sum(configBytes)
	return fmt.Sprintf("%x", hash), nil
}
