package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/utils"
)

const bufferSize = 32 * 1024

// Per ARP instance - to be used to manage state across requests like connection pool and reusable buffers
type Service struct {
	buf *utils.Pool[[]byte]
	log *logger.Logger
}

func NewService(log *logger.Logger) *Service {
	return &Service{
		buf: utils.NewPool(func() []byte {
			return make([]byte, bufferSize)
		}),
		log: log.WithComponent("proxy_service"),
	}
}

type ReverseProxy struct {
	transport http.RoundTripper
	logger    *logger.Logger
	service   *Service
}

func NewReverseProxy(logger *logger.Logger, rp *Service) *ReverseProxy {
	return &ReverseProxy{
		transport: &http.Transport{},
		logger:    logger.WithComponent("reverse_proxy"),
		service:   rp,
	}
}

func (p *ReverseProxy) copyAndFlush(dst http.ResponseWriter, src io.Reader, bufferSize int) {
	flusher, hasFlusher := dst.(http.Flusher)
	buf := make([]byte, bufferSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			// Write the data
			fmt.Println("writing ", n, " bytes")
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				break
			}

			// Flush if supported
			if hasFlusher {
				flusher.Flush()
			}
		}

		if err != nil {
			if err != io.EOF {
				p.logger.Infof("Copy error: %v", err)
			}
			break
		}
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
	if isStreamingResponse(resp) {
		//TODO: figure out proper buffer size for streaming.
		// This hardcoding if bad and temporary and added to test out that streaming works
		p.copyAndFlush(w, resp.Body, 1)
	} else {
		buf := p.service.buf.Get()
		io.CopyBuffer(w, resp.Body, buf)
		defer p.service.buf.Put(buf)
	}
}

// TODO: improve detection of streaming responses
func isStreamingResponse(resp *http.Response) bool {
	// Check for chunked transfer encoding
	if resp.TransferEncoding != nil {
		for _, enc := range resp.TransferEncoding {
			if strings.ToLower(enc) == "chunked" {
				return true
			}
		}
	}
	// Check for specific content types that typically stream
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream")
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

	//TODO: decouple response handling from request handling in custom roundtripper
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
