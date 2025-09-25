package watcher

import (
	"context"
	"log"

	"github.com/Revolyssup/arp/pkg/config"
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
}

func NewWatcher(providers []config.ProviderConfig, processor Processor) *Watcher {
	if len(providers) == 0 {
		return nil
	}
	watcher := &Watcher{
		receiveChan: make(chan config.Dynamic, 10), // Buffered channel
		applyChan:   make(chan config.Dynamic, 10), // Buffered channel
		Processor:   processor,
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
			log.Println("Watcher received shutdown signal")
			return
		case output <- latestConfiguration:
			output = nil
		default:
			select {
			case <-ctx.Done():
				log.Println("Watcher received shutdown signal")
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
