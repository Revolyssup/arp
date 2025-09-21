package config

type Static struct {
	Listeners   []ListenerConfig `yaml:"listeners"`
	Providers   []ProviderConfig `yaml:"providers"`
	Discoveries []DiscoveryRef   `yaml:"discoveries,omitempty"`
}

type ListenerConfig struct {
	Name string     `yaml:"name"`
	Port int        `yaml:"port"`
	TLS  *TLSConfig `yaml:"tls,omitempty"`
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
