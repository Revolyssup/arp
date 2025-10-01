package responsecache

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Revolyssup/arp/pkg/cache"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/Revolyssup/arp/pkg/plugin/types"
)

type ResponseCache struct {
	config types.PluginConf
	cache  *cache.LRUCache[[]byte]
	logger *logger.Logger
}

type DemoResponseWriter struct {
	*types.BaseResponseWriter
	plugin *ResponseCache
	key    string
}

func NewPlugin(logger *logger.Logger) types.Plugin {
	subLogger := logger.WithComponent("ResponseCache")
	return &ResponseCache{
		config: types.PluginConf{},
		logger: subLogger,
	}
}

func (p *ResponseCache) GetConfig() types.PluginConf {
	return p.config
}

const (
	URIKEY     = "uri"
	HOSTKEY    = "host"
	METHODKEY  = "method"
	DefaultTTL = 30 * time.Second
)

func (p *ResponseCache) ValidateAndSetConfig(conf types.PluginConf) error {
	//validate before setting
	if conf == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if size, ok := conf["size"].(int); ok {
		if size <= 0 {
			return fmt.Errorf("size must be a positive integer")
		}
	} else {
		return fmt.Errorf("size must be an integer")
	}
	if ttl, ok := conf["ttl"].(int); ok {
		if ttl <= 0 {
			return fmt.Errorf("ttl must be a positive integer")
		}
	} else {
		return fmt.Errorf("ttl must be an integer")
	}
	if key, ok := conf["key"].(string); ok {
		validKeys := map[string]bool{
			URIKEY:    true,
			HOSTKEY:   true,
			METHODKEY: true,
		}
		if !validKeys[key] {
			return fmt.Errorf("key must be one of uri, host, method")
		}
	} else {
		return fmt.Errorf("key must be a string")
	}
	p.config = conf
	size := conf["size"].(int)
	p.cache = cache.NewLRUCache[[]byte](size, p.logger)
	p.logger.Debugf("Initialized ResponseCache plugin with size %d", size)
	return nil
}

func getKeyFromConf(conf types.PluginConf, req *http.Request) string {
	if key, ok := conf["key"].(string); ok {
		switch key {
		case URIKEY:
			return req.RequestURI
		case HOSTKEY:
			return req.Host
		case METHODKEY:
			return req.Method
		default:
			return req.RequestURI // default
		}
	}
	return req.RequestURI // default
}
func (p *ResponseCache) HandleRequest(req *http.Request, res http.ResponseWriter) (bool, error) {
	conf := p.GetConfig()
	key := getKeyFromConf(conf, req)

	if ans, ok := p.cache.Get(key); ok {
		res.Header().Add("X-Cache-Hit", "true")
		res.Write(ans)
		return true, nil
	}
	return false, nil
}

func (p *ResponseCache) HandleResponse(req *http.Request, w http.ResponseWriter) http.ResponseWriter {
	return &DemoResponseWriter{
		BaseResponseWriter: &types.BaseResponseWriter{ResponseWriter: w},
		plugin:             p,
		key:                getKeyFromConf(p.GetConfig(), req),
	}
}

func (w *DemoResponseWriter) WriteHeader(statusCode int) {
	w.Header().Add("X-Cache-Hit", "false")
	w.BaseResponseWriter.WriteHeader(statusCode)
}

func (w *DemoResponseWriter) Write(data []byte) (int, error) {
	conf := w.plugin.GetConfig()
	key := w.key
	//TODO: Fixme: This is problematic. When responsecache plugin is used with streaming scenarios.
	// Other than the fact that ResponseWriter here wont do streaming and write all at once which is generatl problem for now while using plugin.
	// The last chunk will be the one that gets cached and subsequent requests will get only that chunk.
	if ttl, ok := conf["ttl"].(int); ok {
		w.plugin.cache.Set(key, data, time.Duration(ttl)*time.Second)
	} else {
		w.plugin.cache.Set(key, data, DefaultTTL)
	}
	return w.BaseResponseWriter.Write(data)
}

func (p *ResponseCache) Priority() int {
	return 100
}

func (p *ResponseCache) Destroy() {
	p.cache.Reset()
}
