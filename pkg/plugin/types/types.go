package types

import (
	"net/http"

	"github.com/Revolyssup/arp/pkg/logger"
)

// TODO: Make this a little more dynamic.
type PluginConf map[string]any

type Plugin interface {
	//If the plugin returns final, no further plugins will be executed
	//The res at this stage won't have any data written from upstream and plugin can use to early return a response
	HandleRequest(req *http.Request, res http.ResponseWriter) (final bool, err error)
	// HandleResponse should be used when the plugin needs to act on the upstream response.
	HandleResponse(*http.Request, http.ResponseWriter) http.ResponseWriter
	Priority() int
	GetConfig() PluginConf
	ValidateAndSetConfig(PluginConf) error //Returns any validation error
	Destroy()                              // garbage collection
}

type PluginFactory func(logger *logger.Logger) Plugin

type ResponseWriterWrapper interface {
	http.ResponseWriter
	Unwrap() http.ResponseWriter
}

// Base wrapper that other plugins can embed
type BaseResponseWriter struct {
	http.ResponseWriter
}

func (b *BaseResponseWriter) Unwrap() http.ResponseWriter {
	return b.ResponseWriter
}

func (b *BaseResponseWriter) Write(data []byte) (int, error) {
	return b.ResponseWriter.Write(data)
}

func (b *BaseResponseWriter) WriteHeader(statusCode int) {
	b.ResponseWriter.WriteHeader(statusCode)
}

func (b *BaseResponseWriter) Header() http.Header {
	return b.ResponseWriter.Header()
}

type Registry struct {
	plugins map[string]PluginFactory
}

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]PluginFactory),
	}
}
func (r *Registry) Register(typ string, pluginFactory PluginFactory) {
	r.plugins[typ] = pluginFactory
}

func (r *Registry) Get(typ string) (PluginFactory, bool) {
	p, exists := r.plugins[typ]
	return p, exists
}
