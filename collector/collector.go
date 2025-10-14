package collector

import (
	"fmt"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// Collector defines the interface for collecting metrics
type Collector interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(
		ch chan<- prometheus.Metric,
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	)
	// Reset any internal state between scrapes
	Reset()
}

// LibvirtCollector implements the prometheus.Collector interface
type LibvirtCollector struct {
	uri          string
	conn         *libvirt.Connect
	mutex        sync.RWMutex
	collectors   []Collector
	reconnectErr chan error
	exporterCollector *ExporterCollector
}

// NewLibvirtCollector creates a new LibvirtCollector
func NewLibvirtCollector(uri string) (*LibvirtCollector, error) {
	log.Printf("Connecting to libvirt at '%s'", uri)
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, err
	}

	alive, err := conn.IsAlive()
	if err != nil || !alive {
		return nil, fmt.Errorf("connection is not alive")
	}

	log.Println("Successfully connected to libvirt")

	collector := &LibvirtCollector{
		uri:          uri,
		conn:         conn,
		reconnectErr: make(chan error),
	}

	// Initialize individual collectors
	collector.exporterCollector = NewExporterCollector()
	collector.collectors = append(collector.collectors, collector.exporterCollector)
	collector.collectors = append(collector.collectors, NewDomainInfoCollector())
	collector.collectors = append(collector.collectors, NewCPUCollector())
	collector.collectors = append(collector.collectors, NewMemoryCollector())
	collector.collectors = append(collector.collectors, NewDiskCollector())
	collector.collectors = append(collector.collectors, NewNetworkCollector())
	collector.collectors = append(collector.collectors, NewDeviceCollector())
	collector.collectors = append(collector.collectors, NewConnectionCollector())

	return collector, nil
}

// Describe implements the prometheus.Collector interface
func (c *LibvirtCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors {
		collector.Describe(ch)
	}
}

// Collect implements the prometheus.Collector interface
func (c *LibvirtCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check connection health
	alive, err := c.conn.IsAlive()
	if err != nil || !alive {
		log.Printf("Warning: Connection to libvirt lost, reconnecting...")
		c.conn.Close()

		conn, err := libvirt.NewConnect(c.uri)
		if err != nil {
			log.Printf("Error: Failed to reconnect to libvirt: %v", err)
			return
		}
		c.conn = conn
		log.Println("Successfully reconnected to libvirt")
	}

	// Get all domains
	domains, err := c.conn.ListAllDomains(
		libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE,
	)
	if err != nil {
		log.Printf("Error: Failed to list domains: %v", err)
		return
	}
	defer func() {
		for _, domain := range domains {
			domain.Free()
		}
	}()

	// Reset all collectors to prepare for a new scrape
	for _, collector := range c.collectors {
		collector.Reset()
	}

	// Collect domain metrics
	for _, domain := range domains {
		// Use individual collectors to gather metrics
		for _, collector := range c.collectors {
			collector.Collect(ch, c.conn, &domain)
		}
	}

	// Update exporter metrics
	if c.exporterCollector != nil {
		c.exporterCollector.SetDomainsFound(len(domains))
	}
}

// Close closes the libvirt connection
func (c *LibvirtCollector) Close() {
	if c.conn != nil {
		log.Println("Closing libvirt connection...")
		c.conn.Close()
		log.Println("Libvirt connection closed")
	}
}
