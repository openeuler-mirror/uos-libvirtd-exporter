package config

import (
	"flag"
	"fmt"
	"log"
)

// Config holds the application configuration
type Config struct {
	LibvirtURI  string
	ListenAddr  string
	MetricsPath string
	ConfigFile  string
	FileConfig  *FileConfig
}

// ParseConfig parses command line flags and returns the configuration
func ParseConfig() (*Config, error) {
	var config Config

	// String parameters
	flag.StringVar(
		&config.LibvirtURI,
		"libvirt.uri",
		"",
		"Libvirt connection URI",
	)
	flag.StringVar(
		&config.ListenAddr,
		"web.listen-address",
		"",
		"Address to listen on for web interface and telemetry",
	)
	flag.StringVar(
		&config.MetricsPath,
		"web.telemetry-path",
		"",
		"Path under which to expose metrics",
	)
	flag.StringVar(
		&config.ConfigFile,
		"config.file",
		"",
		"Path to configuration file",
	)

	flag.Parse()

	// Load configuration from file if specified
	if config.ConfigFile != "" {
		fileConfig, err := LoadConfigFromFile(config.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		config.FileConfig = fileConfig
	}

	// Merge configuration (command line args take precedence over file config)
	config.mergeConfig()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// mergeConfig merges configuration from file and command line arguments
// Command line arguments take precedence over file configuration
func (c *Config) mergeConfig() {
	// If no file config, use defaults for empty command line values
	if c.FileConfig == nil {
		if c.LibvirtURI == "" {
			c.LibvirtURI = "qemu:///system"
		}
		if c.ListenAddr == "" {
			c.ListenAddr = ":9177"
		}
		if c.MetricsPath == "" {
			c.MetricsPath = "/metrics"
		}
		return
	}

	// Use file config as base, override with command line args if provided
	if c.LibvirtURI == "" {
		c.LibvirtURI = c.FileConfig.Libvirt.URI
	}
	if c.ListenAddr == "" {
		c.ListenAddr = c.FileConfig.Web.ListenAddress
	}
	if c.MetricsPath == "" {
		c.MetricsPath = c.FileConfig.Web.TelemetryPath
	}
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
	log.Println(
		"--------------------------------------------------------------------",
	)
	log.Println("UOS Libvirt Exporter Configuration:")

	if c.ConfigFile != "" {
		log.Printf("  Config File      : %s", c.ConfigFile)
		if c.FileConfig != nil {
			c.FileConfig.Log()
		}
	} else {
		log.Printf("  Libvirt URI      : %s", c.LibvirtURI)
		log.Printf("  Listen Address   : %s", c.ListenAddr)
		log.Printf("  Metrics Path     : %s", c.MetricsPath)
	}

	log.Println(
		"--------------------------------------------------------------------",
	)
}
