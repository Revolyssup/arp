package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/utils"
)

const bufferSize = 32 * 1024

// Service remains the same
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

type UpgradeHandler func(http.ResponseWriter, *http.Request, net.Conn)

type ConnPool struct {
	target *url.URL
	pool   *utils.Pool[net.Conn]
	logger *logger.Logger
}

func NewConnPool(target *url.URL, logger *logger.Logger) *ConnPool {
	return &ConnPool{
		target: target,
		pool: utils.NewPool(func() net.Conn {
			//TODO: Is it a good idea to ignore error here?
			conn, _ := net.Dial("tcp", target.Host)
			return conn
		}),
		logger: logger.WithComponent("conn_pool"),
	}
}

func (p *ConnPool) Get() (net.Conn, error) {
	conn := p.pool.Get()
	if conn == nil {
		return nil, fmt.Errorf("failed to create connection to %s", p.target.Host)
	}
	return conn, nil
}

func (p *ConnPool) Put(conn net.Conn) {
	p.pool.Put(conn)
}

type ReverseProxy struct {
	logger    *logger.Logger
	service   *Service
	connPool  *ConnPool
	targetURL *url.URL
}

func NewReverseProxy(logger *logger.Logger, service *Service, targetURL *url.URL) *ReverseProxy {
	return &ReverseProxy{
		logger:    logger.WithComponent("reverse_proxy"),
		service:   service,
		connPool:  NewConnPool(targetURL, logger),
		targetURL: targetURL,
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upstreamReq := r.Clone(r.Context())
	upstreamReq.URL.Scheme = p.targetURL.Scheme
	upstreamReq.URL.Host = p.targetURL.Host
	removeHopHeaders(upstreamReq.Header)

	var upgradeHandler UpgradeHandler
	if isWebSocketUpgrade(r) {
		upgradeHandler = p.webSocketUpgradeHandler
	}

	p.roundTrip(w, r, upstreamReq, upgradeHandler)
}

// Custom round trip implementation
func (p *ReverseProxy) roundTrip(w http.ResponseWriter, r *http.Request, upstreamReq *http.Request, upgradeHandler UpgradeHandler) {
	conn, err := p.connPool.Get()
	if err != nil {
		p.logger.Errorf("Failed to get connection from pool: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// we don't need to put back long lived connections like WebSocket for now.
	if upgradeHandler == nil {
		defer p.connPool.Put(conn)
	}

	if err := upstreamReq.Write(conn); err != nil {
		p.logger.Errorf("Failed to write request to connection: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	p.handleResponse(conn, w, r, upgradeHandler)
}

func (p *ReverseProxy) handleResponse(conn net.Conn, w http.ResponseWriter, r *http.Request, upgradeHandler UpgradeHandler) {
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, r)
	if err != nil {
		p.logger.Errorf("Failed to read response: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusSwitchingProtocols && upgradeHandler != nil {
		upgradeHandler(w, r, conn)
		return
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	if isStreamingResponse(resp) {
		p.copyAndFlush(w, resp.Body, bufferSize)
	} else {
		buf := p.service.buf.Get()
		defer p.service.buf.Put(buf)
		io.CopyBuffer(w, resp.Body, buf)
	}
}

func (p *ReverseProxy) webSocketUpgradeHandler(w http.ResponseWriter, r *http.Request, conn net.Conn) {
	defer conn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		p.logger.Error("Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		p.logger.Errorf("Failed to hijack connection: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	response := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"
	if _, err := clientConn.Write([]byte(response)); err != nil {
		p.logger.Errorf("Failed to write upgrade response: %v", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn, clientConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientConn, conn)
	}()

	wg.Wait()
}

func (p *ReverseProxy) copyAndFlush(dst http.ResponseWriter, src io.Reader, bufferSize int) {
	flusher, hasFlusher := dst.(http.Flusher)
	buf := make([]byte, bufferSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				break
			}

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

func removeHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade",
	}
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

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
	streamingTypes := []string{
		"text/event-stream",
		"application/stream+json",
	}
	for _, streamType := range streamingTypes {
		if strings.Contains(contentType, streamType) {
			return true
		}
	}
	return false
}
