package proxy

import (
	"fmt"

	"github.com/gateway/template/internal/config"
	"github.com/gateway/template/pkg/logger"
)

// Factory creates and manages multiple reverse proxies.
type Factory struct {
	proxies map[string]*ReverseProxy
	log     logger.Logger
}

// NewFactory creates a new proxy factory with multiple reverse proxies.
func NewFactory(cfg *config.ProxyConfig, log logger.Logger) (*Factory, error) {
	if len(cfg.Targets) == 0 {
		return nil, fmt.Errorf("no proxy targets configured")
	}

	proxies := make(map[string]*ReverseProxy)

	for name, targetCfg := range cfg.Targets {
		// create a single proxy config for this target
		singleCfg := &config.ProxyConfig{
			Targets: map[string]config.TargetConfig{
				name: targetCfg,
			},
			Timeout: cfg.Timeout,
		}

		// create proxy
		proxy, err := New(singleCfg, targetCfg.URL, log, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy for %q: %w", name, err)
		}

		proxies[name] = proxy
		log.Info("created proxy", "service", name, "target", targetCfg.URL)
	}

	return &Factory{
		proxies: proxies,
		log:     log,
	}, nil
}

// Get returns a proxy by service name.
func (f *Factory) Get(name string) (*ReverseProxy, bool) {
	proxy, ok := f.proxies[name]
	return proxy, ok
}

// All returns all proxies.
func (f *Factory) All() map[string]*ReverseProxy {
	return f.proxies
}

// Services returns a list of all configured service names.
func (f *Factory) Services() []string {
	services := make([]string, 0, len(f.proxies))
	for name := range f.proxies {
		services = append(services, name)
	}
	return services
}
