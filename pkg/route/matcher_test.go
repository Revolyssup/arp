package route

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/plugin"
	"github.com/Revolyssup/arp/pkg/upstream"
)

// Helper function to create test routes
func createTestRoute() *Route {
	return &Route{
		Plugins:  plugin.NewChain(),
		Upstream: &upstream.Upstream{},
	}
}

func TestPathMatcher(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		testPath    string
		shouldMatch bool
	}{
		{
			name:        "Static path - match",
			path:        "/test",
			testPath:    "/test",
			shouldMatch: true,
		},
		{
			name:        "Static path - no match",
			path:        "/test",
			testPath:    "/no-match",
			shouldMatch: false,
		},
		{
			name:        "Prefix path - match",
			path:        "/api/*",
			testPath:    "/api/v1/users",
			shouldMatch: true,
		},
		{
			name:        "Prefix path - no match",
			path:        "/api/*",
			testPath:    "/other/v1/users",
			shouldMatch: false,
		},
		{
			name:        "Regex path - match",
			path:        "/users/[0-9]+",
			testPath:    "/users/123",
			shouldMatch: true,
		},
		{
			name:        "Regex path - no match",
			path:        "/users/[0-9]+",
			testPath:    "/users/abc",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewPathMatcher(logger.New(logger.LevelError))
			route := createTestRoute()
			matcher.Add(tt.path, route)

			matches := matcher.Match(tt.testPath)
			matched := len(matches) > 0

			if matched != tt.shouldMatch {
				t.Errorf("Match() = %v, want %v", matched, tt.shouldMatch)
			}
		})
	}
}

func TestPathMatcher_MultipleRoutes(t *testing.T) {
	matcher := NewPathMatcher(logger.New(logger.LevelError))

	route1 := createTestRoute()
	route2 := createTestRoute()
	route3 := createTestRoute()

	matcher.Add("/api/v1/*", route1)
	matcher.Add("/api/v2/users", route2)
	matcher.Add("/api/v[0-9]+/.*", route3)

	tests := []struct {
		name          string
		path          string
		expectedCount int
	}{
		{
			name:          "Multiple prefix matches",
			path:          "/api/v1/users/123",
			expectedCount: 2, // route1 (prefix) and route3 (regex)
		},
		{
			name:          "Exact static match",
			path:          "/api/v2/users",
			expectedCount: 2,
		},
		{
			name:          "No matches",
			path:          "/other/path",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matcher.Match(tt.path)
			if len(matches) != tt.expectedCount {
				t.Errorf("Match() returned %d routes, want %d", len(matches), tt.expectedCount)
			}
		})
	}
}

func TestHeaderMatcher(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		reqHeaders  map[string]string
		shouldMatch bool
	}{
		{
			name: "Single header - match",
			headers: map[string]string{
				"X-Test-Header": "test-value",
			},
			reqHeaders: map[string]string{
				"X-Test-Header": "test-value",
			},
			shouldMatch: true,
		},
		{
			name: "Single header - no match",
			headers: map[string]string{
				"X-Test-Header": "test-value",
			},
			reqHeaders: map[string]string{
				"X-Test-Header": "wrong-value",
			},
			shouldMatch: false,
		},
		{
			name: "Multiple headers - all match",
			headers: map[string]string{
				"X-Test-Header": "test-value",
				"Content-Type":  "application/json",
			},
			reqHeaders: map[string]string{
				"X-Test-Header": "test-value",
				"Content-Type":  "application/json",
			},
			shouldMatch: true,
		},
		{
			name: "Multiple headers - partial match",
			headers: map[string]string{
				"X-Test-Header": "test-value",
				"Content-Type":  "application/json",
			},
			reqHeaders: map[string]string{
				"X-Test-Header": "test-value",
				"Content-Type":  "text/plain",
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewHeaderMatcher()
			route := createTestRoute()
			matcher.Add(tt.headers, route)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.reqHeaders {
				req.Header.Set(k, v)
			}

			candidateRoutes := []*Route{route}
			matches := matcher.Match(req.Header, candidateRoutes)
			matched := len(matches) > 0

			if matched != tt.shouldMatch {
				t.Errorf("Match() = %v, want %v", matched, tt.shouldMatch)
			}
		})
	}
}

func TestMethodMatcher(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		testMethod  string
		shouldMatch bool
	}{
		{
			name:        "GET method - match",
			method:      "GET",
			testMethod:  "GET",
			shouldMatch: true,
		},
		{
			name:        "GET method - no match",
			method:      "GET",
			testMethod:  "POST",
			shouldMatch: false,
		},
		{
			name:        "POST method - match",
			method:      "POST",
			testMethod:  "POST",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewMethodMatcher()
			route := createTestRoute()
			matcher.Add(tt.method, route)

			matches := matcher.Match(tt.testMethod)
			matched := len(matches) > 0

			if matched != tt.shouldMatch {
				t.Errorf("Match() = %v, want %v", matched, tt.shouldMatch)
			}
		})
	}
}

func TestMethodMatcher_MultipleRoutes(t *testing.T) {
	matcher := NewMethodMatcher()

	route1 := createTestRoute()
	route2 := createTestRoute()
	route3 := createTestRoute()

	matcher.Add("GET", route1)
	matcher.Add("POST", route2)
	matcher.Add("GET", route3) // Another GET route

	tests := []struct {
		name          string
		method        string
		expectedCount int
	}{
		{
			name:          "GET method - multiple routes",
			method:        "GET",
			expectedCount: 2,
		},
		{
			name:          "POST method - single route",
			method:        "POST",
			expectedCount: 1,
		},
		{
			name:          "PUT method - no routes",
			method:        "PUT",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matcher.Match(tt.method)
			if len(matches) != tt.expectedCount {
				t.Errorf("Match() returned %d routes, want %d", len(matches), tt.expectedCount)
			}
		})
	}
}

func TestRouteIntersection(t *testing.T) {
	route1 := createTestRoute()
	route2 := createTestRoute()
	route3 := createTestRoute()

	tests := []struct {
		name     string
		r1       []*Route
		r2       []*Route
		expected []*Route
	}{
		{
			name:     "Empty slices",
			r1:       []*Route{},
			r2:       []*Route{},
			expected: []*Route{},
		},
		{
			name:     "No intersection",
			r1:       []*Route{route1, route2},
			r2:       []*Route{route3},
			expected: []*Route{},
		},
		{
			name:     "Partial intersection",
			r1:       []*Route{route1, route2},
			r2:       []*Route{route2, route3},
			expected: []*Route{route2},
		},
		{
			name:     "Full intersection",
			r1:       []*Route{route1, route2},
			r2:       []*Route{route1, route2},
			expected: []*Route{route1, route2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IntersectRoutes(tt.r1, tt.r2)

			if len(result) != len(tt.expected) {
				t.Errorf("intersectRoutes() returned %d routes, want %d", len(result), len(tt.expected))
				return
			}

			// Check that all expected routes are present
			for _, expectedRoute := range tt.expected {
				found := false
				for _, actualRoute := range result {
					if actualRoute == expectedRoute {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected route %v not found in result", expectedRoute)
				}
			}
		})
	}
}

// Benchmark tests for the new matchers
func BenchmarkPathMatcher(b *testing.B) {
	sizes := []int{10, 100, 1000, 5000}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			matcher := NewPathMatcher(logger.New(logger.LevelError))

			// Add routes with various path patterns
			for i := 0; i < size; i++ {
				route := createTestRoute()
				if i%3 == 0 {
					// Static paths
					matcher.Add("/api/v"+strconv.Itoa(i)+"/users", route)
				} else if i%3 == 1 {
					// Prefix paths
					matcher.Add("/static/"+strconv.Itoa(i)+"/*", route)
				} else {
					// Regex paths
					matcher.Add("/regex/v[0-9]+/item/"+strconv.Itoa(i), route)
				}
			}

			testPaths := []string{
				"/api/v5/users",       // Static match
				"/static/42/resource", // Prefix match
				"/regex/v1/item/99",   // Regex match
				"/unknown/path",       // No match
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				path := testPaths[i%len(testPaths)]
				matcher.Match(path)
			}
		})
	}
}

func BenchmarkMethodMatcher(b *testing.B) {
	sizes := []int{10, 100, 1000, 5000}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			matcher := NewMethodMatcher()

			// Add routes with various methods
			methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
			for i := 0; i < size; i++ {
				route := createTestRoute()
				method := methods[i%len(methods)]
				matcher.Add(method, route)
			}

			testMethods := []string{"GET", "POST", "PUT", "OPTIONS"}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				method := testMethods[i%len(testMethods)]
				matcher.Match(method)
			}
		})
	}
}

func BenchmarkHeaderMatcher(b *testing.B) {
	sizes := []int{10, 100, 500}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			matcher := NewHeaderMatcher()

			// Add routes with various header requirements
			for i := 0; i < size; i++ {
				route := createTestRoute()
				headers := map[string]string{
					"X-API-Key":     "key-" + strconv.Itoa(i),
					"Content-Type":  "application/json",
					"Authorization": "Bearer token-" + strconv.Itoa(i),
				}
				matcher.Add(headers, route)
			}

			// Create candidate routes (simulating already filtered routes from path/method matching)
			candidateRoutes := make([]*Route, size/10) // 10% of total routes
			for i := 0; i < len(candidateRoutes); i++ {
				candidateRoutes[i] = createTestRoute()
			}

			// Test headers
			reqHeaders := http.Header{}
			reqHeaders.Set("X-API-Key", "key-5")
			reqHeaders.Set("Content-Type", "application/json")
			reqHeaders.Set("Authorization", "Bearer token-5")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matcher.Match(reqHeaders, candidateRoutes)
			}
		})
	}
}

func BenchmarkRouteIntersection(b *testing.B) {
	sizes := []int{10, 100, 1000, 5000}

	for _, size := range sizes {
		b.Run("Size_"+strconv.Itoa(size), func(b *testing.B) {
			// Create two slices with some overlap
			allRoutes := make([]*Route, size*2)
			for i := 0; i < size*2; i++ {
				allRoutes[i] = createTestRoute()
			}

			r1 := allRoutes[:size]
			r2 := allRoutes[size/2 : size+size/2] // 50% overlap

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				IntersectRoutes(r1, r2)
			}
		})
	}
}

// Benchmark for the complete matching flow
func BenchmarkCompleteMatchingFlow(b *testing.B) {
	pathMatcher := NewPathMatcher(logger.New(logger.LevelError))
	methodMatcher := NewMethodMatcher()
	headerMatcher := NewHeaderMatcher()

	// Setup a realistic scenario with 1000 routes
	for i := 0; i < 1000; i++ {
		route := createTestRoute()

		// Add to path matcher
		if i%4 == 0 {
			pathMatcher.Add("/api/v1/*", route)
		} else if i%4 == 1 {
			pathMatcher.Add("/api/v2/users", route)
		} else if i%4 == 2 {
			pathMatcher.Add("/static/"+strconv.Itoa(i)+"/*", route)
		} else {
			pathMatcher.Add("/api/v[0-9]+/.*", route)
		}

		// Add to method matcher
		if i%3 == 0 {
			methodMatcher.Add("GET", route)
		} else if i%3 == 1 {
			methodMatcher.Add("POST", route)
		} else {
			methodMatcher.Add("PUT", route)
		}

		// Add to header matcher for some routes
		if i%5 == 0 {
			headers := map[string]string{
				"Authorization": "Bearer .*",
				"Content-Type":  "application/json",
			}
			headerMatcher.Add(headers, route)
		}
	}

	testRequests := []struct {
		name    string
		path    string
		method  string
		headers map[string]string
	}{
		{
			name:   "API v1 request",
			path:   "/api/v1/users/123",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"Content-Type":  "application/json",
			},
		},
		{
			name:   "API v2 request",
			path:   "/api/v2/users",
			method: "POST",
		},
		{
			name:   "Static content",
			path:   "/static/500/resource",
			method: "GET",
		},
		{
			name:   "No match path",
			path:   "/unknown/path",
			method: "GET",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		test := testRequests[i%len(testRequests)]

		// Simulate the complete matching flow
		pathRoutes := pathMatcher.Match(test.path)
		if len(pathRoutes) == 0 {
			continue
		}

		methodRoutes := methodMatcher.Match(test.method)
		if len(methodRoutes) == 0 {
			continue
		}

		candidateRoutes := IntersectRoutes(pathRoutes, methodRoutes)
		if len(candidateRoutes) == 0 {
			continue
		}

		// Prepare headers
		header := http.Header{}
		for k, v := range test.headers {
			header.Set(k, v)
		}

		headerMatcher.Match(header, candidateRoutes)
	}
}
