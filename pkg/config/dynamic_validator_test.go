package config

import "testing"

func TestDynamicValidati(t *testing.T) {
	validator := NewDynamicValidator()

	tests := []struct {
		name    string
		cfg     Dynamic
		wantErr bool
	}{
		{
			name: "valid configuration",
			cfg: Dynamic{
				Routes: []RouteConfig{
					{Name: "route1", Listener: "listener1", Upstream: &UpstreamConfig{Name: "upstream1"}, Matches: []Match{{Path: "/path1"}}},
				},
				Upstreams: []UpstreamConfig{
					{Name: "upstream1", Nodes: []Node{{URL: "http://example.com"}}},
				},
				Plugins: []PluginConfig{
					{Name: "plugin1", Type: "type1", Config: map[string]interface{}{"key": "value"}},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate names",
			cfg: Dynamic{
				Routes: []RouteConfig{
					{Name: "dupName", Listener: "listener1", Upstream: &UpstreamConfig{Name: "upstream1"}, Matches: []Match{{Path: "/path1"}}},
					{Name: "dupName", Listener: "listener2", Upstream: &UpstreamConfig{Name: "upstream2"}, Matches: []Match{{Path: "/path2"}}},
				},
				Upstreams: []UpstreamConfig{
					{Name: "upstream1", Nodes: []Node{{URL: "http://example.com"}}},
					{Name: "upstream2", Nodes: []Node{{URL: "http://example.org"}}},
				},
				Plugins: []PluginConfig{
					{Name: "plugin1", Type: "type1", Config: map[string]interface{}{"key": "value"}},
				},
			},
			wantErr: true,
		},
		{
			name: "empty route name",
			cfg: Dynamic{
				Routes: []RouteConfig{
					{Name: "", Listener: "listener1", Upstream: &UpstreamConfig{Name: "upstream1"}, Matches: []Match{{Path: "/test"}}},
				},
				Upstreams: []UpstreamConfig{
					{Name: "upstream1", Nodes: []Node{{URL: "http://example.com"}}},
				},
				Plugins: []PluginConfig{
					{Name: "plugin1", Type: "type1", Config: map[string]interface{}{"key": "value"}},
				},
			},
			wantErr: true,
		},
		{
			name: "empty listener name",
			cfg: Dynamic{
				Routes: []RouteConfig{
					{Name: "route1", Listener: "", Upstream: &UpstreamConfig{Name: "upstream1"}, Matches: []Match{{Path: "/test"}}},
				},
				Upstreams: []UpstreamConfig{
					{Name: "upstream1", Nodes: []Node{{URL: "http://example.com"}}},
				},
				Plugins: []PluginConfig{
					{Name: "plugin1", Type: "type1", Config: map[string]interface{}{"key": "value"}},
				},
			},
			wantErr: true,
		},
		{
			name: "no matches in route",
			cfg: Dynamic{
				Routes: []RouteConfig{
					{Name: "route1", Listener: "listener1", Upstream: &UpstreamConfig{Name: "upstream1"}, Matches: []Match{}},
				},
				Upstreams: []UpstreamConfig{
					{Name: "upstream1", Nodes: []Node{{URL: "http://example.com"}}},
				},
				Plugins: []PluginConfig{
					{Name: "plugin1", Type: "type1", Config: map[string]interface{}{"key": "value"}},
				},
			},
			wantErr: true,
		},
		{
			name: "plugin type cannot be empty",
			cfg: Dynamic{
				Routes: []RouteConfig{
					{Name: "route1", Listener: "listener1", Upstream: &UpstreamConfig{Name: "upstream1"}, Matches: []Match{{Path: "/test"}}},
				},
				Upstreams: []UpstreamConfig{
					{Name: "upstream1", Nodes: []Node{{URL: "http://example.com"}}},
				},
				Plugins: []PluginConfig{
					{Name: "plugin1", Type: "", Config: map[string]interface{}{"key": "value"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
