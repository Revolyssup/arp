package proxy

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Revolyssup/arp/pkg/logger"
)

type ReverseProxy struct {
	transport http.RoundTripper
	logger    *logger.Logger
}

func NewReverseProxy(logger *logger.Logger) *ReverseProxy {
	return &ReverseProxy{
		transport: &http.Transport{},
		logger:    logger.WithComponent("reverse_proxy"),
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, target *url.URL) {
	if isWebSocketUpgrade(r) {
		p.serveWebSocket(w, r, target)
		return
	}

	upstreamReq := r.Clone(r.Context())
	upstreamReq.URL.Scheme = target.Scheme
	upstreamReq.URL.Host = target.Host
	removeHopHeaders(upstreamReq.Header)
	resp, err := p.transport.RoundTrip(upstreamReq)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func (p *ReverseProxy) serveWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Prepare the upstream WebSocket request
	upstreamReq := r.Clone(r.Context())
	upstreamReq.URL.Scheme = target.Scheme
	upstreamReq.URL.Host = target.Host

	upstreamConn, err := net.Dial("tcp", target.Host)
	if err != nil {
		http.Error(w, "Cannot connect to upstream", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	if err := upstreamReq.Write(upstreamConn); err != nil {
		http.Error(w, "Error writing to upstream", http.StatusBadGateway)
		return
	}

	go io.Copy(upstreamConn, clientConn)
	io.Copy(clientConn, upstreamConn)
}

func removeHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
	}
	for _, h := range hopHeaders {
		header.Del(h)
	}
}
