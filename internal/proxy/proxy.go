package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gateway/template/internal/config"
	"github.com/gateway/template/pkg/logger"
)

// ReverseProxy wraps httputil.ReverseProxy with additional functionality.
type ReverseProxy struct {
	proxy       *httputil.ReverseProxy
	target      *url.URL
	log         logger.Logger
	cfg         *config.ProxyConfig
	serviceName string
}

// New creates a new reverse proxy instance.
func New(cfg *config.ProxyConfig, targetURL string, log logger.Logger, serviceName string) (*ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	rp := &ReverseProxy{
		proxy:       proxy,
		target:      target,
		log:         log,
		cfg:         cfg,
		serviceName: serviceName,
	}

	// customize director to modify requests before proxying
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		rp.modifyRequest(req)
	}

	// customize error handler
	proxy.ErrorHandler = rp.errorHandler

	// customize response modifier
	proxy.ModifyResponse = rp.modifyResponse

	return rp, nil
}

// ServeHTTP implements http.Handler interface.
// This is called after all middleware (logging, CORS, auth) have run.
// It forwards the request to the backend service and returns the response.
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// create a context with timeout to prevent hanging requests
	// if backend doesn't respond within PROXY_TIMEOUT, returns 504
	ctx, cancel := context.WithTimeout(r.Context(), rp.cfg.Timeout)
	defer cancel()

	// update request with timeout context
	r = r.WithContext(ctx)

	rp.log.Debug("proxying request",
		"method", r.Method,
		"path", r.URL.Path,
		"target", rp.target.String(),
		"service", rp.serviceName,
	)

	// proxy.ServeHTTP does the actual work:
	// 1. Calls Director (modifyRequest) to prepare the request
	// 2. Sends request to backend (PROXY_TARGET_URL)
	// 3. Waits for backend response
	// 4. Calls ModifyResponse (currently just logs)
	// 5. Writes backend response to client
	// 6. If error occurs, calls ErrorHandler
	rp.proxy.ServeHTTP(w, r)
}

// modifyRequest modifies the request before proxying to backend.
// This is called by the Director function before sending to backend.
// The httputil.ReverseProxy already changes req.URL to point to the target,
// we just add additional headers here.
//
// SECURITY: We ALWAYS overwrite X-Forwarded headers to prevent client spoofing.
// See docs/X_FORWARDED_HEADERS.md for details.
func (rp *ReverseProxy) modifyRequest(req *http.Request) {
	// extract real client IP from connection
	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		// if SplitHostPort fails, use RemoteAddr as-is
		clientIP = req.RemoteAddr
	}

	// SECURITY: Delete any X-Forwarded headers from client request
	// to prevent spoofing. We don't trust client-provided headers.
	req.Header.Del("X-Real-IP")
	req.Header.Del("X-Forwarded-For")
	req.Header.Del("X-Forwarded-Proto")
	req.Header.Del("X-Forwarded-Host")

	// set our own trusted X-Forwarded headers based on actual connection
	req.Header.Set("X-Real-IP", clientIP)
	req.Header.Set("X-Forwarded-For", clientIP)

	// set protocol based on TLS connection state
	if req.TLS != nil {
		req.Header.Set("X-Forwarded-Proto", "https")
	} else {
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	// set original host from request
	req.Header.Set("X-Forwarded-Host", req.Host)

	// IMPORTANT: Change Host header to target host for virtual host routing
	// Backend nginx may use Host header for routing (virtual hosts)
	req.Host = req.URL.Host

	// Note: All other headers (including Authorization with JWT)
	// are preserved and forwarded to the backend unchanged
}

// modifyResponse modifies the response before returning to client.
func (rp *ReverseProxy) modifyResponse(resp *http.Response) error {
	rp.log.Debug("received response from target",
		"status", resp.StatusCode,
		"target", rp.target.String(),
		"service", rp.serviceName,
	)
	return nil
}

// errorHandler handles errors that occur during proxying.
func (rp *ReverseProxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	rp.log.Error("proxy error",
		"method", r.Method,
		"path", r.URL.Path,
		"target", rp.target.String(),
		"service", rp.serviceName,
		"error", err,
	)

	// check if context deadline exceeded
	if r.Context().Err() == context.DeadlineExceeded {
		http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
		return
	}

	http.Error(w, "bad gateway", http.StatusBadGateway)
}
