package plugin

import (
	"net/http"
	"sort"

	"github.com/Revolyssup/arp/pkg/plugin/demo"
	"github.com/Revolyssup/arp/pkg/plugin/responsecache"
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

func (c *Chain) Destroy() {
	for _, p := range c.plugins {
		p.Destroy()
	}
}

func (c *Chain) HandleRequest(req *http.Request, res http.ResponseWriter) (bool, error) {
	for _, p := range c.plugins {
		if final, err := p.HandleRequest(req, res); err != nil {
			return final, err
		} else if final {
			return true, nil
		}
	}
	return false, nil
}

func (c *Chain) HandleResponse(req *http.Request, w http.ResponseWriter) http.ResponseWriter {
	// Wrap in reverse order so the first plugin in the chain
	// becomes the innermost wrapper (executed last on response)
	wrapped := w
	for i := len(c.plugins) - 1; i >= 0; i-- {
		wrapped = c.plugins[i].HandleResponse(req, wrapped)
	}
	return wrapped
}

// For now just have a local registry of plugins which will be a singleton
// instantiated on startup. Later we can have dynamic registration of plugins.

var Registry *types.Registry

func init() {
	Registry = types.NewRegistry()
	Registry.Register("demo", demo.NewPlugin)
	Registry.Register("responsecache", responsecache.NewPlugin)
}
