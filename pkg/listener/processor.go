package listener

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/eventbus"
	"github.com/Revolyssup/arp/pkg/watcher"
)

type ListenerProcessor struct {
	eventBus       *eventbus.EventBus[config.Dynamic]
	listenerHashes map[string]string // Maps listener name to config hash
}

func NewListenerProcessor(eventBus *eventbus.EventBus[config.Dynamic]) watcher.Processor {
	return &ListenerProcessor{
		eventBus:       eventBus,
		listenerHashes: make(map[string]string),
	}
}

// processConfig helps to send each provider config for the listener that it's specifically subscribed to.
// Processor -> EventBus -> Listener -> Router
func (w *ListenerProcessor) Process(dynCfg config.Dynamic) {
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

	pluginMap := make(map[string]config.PluginConfig)
	for _, plugin := range dynCfg.Plugins {
		pluginMap[plugin.Name] = plugin
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

		pluginConfigs := make([]config.PluginConfig, 0, len(pluginMap))
		for _, route := range routes {
			for _, p := range route.Plugins {
				if pl, exists := pluginMap[p.Name]; exists {
					pluginConfigs = append(pluginConfigs, pl)
				}
			}
		}
		listenerConfig := config.Dynamic{
			Routes:    routes,
			Upstreams: upstreamConfigs,
			Plugins:   pluginConfigs,
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

func (w *ListenerProcessor) calculateHash(cfg config.Dynamic) (string, error) {
	// Convert config to JSON for hashing
	configBytes, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	// Calculate MD5 hash
	hash := md5.Sum(configBytes)
	return fmt.Sprintf("%x", hash), nil
}
