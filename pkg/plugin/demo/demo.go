package demo

import (
	"net/http"

	"github.com/Revolyssup/arp/pkg/plugin/types"
)

type DemoPlugin struct {
	config types.PluginConf
}

type DemoResponseWriter struct {
	*types.BaseResponseWriter
	plugin *DemoPlugin
}

func NewPlugin() types.Plugin {
	return &DemoPlugin{
		config: types.PluginConf{},
	}
}

func (p *DemoPlugin) GetConfig() types.PluginConf {
	return p.config
}

func (p *DemoPlugin) SetConfig(conf types.PluginConf) {
	p.config = conf
}

func (p *DemoPlugin) HandleRequest(req *http.Request) error {
	req.Header.Set("X-Demo-Plugin", "RequestProcessed")
	conf := p.GetConfig()
	for k, v := range conf {
		if strVal, ok := v.(string); ok {
			req.Header.Set("X-Demo-"+k, strVal)
		}
	}
	return nil
}

func (p *DemoPlugin) WrapResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	return &DemoResponseWriter{
		BaseResponseWriter: &types.BaseResponseWriter{ResponseWriter: w},
		plugin:             p,
	}
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
