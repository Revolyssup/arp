package plugin

import (
	"net/http"
	"sort"

	"github.com/Revolyssup/arp/pkg/plugin/demo"
	"github.com/Revolyssup/arp/pkg/plugin/types"
)

type Chain struct {
	plugins []types.Plugin
}

func NewChain() *Chain {
	return &Chain{
		plugins: []types.Plugin{},
	}
}

func (c *Chain) Add(p types.Plugin) {
	c.plugins = append(c.plugins, p)
}

func (c *Chain) Sort() {
	sort.Slice(c.plugins, func(i, j int) bool {
		return c.plugins[i].Priority() < c.plugins[j].Priority()
	})
}

func (c *Chain) HandleRequest(req *http.Request) error {
	for _, p := range c.plugins {
		if err := p.HandleRequest(req); err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) WrapResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	// Wrap in reverse order so the first plugin in the chain
	// becomes the innermost wrapper (executed last on response)
	wrapped := w
	for i := len(c.plugins) - 1; i >= 0; i-- {
		wrapped = c.plugins[i].WrapResponseWriter(wrapped)
	}
	return wrapped
}

// For now just have a local registry of plugins which will be a singleton
// instantiated on startup. Later we can have dynamic registration of plugins.

var Registry *types.Registry

func init() {
	Registry = types.NewRegistry()
	Registry.Register("demo", demo.NewPlugin())
}
