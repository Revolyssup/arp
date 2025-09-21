package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestReverseProxy_ServeHTTP(t *testing.T) {
	targetURL, _ := url.Parse("http://httpbin.org/headers")
	proxy := NewReverseProxy()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req, targetURL)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("httpbin")) {
		t.Errorf("Expected response body to contain 'httpbin', got %s", string(body))
	}
}
