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

type UpgradeHandler func(http.ResponseWriter, *http.Request, net.Conn, *http.Response)

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
	upstreamReq.Header = r.Header.Clone()
	p.roundTrip(w, r, upstreamReq)
}

// Custom round trip implementation
func (p *ReverseProxy) roundTrip(w http.ResponseWriter, r *http.Request, upstreamReq *http.Request) {
	conn, err := p.connPool.Get()
	if err != nil {
		p.logger.Errorf("Failed to get connection from pool: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	var upgradeHandler UpgradeHandler
	if isWebSocketUpgrade(r) {
		upgradeHandler = p.webSocketUpgradeHandler
	} else {
		//TODO: fixme: removeHopHeaders unconditionally and add new for specific upgradehandler
		removeHopHeaders(upstreamReq.Header)
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
		upgradeHandler(w, r, conn, resp)
		return
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	if isStreamingResponse(resp) {
		//ideally instead of simple copy and flush. httputil.ChunkedWriter can be used.
		// But for some fun reasons, I cannot use it currently.
		// TODO: Replace this with chunked writer later.
		p.copyAndFlush(w, resp.Body, bufferSize)
	} else {
		buf := p.service.buf.Get()
		defer p.service.buf.Put(buf)
		io.CopyBuffer(w, resp.Body, buf)
	}
}

func (p *ReverseProxy) webSocketUpgradeHandler(w http.ResponseWriter, r *http.Request, conn net.Conn, resp *http.Response) {
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

	responseLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", resp.StatusCode, resp.Status)
	if _, err := clientConn.Write([]byte(responseLine)); err != nil {
		p.logger.Errorf("Failed to write response line: %v", err)
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
			if _, err := clientConn.Write([]byte(headerLine)); err != nil {
				p.logger.Errorf("Failed to write header %s: %v", key, err)
				return
			}
		}
	}

	if _, err := clientConn.Write([]byte("\r\n")); err != nil {
		p.logger.Errorf("Failed to write header terminator: %v", err)
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
