package watcher

import (
	"context"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/provider"
	"github.com/Revolyssup/arp/pkg/provider/file"
)

// Provider-> Watcher -> Processor
// Watcher is mainly responsbile for throttling.
type Processor interface {
	Process(config.Dynamic)
}

type Watcher struct {
	receiveChan chan config.Dynamic
	applyChan   chan config.Dynamic
	Processor   Processor
	logger      *logger.Logger
}

func NewWatcher(providers []config.ProviderConfig, processor Processor, logger *logger.Logger) *Watcher {
	if len(providers) == 0 {
		return nil
	}
	watcher := &Watcher{
		receiveChan: make(chan config.Dynamic, 10), // Buffered channel
		applyChan:   make(chan config.Dynamic, 10), // Buffered channel
		Processor:   processor,
		logger:      logger.WithComponent("watcher"),
	}
	for _, pCfg := range providers {
		var p provider.Provider
		var err error
		switch pCfg.Type {
		case "file":
			p, err = file.NewFileProvider(pCfg, logger)
		default:
			continue // Unsupported provider type
		}
		if err != nil {
			logger.Errorf("Failed to create provider for %s: %v", pCfg.Name, err)
			continue
		}
		go p.Provide(watcher.receiveChan)
	}
	return watcher
}

// Throttling mechanism inspired by configwatcher in Traefik :) (Though a simplified version)
func (w *Watcher) Watch(ctx context.Context) {
	go func() {
		for dynCfg := range w.applyChan {
			w.Processor.Process(dynCfg)
		}
	}()
	var output chan config.Dynamic
	latestConfiguration := config.Dynamic{}
	for {
		select {
		case <-ctx.Done():
			w.logger.Warn("Watcher received shutdown signal")
			return
		case output <- latestConfiguration:
			output = nil
		default:
			select {
			case <-ctx.Done():
				w.logger.Warn("Watcher received shutdown signal")
				return
			case dynCfg := <-w.receiveChan:
				latestConfiguration = dynCfg
				output = w.applyChan
			case output <- latestConfiguration:
				output = nil
			}
		}
	}
}
