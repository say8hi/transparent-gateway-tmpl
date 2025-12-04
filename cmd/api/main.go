package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gateway/template/internal/config"
	"github.com/gateway/template/internal/middleware"
	"github.com/gateway/template/internal/proxy"
	"github.com/gateway/template/pkg/logger"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// initialize logger
	logCfg := &logger.Config{
		Level:         cfg.Log.Level,
		ComponentName: cfg.Log.ComponentName,
		EnableStdout:  true,
		Development:   cfg.Log.Level == "debug",
	}

	log, err := logger.NewZapLogger(logCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
		}
	}()

	log.Info("api gateway started",
		"version", "1.0.0",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"services", getServiceNames(cfg),
	)

	// create proxy factory for multiple backends
	proxyFactory, err := proxy.NewFactory(&cfg.Proxy, log)
	if err != nil {
		return fmt.Errorf("failed to create proxy factory: %w", err)
	}

	// create router with middleware
	router := buildHandler(proxyFactory, cfg, log)

	// create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// wait for interrupt signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info("received shutdown signal", "signal", sig.String())

		// graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error("failed to gracefully shutdown server", "error", err)
			if err := server.Close(); err != nil {
				return fmt.Errorf("failed to close server: %w", err)
			}
		}

		log.Info("server stopped gracefully")
	}

	return nil
}

// buildHandler creates the main HTTP handler with routing and middleware.
func buildHandler(proxyFactory *proxy.Factory, cfg *config.Config, log logger.Logger) http.Handler {
	router := chi.NewRouter()

	// global middleware (applies to all routes)
	router.Use(middleware.Logging(log))
	router.Use(middleware.CORS(&cfg.CORS))

	// health check endpoint (no authentication required)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// route requests to different backend services
	for _, serviceName := range proxyFactory.Services() {
		serviceProxy, ok := proxyFactory.Get(serviceName)
		if !ok {
			continue
		}

		if serviceName == "default" {
			// legacy single backend: route everything to default with auth
			// TODO: Replace with your corporate authentication middleware from common package:
			// router.Use(common.JWTAuthMiddleware())
			router.Group(func(r chi.Router) {
				r.Use(middleware.Auth(&cfg.JWT, log))
				r.Handle("/*", serviceProxy)
			})

			log.Info("registered route", "pattern", "/*", "service", serviceName)
		} else {
			// multi-backend: route by service prefix with auth
			// TODO: Replace with your corporate authentication middleware from common package:
			//
			// Example corporate middleware usage:
			// import "yourcompany.com/common/auth"
			// router.Route("/"+serviceName, func(r chi.Router) {
			//     r.Use(auth.NewJWTMiddleware(auth.Config{
			//         SecretKey: cfg.JWT.Secret,
			//         Issuer:    cfg.JWT.Issuer,
			//         Audience:  cfg.JWT.Audience,
			//     }))
			//     r.Handle("/*", serviceProxy)
			// })

			router.Route("/"+serviceName, func(r chi.Router) {
				// skip auth in test mode
				if os.Getenv("SKIP_AUTH") != "true" {
					r.Use(middleware.Auth(&cfg.JWT, log))
				}

				// strip service prefix before forwarding to backend
				r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					// remove service prefix from path
					req.URL.Path = chi.URLParam(req, "*")
					if req.URL.Path == "" {
						req.URL.Path = "/"
					}
					serviceProxy.ServeHTTP(w, req)
				}))
			})

			log.Info("registered route", "pattern", "/"+serviceName+"/*", "service", serviceName)
		}
	}

	return router
}

// getServiceNames extracts service names from proxy configuration.
func getServiceNames(cfg *config.Config) []string {
	services := make([]string, 0, len(cfg.Proxy.Targets))
	for name := range cfg.Proxy.Targets {
		services = append(services, name)
	}
	return services
}
