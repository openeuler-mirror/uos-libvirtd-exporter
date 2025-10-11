package main

import (
	"log"

	"gitee.com/openeuler/uos-libvirtd-exporter/collector"
	"gitee.com/openeuler/uos-libvirtd-exporter/config"
	"gitee.com/openeuler/uos-libvirtd-exporter/server"
	"gitee.com/openeuler/uos-libvirtd-exporter/signal"
	"github.com/prometheus/client_golang/prometheus"
)

var version = "dev"

// configWrapper wraps the config struct to implement the server.Config interface
type configWrapper struct {
	*config.Config
}

func (c *configWrapper) GetListenAddr() string {
	return c.Config.ListenAddr
}

func (c *configWrapper) GetMetricsPath() string {
	return c.Config.MetricsPath
}

func main() {
	// Parse configuration
	cfg, err := config.ParseConfig()
	if err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	log.Printf("Starting UOS Libvirt Exporter %s", version)
	cfg.Log()

	// Create libvirt collector
	collector, err := collector.NewLibvirtCollector(cfg.LibvirtURI)
	if err != nil {
		log.Fatalf("Failed to create libvirt collector: %v", err)
	}
	defer collector.Close()

	// Register collector
	prometheus.MustRegister(collector)

	// Create and setup HTTP server
	server := server.NewServer(&configWrapper{cfg}, collector)
	server.SetupHandlers()

	// Setup signal handling
	signalHandler := signal.NewHandler(collector)
	signalHandler.Start()

	log.Printf("UOS Libvirt Exporter is ready to serve requests on %s%s", cfg.ListenAddr, cfg.MetricsPath)

	// Start HTTP server
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
