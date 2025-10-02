package config

type Static struct {
	Listeners        []ListenerConfig  `yaml:"listeners"`
	Providers        []ProviderConfig  `yaml:"providers"`
	DiscoveryConfigs []DiscoveryConfig `yaml:"discovery"`
	LogLevel         string            `yaml:"log_level"`
}

type ListenerConfig struct {
	Name  string     `yaml:"name"`
	Port  int        `yaml:"port"`
	TLS   *TLSConfig `yaml:"tls,omitempty"`
	HTTP2 bool       `yaml:"http2,omitempty"`
}

type TLSConfig struct {
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

type ProviderConfig struct {
	Name   string                 `yaml:"name"`
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

type DiscoveryConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}
