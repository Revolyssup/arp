package proxy

import (
	"io"
	"net/http"
	"net/url"
)

type ReverseProxy struct {
	transport http.RoundTripper
}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{
		transport: &http.Transport{},
	}
}

// Simplest Reverse proxy for now.
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, target *url.URL) {
	upstreamReq := r.Clone(r.Context())
	upstreamReq.URL.Scheme = target.Scheme
	upstreamReq.URL.Host = target.Host
	upstreamReq.URL.Path = target.Path
	upstreamReq.Host = target.Host

	upstreamReq.Header.Del("Connection") //TODO: Delete other hop-by-hop headers

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
