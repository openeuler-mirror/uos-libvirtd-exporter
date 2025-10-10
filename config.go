package main

import (
	"flag"
	"fmt"
	"log"
)

// Config holds the application configuration
type Config struct {
	LibvirtURI   string
	ListenAddr   string
	MetricsPath  string
}

// ParseConfig parses command line flags and returns the configuration
func ParseConfig() (*Config, error) {
	var config Config

	flag.StringVar(&config.LibvirtURI, "libvirt.uri", "qemu:///system", "Libvirt connection URI")
	flag.StringVar(&config.ListenAddr, "web.listen-address", ":9177", "Address to listen on for web interface and telemetry")
	flag.StringVar(&config.MetricsPath, "web.telemetry-path", "/metrics", "Path under which to expose metrics")

	flag.Parse()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.LibvirtURI == "" {
		return fmt.Errorf("libvirt URI cannot be empty")
	}
	if c.ListenAddr == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	if c.MetricsPath == "" {
		return fmt.Errorf("metrics path cannot be empty")
	}
	return nil
}

// Log logs the configuration values
func (c *Config) Log() {
	log.Printf("Libvirt URI: %s", c.LibvirtURI)
	log.Printf("Listening on: %s", c.ListenAddr)
	log.Printf("Metrics path: %s", c.MetricsPath)
}