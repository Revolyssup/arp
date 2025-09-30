package demo

import (
	"bufio"
	"net"
	"net/http"

	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/plugin/types"
)

type DemoPlugin struct {
	config types.PluginConf
}

type DemoResponseWriter struct {
	*types.BaseResponseWriter
	plugin   *DemoPlugin
	hijacker http.Hijacker
}

func NewPlugin(logger *logger.Logger) types.Plugin {
	return &DemoPlugin{
		config: types.PluginConf{},
	}
}

func (p *DemoPlugin) GetConfig() types.PluginConf {
	return p.config
}

func (p *DemoPlugin) ValidateAndSetConfig(conf types.PluginConf) error {
	p.config = conf
	return nil
}

func (p *DemoPlugin) Destroy() {}

func (p *DemoPlugin) HandleRequest(req *http.Request, _ http.ResponseWriter) (bool, error) {
	req.Header.Set("X-Demo-Plugin", "RequestProcessed")
	conf := p.GetConfig()
	for k, v := range conf {
		if strVal, ok := v.(string); ok {
			req.Header.Set("X-Demo-"+k, strVal)
		}
	}
	return false, nil
}

func (p *DemoPlugin) HandleResponse(_ *http.Request, w http.ResponseWriter) http.ResponseWriter {
	hijacker, _ := w.(http.Hijacker)
	return &DemoResponseWriter{
		BaseResponseWriter: &types.BaseResponseWriter{ResponseWriter: w},
		plugin:             p,
		hijacker:           hijacker, // copy the hijacker from underlying response
	}
}

// To make sure the plugin works with websocket
func (w *DemoResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.hijacker.Hijack()
}

// Override WriteHeader to modify response headers
func (w *DemoResponseWriter) WriteHeader(statusCode int) {
	w.Header().Set("X-Demo-Plugin", "ResponseProcessed")
	w.BaseResponseWriter.WriteHeader(statusCode)
}

// Override Write to modify response body if needed
func (w *DemoResponseWriter) Write(data []byte) (int, error) {
	return w.BaseResponseWriter.Write(data)
}

func (p *DemoPlugin) Priority() int {
	return 100
}
