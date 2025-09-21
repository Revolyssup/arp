package plugin

import (
	"net/http"
	"sort"
)

type Plugin interface {
	HandleRequest(*http.Request) error
	HandleResponse(*http.Response) error
	Priority() int
}

type Chain struct {
	plugins []Plugin
}

func NewChain() *Chain {
	return &Chain{}
}

func (c *Chain) Add(p Plugin) {
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

func (c *Chain) HandleResponse(res *http.Response) error {
	// Process in reverse order
	for i := len(c.plugins) - 1; i >= 0; i-- {
		if err := c.plugins[i].HandleResponse(res); err != nil {
			return err
		}
	}
	return nil
}
