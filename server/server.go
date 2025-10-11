package server

import (
	"fmt"
	"log"
	"net/http"

	"gitee.com/openeuler/uos-libvirtd-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "dev"

// Server represents the HTTP server
type Server struct {
	config interface {
		GetListenAddr() string
		GetMetricsPath() string
	}
	collector *collector.LibvirtCollector
}

// Config interface for server configuration
type Config interface {
	GetListenAddr() string
	GetMetricsPath() string
}

// NewServer creates a new HTTP server
func NewServer(config Config, collector *collector.LibvirtCollector) *Server {
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
	http.Handle(s.config.GetMetricsPath(), promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

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
</html>`, s.config.GetMetricsPath(), version)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.config.GetListenAddr())
	if err := http.ListenAndServe(s.config.GetListenAddr(), nil); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}
