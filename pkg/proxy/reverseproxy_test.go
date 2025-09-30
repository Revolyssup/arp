package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Revolyssup/arp/pkg/logger"
)

const UpstreamAddr = "127.0.0.1:9090"

func TestReverseProxy_ServeHTTP(t *testing.T) {
	targetURL, _ := url.Parse(fmt.Sprintf("http://%s", UpstreamAddr))
	proxy := NewReverseProxy(logger.New(logger.LevelDebug))
	req := httptest.NewRequest(http.MethodGet, "http://example.com/headers", nil)
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
