package main

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var version = "dev"

func main() {
	// Parse configuration
	config, err := ParseConfig()
	if err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	log.Printf("Starting UOS Libvirt Exporter %s", version)
	config.Log()

	// Create libvirt collector
	collector, err := NewLibvirtCollector(config.LibvirtURI)
	if err != nil {
		log.Fatalf("Failed to create libvirt collector: %v", err)
	}
	defer collector.Close()

	// Register collector
	prometheus.MustRegister(collector)

	// Create and setup HTTP server
	server := NewServer(config, collector)
	server.SetupHandlers()

	// Setup signal handling
	signalHandler := NewSignalHandler(collector)
	signalHandler.Start()

	log.Printf("UOS Libvirt Exporter is ready to serve requests on %s%s", config.ListenAddr, config.MetricsPath)

	// Start HTTP server
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
