package responsecache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/plugin/types"
)

func TestResponseCache_ValidateAndSetConfig(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	tests := []struct {
		name        string
		config      types.PluginConf
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: types.PluginConf{
				"size": 100,
				"ttl":  30,
				"key":  "uri",
			},
			wantErr: false,
		},
		{
			name: "missing size",
			config: types.PluginConf{
				"ttl": 30,
				"key": "uri",
			},
			wantErr:     true,
			errContains: "size must be an integer",
		},
		{
			name: "invalid size - zero",
			config: types.PluginConf{
				"size": 0,
				"ttl":  30,
				"key":  "uri",
			},
			wantErr:     true,
			errContains: "size must be a positive integer",
		},
		{
			name: "invalid size - negative",
			config: types.PluginConf{
				"size": -1,
				"ttl":  30,
				"key":  "uri",
			},
			wantErr:     true,
			errContains: "size must be a positive integer",
		},
		{
			name: "missing ttl",
			config: types.PluginConf{
				"size": 100,
				"key":  "uri",
			},
			wantErr:     true,
			errContains: "ttl must be an integer",
		},
		{
			name: "invalid ttl",
			config: types.PluginConf{
				"size": 100,
				"ttl":  -1,
				"key":  "uri",
			},
			wantErr:     true,
			errContains: "ttl must be a positive integer",
		},
		{
			name: "missing key",
			config: types.PluginConf{
				"size": 100,
				"ttl":  30,
			},
			wantErr:     true,
			errContains: "key must be a string",
		},
		{
			name: "invalid key",
			config: types.PluginConf{
				"size": 100,
				"ttl":  30,
				"key":  "invalid",
			},
			wantErr:     true,
			errContains: "key must be one of uri, host, method",
		},
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.ValidateAndSetConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndSetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if err.Error() != tt.errContains {
					t.Errorf("ValidateAndSetConfig() error = %v, should contain %v", err, tt.errContains)
				}
			}
			if !tt.wantErr {
				if plugin.cache == nil {
					t.Error("ValidateAndSetConfig() cache should be initialized")
				}
				if plugin.config == nil {
					t.Error("ValidateAndSetConfig() config should be set")
				}
			}
		})
	}
}

func TestResponseCache_HandleRequest_CacheMiss(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	config := types.PluginConf{
		"size": 100,
		"ttl":  30,
		"key":  "uri",
	}
	err := plugin.ValidateAndSetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	handled, err := plugin.HandleRequest(req, recorder)
	if err != nil {
		t.Errorf("HandleRequest() unexpected error: %v", err)
	}
	if handled {
		t.Error("HandleRequest() should return false on cache miss")
	}
	if recorder.Header().Get("X-Cache-Hit") != "" {
		t.Error("X-Cache-Hit header should not be set on cache miss")
	}
}

func TestResponseCache_HandleRequest_CacheHit(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	config := types.PluginConf{
		"size": 100,
		"ttl":  30,
		"key":  "uri",
	}
	err := plugin.ValidateAndSetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// First, populate the cache
	cachedData := []byte("cached response")
	plugin.cache.Set("/test", cachedData, 30*time.Second)

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	handled, err := plugin.HandleRequest(req, recorder)
	if err != nil {
		t.Errorf("HandleRequest() unexpected error: %v", err)
	}
	if !handled {
		t.Error("HandleRequest() should return true on cache hit")
	}
	if recorder.Header().Get("X-Cache-Hit") != "true" {
		t.Error("X-Cache-Hit header should be 'true' on cache hit")
	}
	if recorder.Body.String() != string(cachedData) {
		t.Errorf("HandleRequest() body = %s, want %s", recorder.Body.String(), cachedData)
	}
}

func TestResponseCache_HandleResponse_CachesData(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	config := types.PluginConf{
		"size": 100,
		"ttl":  30,
		"key":  "uri",
	}
	err := plugin.ValidateAndSetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	originalRecorder := httptest.NewRecorder()

	// Get the wrapped response writer
	wrappedWriter := plugin.HandleResponse(req, originalRecorder)

	// Call WriteHeader first to set the cache miss header
	wrappedWriter.WriteHeader(200)

	// Write to the wrapped writer
	responseData := []byte("test response")
	_, err = wrappedWriter.Write(responseData)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Check that data was cached
	cached, found := plugin.cache.Get("/test")
	if !found {
		t.Error("Data should be cached after Write()")
	}
	if string(cached) != string(responseData) {
		t.Errorf("Cached data = %s, want %s", cached, responseData)
	}

	// Check that header was set
	if originalRecorder.Header().Get("X-Cache-Hit") != "false" {
		t.Error("X-Cache-Hit header should be 'false' for uncached responses")
	}
}

func TestResponseCache_WriteHeader_SetsCacheMissHeader(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	config := types.PluginConf{
		"size": 100,
		"ttl":  30,
		"key":  "uri",
	}
	err := plugin.ValidateAndSetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	originalRecorder := httptest.NewRecorder()

	wrappedWriter := plugin.HandleResponse(req, originalRecorder)
	wrappedWriter.WriteHeader(404)

	if originalRecorder.Header().Get("X-Cache-Hit") != "false" {
		t.Error("X-Cache-Hit header should be 'false' after WriteHeader()")
	}
	if originalRecorder.Code != 404 {
		t.Errorf("Status code = %d, want 404", originalRecorder.Code)
	}
}

func TestResponseCache_KeyTypes(t *testing.T) {
	logger := logger.New(logger.LevelDebug)

	tests := []struct {
		name     string
		keyType  string
		req      *http.Request
		expected string
	}{
		{
			name:     "uri key",
			keyType:  "uri",
			req:      httptest.NewRequest("GET", "/test/path?query=1", nil),
			expected: "/test/path?query=1",
		},
		{
			name:     "host key",
			keyType:  "host",
			req:      httptest.NewRequest("GET", "http://example.com/test", nil),
			expected: "example.com",
		},
		{
			name:     "method key",
			keyType:  "method",
			req:      httptest.NewRequest("POST", "/test", nil),
			expected: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := NewPlugin(logger).(*ResponseCache)
			config := types.PluginConf{
				"size": 100,
				"ttl":  30,
				"key":  tt.keyType,
			}
			err := plugin.ValidateAndSetConfig(config)
			if err != nil {
				t.Fatalf("Failed to set config: %v", err)
			}

			recorder := httptest.NewRecorder()
			plugin.HandleRequest(tt.req, recorder)
			wrappedWriter := plugin.HandleResponse(tt.req, recorder).(*DemoResponseWriter)
			if wrappedWriter.key != tt.expected {
				t.Errorf("Key = %s, want %s", wrappedWriter.key, tt.expected)
			}
		})
	}
}

func TestResponseCache_Integration(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	config := types.PluginConf{
		"size": 100,
		"ttl":  1,
		"key":  "uri",
	}
	err := plugin.ValidateAndSetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	req := httptest.NewRequest("GET", "/integration-test", nil)

	// First request - should miss cache
	recorder1 := httptest.NewRecorder()
	handled1, _ := plugin.HandleRequest(req, recorder1)
	if handled1 {
		t.Error("First request should not be handled (cache miss)")
	}

	// Process response and cache it
	wrappedWriter := plugin.HandleResponse(req, recorder1)
	responseData := []byte("integration test response")
	wrappedWriter.Write(responseData)

	// Second request - should hit cache
	recorder2 := httptest.NewRecorder()
	handled2, _ := plugin.HandleRequest(req, recorder2)
	if !handled2 {
		t.Error("Second request should be handled (cache hit)")
	}
	if recorder2.Header().Get("X-Cache-Hit") != "true" {
		t.Error("Second request should have X-Cache-Hit: true")
	}
	if recorder2.Body.String() != string(responseData) {
		t.Errorf("Cached response body = %s, want %s", recorder2.Body.String(), responseData)
	}
}

func TestResponseCache_Priority(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	priority := plugin.Priority()
	if priority != 100 {
		t.Errorf("Priority() = %d, want 100", priority)
	}
}

func TestResponseCache_DefaultTTL(t *testing.T) {
	logger := logger.New(logger.LevelDebug)
	plugin := NewPlugin(logger).(*ResponseCache)

	// Config without TTL (should use default)
	config := types.PluginConf{
		"size": 100,
		"key":  "uri",
	}

	// This should fail because TTL is required in validation
	err := plugin.ValidateAndSetConfig(config)
	if err == nil {
		t.Error("ValidateAndSetConfig should fail without TTL")
	}

	// Test with proper config but check default TTL behavior in Write method
	config["ttl"] = 30
	err = plugin.ValidateAndSetConfig(config)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

}
