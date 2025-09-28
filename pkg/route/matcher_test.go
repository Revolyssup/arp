package route

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Revolyssup/arp/pkg/config"
)

func TestPathMatcher(t *testing.T) {
	tests := []struct {
		name        string
		matchers    []config.Match
		request     *http.Request
		shouldMatch bool
	}{
		{
			name: "Path matcher - match",
			matchers: []config.Match{
				{
					Path: "/test",
				},
			},
			request:     httptest.NewRequest(http.MethodGet, "/test", nil),
			shouldMatch: true,
		},
		{
			name: "Path matcher - no match",
			matchers: []config.Match{
				{
					Path: "/test",
				},
			},
			request:     httptest.NewRequest(http.MethodGet, "/no-match", nil),
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewCompositeMatcher(tt.matchers)
			if err != nil {
				t.Fatalf("Failed to create matcher: %v", err)
			}
			if got := matcher.Match(tt.request); got != tt.shouldMatch {
				t.Errorf("Match() = %v, want %v", got, tt.shouldMatch)
			}
		})
	}
}

func TestHeaderMatcher(t *testing.T) {
	tests := []struct {
		name        string
		matchers    []config.Match
		request     *http.Request
		shouldMatch bool
	}{
		{
			name: "Header matcher - match",
			matchers: []config.Match{
				{
					Headers: map[string]string{
						"X-Test-Header": "test-value",
					},
				},
			},
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Test-Header", "test-value")
				return req
			}(),
			shouldMatch: true,
		},
		{
			name: "Header matcher - no match",
			matchers: []config.Match{
				{
					Headers: map[string]string{
						"X-Test-Header": "test-value",
					},
				},
			},
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Test-Header", "wrong-value")
				return req
			}(),
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewCompositeMatcher(tt.matchers)
			if err != nil {
				t.Fatalf("Failed to create matcher: %v", err)
			}
			if got := matcher.Match(tt.request); got != tt.shouldMatch {
				t.Errorf("Match() = %v, want %v", got, tt.shouldMatch)
			}
		})
	}
}

func TestMethodMatcher(t *testing.T) {
	tests := []struct {
		name        string
		matchers    []config.Match
		request     *http.Request
		shouldMatch bool
	}{
		{
			name: "Method matcher - match",
			matchers: []config.Match{
				{
					Method: http.MethodGet,
				},
			},
			request:     httptest.NewRequest(http.MethodGet, "/test", nil),
			shouldMatch: true,
		},
		{
			name: "Method matcher - no match",
			matchers: []config.Match{
				{
					Method: http.MethodGet,
				},
			},
			request:     httptest.NewRequest(http.MethodPost, "/test", nil),
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewCompositeMatcher(tt.matchers)
			if err != nil {
				t.Fatalf("Failed to create matcher: %v", err)
			}
			if got := matcher.Match(tt.request); got != tt.shouldMatch {
				t.Errorf("Match() = %v, want %v", got, tt.shouldMatch)
			}
		})
	}
}

// BenchmarkCompositeMatcher_LargeNumber tests performance with many matchers
func BenchmarkCompositeMatcher_LargeNumber(b *testing.B) {
	// Test different sizes to see how performance scales
	sizes := []int{10, 100, 1000, 5000}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			// Create a large number of match configurations
			matchConfigs := make([]config.Match, size)
			for i := 0; i < size; i++ {
				matchConfigs[i] = config.Match{
					Path:   "/api/v" + strconv.Itoa(i) + "/users/.*",
					Method: "GET",
					Headers: map[string]string{
						"X-API-Version": "v" + strconv.Itoa(i),
						"Content-Type":  "application/json",
					},
				}
			}

			matcher, err := NewCompositeMatcher(matchConfigs)
			if err != nil {
				b.Fatalf("Failed to create matcher: %v", err)
			}

			// Create a request that won't match any of the patterns
			req, _ := http.NewRequest("GET", "/api/unknown/path", nil)
			req.Header.Set("Content-Type", "application/json")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matcher.Match(req)
			}
		})
	}
}

// BenchmarkCompositeMatcher_MatchAtEnd tests worst-case scenario where match is at the end
func BenchmarkCompositeMatcher_MatchAtEnd(b *testing.B) {
	sizes := []int{100, 1000, 5000}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			matchConfigs := make([]config.Match, size)

			// Fill with non-matching patterns
			for i := 0; i < size-1; i++ {
				matchConfigs[i] = config.Match{
					Path:   "/non/matching/" + strconv.Itoa(i) + "/.*",
					Method: "POST", // Different method
					Headers: map[string]string{
						"X-Non-Matching": "value" + strconv.Itoa(i),
					},
				}
			}

			// Add one matching pattern at the end
			matchConfigs[size-1] = config.Match{
				Path:   "/api/v1/.*",
				Method: "GET",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}

			matcher, err := NewCompositeMatcher(matchConfigs)
			if err != nil {
				b.Fatalf("Failed to create matcher: %v", err)
			}

			req, _ := http.NewRequest("GET", "/api/v1/users/123", nil)
			req.Header.Set("Content-Type", "application/json")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matcher.Match(req)
			}
		})
	}
}

// BenchmarkCompositeMatcher_MatchAtBeginning tests best-case scenario
func BenchmarkCompositeMatcher_MatchAtBeginning(b *testing.B) {
	sizes := []int{100, 1000, 5000}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			matchConfigs := make([]config.Match, size)

			// Add matching pattern at the beginning
			matchConfigs[0] = config.Match{
				Path:   "/api/v1/.*",
				Method: "GET",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}

			// Fill rest with non-matching patterns
			for i := 1; i < size; i++ {
				matchConfigs[i] = config.Match{
					Path:   "/non/matching/" + strconv.Itoa(i) + "/.*",
					Method: "POST",
					Headers: map[string]string{
						"X-Non-Matching": "value" + strconv.Itoa(i),
					},
				}
			}

			matcher, err := NewCompositeMatcher(matchConfigs)
			if err != nil {
				b.Fatalf("Failed to create matcher: %v", err)
			}

			req, _ := http.NewRequest("GET", "/api/v1/users/123", nil)
			req.Header.Set("Content-Type", "application/json")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matcher.Match(req)
			}
		})
	}
}

// BenchmarkIndividualMatchers tests performance of individual matcher types
func BenchmarkIndividualMatchers(b *testing.B) {
	// Benchmark path matcher with complex regex
	b.Run("PathMatcher_ComplexRegex", func(b *testing.B) {
		matcher, err := newPathMatcher(`/api/v[0-9]+/users/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/.*`)
		if err != nil {
			b.Fatalf("Failed to create path matcher: %v", err)
		}

		req, _ := http.NewRequest("GET", "/api/v1/users/550e8400-e29b-41d4-a716-446655440000/profile", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matcher.Match(req)
		}
	})

	// Benchmark header matcher with many headers
	b.Run("HeaderMatcher_ManyHeaders", func(b *testing.B) {
		headers := make(map[string]string)
		for i := 0; i < 50; i++ {
			headers["X-Custom-Header-"+strconv.Itoa(i)] = "value-" + strconv.Itoa(i)
		}

		matcher := newHeaderMatcher(headers)
		req, _ := http.NewRequest("GET", "/test", nil)

		// Set only some of the headers (partial match scenario)
		for i := 0; i < 25; i++ {
			req.Header.Set("X-Custom-Header-"+strconv.Itoa(i), "value-"+strconv.Itoa(i))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matcher.Match(req)
		}
	})

	// Benchmark method matcher
	b.Run("MethodMatcher", func(b *testing.B) {
		matcher := newMethodMatcher("POST")
		req, _ := http.NewRequest("GET", "/test", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matcher.Match(req)
		}
	})
}

// BenchmarkCompositeMatcher_RealisticScenario tests a more realistic scenario
func BenchmarkCompositeMatcher_RealisticScenario(b *testing.B) {
	// Simulate a realistic API gateway configuration
	matchConfigs := []config.Match{
		{
			Path:   "^/api/v1/health$",
			Method: "GET",
		},
		{
			Path:   "^/api/v1/users/[0-9]+$",
			Method: "GET",
			Headers: map[string]string{
				"Authorization": "Bearer .*",
			},
		},
		{
			Path:   "^/api/v1/users$",
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Path:   "^/api/v2/.*",
			Method: "GET",
			Headers: map[string]string{
				"X-API-Version": "v2",
			},
		},
	}

	// Add many more generic matchers to simulate large configuration
	for i := 0; i < 1000; i++ {
		matchConfigs = append(matchConfigs, config.Match{
			Path:   "^/static/" + strconv.Itoa(i) + "/.*",
			Method: "GET",
		})
	}

	matcher, err := NewCompositeMatcher(matchConfigs)
	if err != nil {
		b.Fatalf("Failed to create matcher: %v", err)
	}

	// Test different request scenarios
	scenarios := []struct {
		name    string
		path    string
		method  string
		headers map[string]string
	}{
		{
			name:   "HealthCheck",
			path:   "/api/v1/health",
			method: "GET",
		},
		{
			name:   "UserGet",
			path:   "/api/v1/users/123",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer token123",
			},
		},
		{
			name:   "StaticContent",
			path:   "/static/500/resource",
			method: "GET",
		},
		{
			name:   "NoMatch",
			path:   "/unknown/path",
			method: "GET",
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			req, _ := http.NewRequest(scenario.method, scenario.path, nil)
			for k, v := range scenario.headers {
				req.Header.Set(k, v)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matcher.Match(req)
			}
		})
	}
}
