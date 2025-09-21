package router

import (
	"net/http"
	"net/http/httptest"
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
