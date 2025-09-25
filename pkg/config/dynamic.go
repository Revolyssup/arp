package config

import "github.com/Revolyssup/arp/pkg/plugin/types"

type Dynamic struct {
	Routes    []RouteConfig    `yaml:"routes"`
	Upstreams []UpstreamConfig `yaml:"upstreams,omitempty"`
	Plugins   []PluginConfig   `yaml:"plugins,omitempty"`
}

type RouteConfig struct {
	Name     string          `yaml:"name"`
	Listener string          `yaml:"listener"`
	Matches  []Match         `yaml:"matches"`
	Plugins  []PluginConfig  `yaml:"plugins,omitempty"`
	Upstream *UpstreamConfig `yaml:"upstream,omitempty"`
}

type Match struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Method  string            `yaml:"method,omitempty"`
}

type UpstreamConfig struct {
	Name      string       `yaml:"name"`
	Type      string       `yaml:"type"`
	Nodes     []Node       `yaml:"nodes,omitempty"`
	Service   string       `yaml:"service,omitempty"`
	Discovery DiscoveryRef `yaml:"discovery,omitempty"`
}

type Node struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight,omitempty"`
}

type DiscoveryRef struct {
	Type   string            `yaml:"type"`
	Params map[string]string `yaml:"params,omitempty"`
}

type PluginConfig struct {
	Name   string           `yaml:"name"`
	Type   string           `yaml:"type"`
	Config types.PluginConf `yaml:"config"`
}
