package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v2"
)

// FileConfig represents the configuration structure from YAML file
type FileConfig struct {
	Libvirt    LibvirtConfig    `yaml:"libvirt"`
	Web        WebConfig        `yaml:"web"`
	Logging    LoggingConfig    `yaml:"logging"`
	Collection CollectionConfig `yaml:"collection"`
	Metrics    MetricsConfig    `yaml:"metrics"`
}

// LibvirtConfig holds libvirt connection settings
type LibvirtConfig struct {
	URI               string `yaml:"uri"`
	Timeout           int    `yaml:"timeout"`
	ReconnectInterval int    `yaml:"reconnect_interval"`
}

// WebConfig holds HTTP server settings
type WebConfig struct {
	ListenAddress string `yaml:"listen_address"`
	TelemetryPath string `yaml:"telemetry_path"`
	EnablePprof   bool   `yaml:"enable_pprof"`
	PprofAddress  string `yaml:"pprof_address"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// CollectionConfig holds metrics collection settings
type CollectionConfig struct {
	Interval      int `yaml:"interval"`
	Timeout       int `yaml:"timeout"`
	MaxConcurrent int `yaml:"max_concurrent"`
}

// MetricsConfig holds metric filtering settings
type MetricsConfig struct {
	Enabled     []string          `yaml:"enabled"`
	ExtraLabels map[string]string `yaml:"extra_labels"`
}

// getDefaultConfigPaths 返回默认配置文件路径列表，按优先级排序
func getDefaultConfigPaths() []string {
	return []string{
		"./config.yaml",
		"/etc/uos-libvirtd-exporter/config.yaml",
	}
}

// LoadConfigFromFile loads configuration from YAML file
func LoadConfigFromFile(configFile string) (*FileConfig, error) {
	paths := []string{}
	
	if configFile == "" {
		// 如果没有指定配置文件，则按照默认路径搜索
		paths = getDefaultConfigPaths()
	} else {
		// 如果指定了配置文件，则只使用指定的路径
		paths = []string{configFile}
	}

	var data []byte
	var usedPath string
	var err error
	
	// 按优先级顺序尝试加载配置文件
	for _, path := range paths {
		absPath, _ := filepath.Abs(path)
		data, err = ioutil.ReadFile(absPath)
		if err == nil {
			usedPath = absPath
			break
		}
		
		// 如果是用户指定的配置文件且不存在，则返回错误
		if configFile != "" {
			return nil, fmt.Errorf("failed to read config file %s: %w", absPath, err)
		}
		
		// 如果是默认路径且文件不存在，继续尝试下一个路径
		if os.IsNotExist(err) {
			continue
		}
		
		// 其他错误，返回
		return nil, fmt.Errorf("failed to read config file %s: %w", absPath, err)
	}
	
	// 如果所有路径都尝试过了还是没有找到配置文件
	if err != nil {
		return nil, fmt.Errorf("failed to find config file in any of the default locations: %v", paths)
	}

	// Parse YAML
	var config FileConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Apply defaults if not specified
	config.applyDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	log.Printf("Configuration loaded from file: %s", usedPath)
	return &config, nil
}

// applyDefaults sets default values for missing configuration
func (c *FileConfig) applyDefaults() {
	// Libvirt defaults
	if c.Libvirt.URI == "" {
		c.Libvirt.URI = "qemu:///system"
	}
	if c.Libvirt.Timeout == 0 {
		c.Libvirt.Timeout = 30
	}
	if c.Libvirt.ReconnectInterval == 0 {
		c.Libvirt.ReconnectInterval = 10
	}

	// Web defaults
	if c.Web.ListenAddress == "" {
		c.Web.ListenAddress = ":9177"
	}
	if c.Web.TelemetryPath == "" {
		c.Web.TelemetryPath = "/metrics"
	}
	if c.Web.PprofAddress == "" {
		c.Web.PprofAddress = ":6060"
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}

	// Collection defaults
	if c.Collection.Interval == 0 {
		c.Collection.Interval = 15
	}
	if c.Collection.Timeout == 0 {
		c.Collection.Timeout = 10
	}
	if c.Collection.MaxConcurrent == 0 {
		c.Collection.MaxConcurrent = 10
	}

	// Metrics defaults
	if len(c.Metrics.Enabled) == 0 {
		c.Metrics.Enabled = []string{
			"vm_status",
			"vm_cpu",
			"vm_memory",
			"vm_disk",
			"vm_network",
			"vm_uptime",
		}
	}
	if c.Metrics.ExtraLabels == nil {
		c.Metrics.ExtraLabels = make(map[string]string)
	}
}

// Validate validates the file configuration
func (c *FileConfig) Validate() error {
	if c.Libvirt.URI == "" {
		return fmt.Errorf("libvirt URI cannot be empty")
	}
	if c.Web.ListenAddress == "" {
		return fmt.Errorf("web listen address cannot be empty")
	}
	if c.Web.TelemetryPath == "" {
		return fmt.Errorf("web telemetry path cannot be empty")
	}
	if c.Collection.Interval <= 0 {
		return fmt.Errorf("collection interval must be positive")
	}
	if c.Collection.Timeout <= 0 {
		return fmt.Errorf("collection timeout must be positive")
	}
	if c.Collection.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent must be positive")
	}
	return nil
}

// Log logs the file configuration
func (c *FileConfig) Log() {
	log.Println("Configuration from file:")
	log.Printf("  Libvirt:")
	log.Printf("    URI:              %s", c.Libvirt.URI)
	log.Printf("    Timeout:          %d", c.Libvirt.Timeout)
	log.Printf("    Reconnect Interval: %d", c.Libvirt.ReconnectInterval)
	log.Printf("  Web:")
	log.Printf("    Listen Address:   %s", c.Web.ListenAddress)
	log.Printf("    Telemetry Path:   %s", c.Web.TelemetryPath)
	log.Printf("    Enable Pprof:     %t", c.Web.EnablePprof)
	log.Printf("    Pprof Address:    %s", c.Web.PprofAddress)
	log.Printf("  Logging:")
	log.Printf("    Level:            %s", c.Logging.Level)
	log.Printf("    Format:           %s", c.Logging.Format)
	log.Printf("  Collection:")
	log.Printf("    Interval:         %d", c.Collection.Interval)
	log.Printf("    Timeout:          %d", c.Collection.Timeout)
	log.Printf("    Max Concurrent:   %d", c.Collection.MaxConcurrent)
	log.Printf("  Metrics:")
	log.Printf("    Enabled:          %v", c.Metrics.Enabled)
	log.Printf("    Extra Labels:     %v", c.Metrics.ExtraLabels)
}