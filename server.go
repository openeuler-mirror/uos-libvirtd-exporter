package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server
type Server struct {
	config    *Config
	collector *LibvirtCollector
}

// NewServer creates a new HTTP server
func NewServer(config *Config, collector *LibvirtCollector) *Server {
	return &Server{
		config:    config,
		collector: collector,
	}
}

// SetupHandlers sets up the HTTP handlers
func (s *Server) SetupHandlers() {
	// Create a custom registry and register only our collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(s.collector)
	
	// Metrics endpoint using custom registry
	http.Handle(s.config.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Root endpoint
	http.HandleFunc("/", s.rootHandler)
}

// rootHandler handles the root endpoint
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`<html>
<head><title>UOS Libvirt Exporter</title></head>
<body>
<h1>UOS Libvirt Exporter</h1>
<p><a href='%s'>Metrics</a></p>
<p>Build version: %s</p>
</body>
</html>`, s.config.MetricsPath, version)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.config.ListenAddr)
	if err := http.ListenAndServe(s.config.ListenAddr, nil); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}