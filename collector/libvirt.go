package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// Scraper is the interface for collecting metrics
type Scraper interface {
	Name() string
	Help() string
	Describe(ch chan<- *prometheus.Desc)
	Collect(ctx context.Context, conn *libvirt.Connect, ch chan<- prometheus.Metric) error
	Version() float64
}

// Exporter collects Libvirt metrics. It implements prometheus.Collector.
type Exporter struct {
	uri              string
	conn             *libvirt.Connect
	scrapers         []Scraper
	mutex            sync.RWMutex
	logger           *log.Logger
	scrapeTimeout    time.Duration
	scrapeSuccess    *prometheus.Desc
	scrapeDuration   *prometheus.Desc
	upMetric         *prometheus.Desc
}

// ExporterOpt configures Exporter
type ExporterOpt func(*Exporter)

// WithScrapeTimeout configures scrape timeout
func WithScrapeTimeout(timeout time.Duration) ExporterOpt {
	return func(e *Exporter) {
		e.scrapeTimeout = timeout
	}
}

// WithLogger configures logger
func WithLogger(logger *log.Logger) ExporterOpt {
	return func(e *Exporter) {
		e.logger = logger
	}
}

// Verify if Exporter implements prometheus.Collector
var _ prometheus.Collector = (*Exporter)(nil)

// NewExporter creates a new Libvirt exporter
func NewExporter(uri string, scrapers []Scraper, opts ...ExporterOpt) (*Exporter, error) {
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

	e := &Exporter{
		uri:           uri,
		conn:          conn,
		scrapers:      scrapers,
		logger:        log.Default(),
		scrapeTimeout: 10 * time.Second,
		upMetric: prometheus.NewDesc(
			"libvirt_up",
			"Whether scraping libvirt's metrics was successful.",
			nil, nil,
		),
		scrapeSuccess: prometheus.NewDesc(
			"libvirt_exporter_scraper_success",
			"Whether a scraper succeeded.",
			[]string{"scraper"}, nil,
		),
		scrapeDuration: prometheus.NewDesc(
			"libvirt_exporter_scraper_duration_seconds",
			"Duration of a scrape job.",
			[]string{"scraper"}, nil,
		),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.upMetric
	ch <- e.scrapeSuccess
	ch <- e.scrapeDuration
	
	// Describe all scrapers
	for _, scraper := range e.scrapers {
		scraper.Describe(ch)
	}
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	up := e.scrape(context.Background(), ch)
	ch <- prometheus.MustNewConstMetric(e.upMetric, prometheus.GaugeValue, up)
}

// scrape collects metrics from all scrapers
func (e *Exporter) scrape(ctx context.Context, ch chan<- prometheus.Metric) float64 {
	// Check connection health
	alive, err := e.conn.IsAlive()
	if err != nil || !alive {
		e.logger.Printf("Warning: Connection to libvirt lost, reconnecting...")
		e.conn.Close()

		conn, err := libvirt.NewConnect(e.uri)
		if err != nil {
			e.logger.Printf("Error: Failed to reconnect to libvirt: %v", err)
			return 0.0
		}
		e.conn = conn
		e.logger.Println("Successfully reconnected to libvirt")
	}

	// Use a wait group to wait for all scrapers to complete
	var wg sync.WaitGroup
	
	// Collect metrics from all scrapers
	for _, scraper := range e.scrapers {
		wg.Add(1)
		go func(scraper Scraper) {
			defer wg.Done()
			
			label := scraper.Name()
			scrapeTime := time.Now()
			
			scrapeContext, cancel := context.WithTimeout(ctx, e.scrapeTimeout)
			defer cancel()
			
			scrapeSuccess := 1.0
			if err := scraper.Collect(scrapeContext, e.conn, ch); err != nil {
				e.logger.Printf("Error scraping %s: %v", label, err)
				scrapeSuccess = 0.0
			}
			
			// Export scrape success and duration metrics
			ch <- prometheus.MustNewConstMetric(e.scrapeSuccess, prometheus.GaugeValue, scrapeSuccess, label)
			ch <- prometheus.MustNewConstMetric(e.scrapeDuration, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), label)
		}(scraper)
	}
	
	wg.Wait()
	return 1.0
}

// Close closes the libvirt connection
func (e *Exporter) Close() error {
	if e.conn != nil {
		e.logger.Println("Closing libvirt connection...")
		_, err := e.conn.Close()
		e.logger.Println("Libvirt connection closed")
		return err
	}
	return nil
}

// DomainInfoScraper collects basic domain information
type DomainInfoScraper struct {
	vmStatus        *prometheus.Desc
	vmCPUTime       *prometheus.Desc
	vmMemoryCurrent *prometheus.Desc
	vmMemoryMax     *prometheus.Desc
	vmUptime        *prometheus.Desc
}

// NewDomainInfoScraper creates a new DomainInfoScraper
func NewDomainInfoScraper() *DomainInfoScraper {
	return &DomainInfoScraper{
		vmStatus: prometheus.NewDesc(
			"libvirt_vm_status",
			"Status of the virtual machine (1=running, 0=other)",
			[]string{"domain", "uuid"},
			nil,
		),
		vmCPUTime: prometheus.NewDesc(
			"libvirt_vm_cpu_time_seconds_total",
			"Total CPU time used by the virtual machine in seconds",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryCurrent: prometheus.NewDesc(
			"libvirt_vm_memory_current_bytes",
			"Current memory usage of the virtual machine in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryMax: prometheus.NewDesc(
			"libvirt_vm_memory_max_bytes",
			"Maximum memory allowed for the virtual machine in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmUptime: prometheus.NewDesc(
			"libvirt_vm_uptime_seconds",
			"Virtual machine uptime in seconds",
			[]string{"domain", "uuid"},
			nil,
		),
	}
}

// Name implements Scraper
func (s *DomainInfoScraper) Name() string {
	return "domain_info"
}

// Help implements Scraper
func (s *DomainInfoScraper) Help() string {
	return "Collect domain information"
}

// Version implements Scraper
func (s *DomainInfoScraper) Version() float64 {
	return 1.0
}

// Describe implements Scraper
func (s *DomainInfoScraper) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.vmStatus
	ch <- s.vmCPUTime
	ch <- s.vmMemoryCurrent
	ch <- s.vmMemoryMax
	ch <- s.vmUptime
}

// Collect implements Scraper
func (s *DomainInfoScraper) Collect(ctx context.Context, conn *libvirt.Connect, ch chan<- prometheus.Metric) error {
	// Get all domains
	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to list domains: %w", err)
	}
	defer func() {
		for _, domain := range domains {
			domain.Free()
		}
	}()

	for _, domain := range domains {
		domainInfo, err := domain.GetInfo()
		if err != nil {
			s.logError(domain, "get domain info", err)
			continue
		}

		domainName, err := domain.GetName()
		if err != nil {
			s.logError(domain, "get domain name", err)
			continue
		}

		domainUUID, err := domain.GetUUIDString()
		if err != nil {
			s.logError(domain, "get domain UUID", err)
			continue
		}

		// VM status metric
		status := 0.0
		if domainInfo.State == libvirt.DOMAIN_RUNNING {
			status = 1.0
		}
		ch <- prometheus.MustNewConstMetric(s.vmStatus, prometheus.GaugeValue, status, domainName, domainUUID)

		// CPU time metric (convert from nanoseconds to seconds)
		ch <- prometheus.MustNewConstMetric(s.vmCPUTime, prometheus.CounterValue, float64(domainInfo.CpuTime)/1e9, domainName, domainUUID)

		// Memory metrics
		ch <- prometheus.MustNewConstMetric(s.vmMemoryCurrent, prometheus.GaugeValue, float64(domainInfo.Memory)*1024, domainName, domainUUID)
		ch <- prometheus.MustNewConstMetric(s.vmMemoryMax, prometheus.GaugeValue, float64(domainInfo.MaxMem)*1024, domainName, domainUUID)

		// Only collect uptime for running domains
		if domainInfo.State == libvirt.DOMAIN_RUNNING {
			// Collect uptime (simplified - using current time minus start time)
			domainTime, _, err := domain.GetTime(0)
			if err == nil {
				uptime := time.Since(time.Unix(int64(domainTime/1000), 0)).Seconds()
				ch <- prometheus.MustNewConstMetric(s.vmUptime, prometheus.GaugeValue, uptime, domainName, domainUUID)
			}
		}
	}

	return nil
}

func (s *DomainInfoScraper) logError(domain libvirt.Domain, operation string, err error) {
	name, _ := domain.GetName()
	uuid, _ := domain.GetUUIDString()
	log.Printf("Error %s for domain %s (%s): %v", operation, name, uuid, err)
}

// DiskScraper collects disk I/O statistics
type DiskScraper struct {
	vmDiskReadBytes  *prometheus.Desc
	vmDiskWriteBytes *prometheus.Desc
	vmDiskReadOps    *prometheus.Desc
	vmDiskWriteOps   *prometheus.Desc
}

// NewDiskScraper creates a new DiskScraper
func NewDiskScraper() *DiskScraper {
	return &DiskScraper{
		vmDiskReadBytes: prometheus.NewDesc(
			"libvirt_vm_disk_read_bytes_total",
			"Total bytes read from disk by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskWriteBytes: prometheus.NewDesc(
			"libvirt_vm_disk_write_bytes_total",
			"Total bytes written to disk by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskReadOps: prometheus.NewDesc(
			"libvirt_vm_disk_read_ops_total",
			"Total disk read operations by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskWriteOps: prometheus.NewDesc(
			"libvirt_vm_disk_write_ops_total",
			"Total disk write operations by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
	}
}

// Name implements Scraper
func (s *DiskScraper) Name() string {
	return "disk"
}

// Help implements Scraper
func (s *DiskScraper) Help() string {
	return "Collect disk I/O statistics"
}

// Version implements Scraper
func (s *DiskScraper) Version() float64 {
	return 1.0
}

// Describe implements Scraper
func (s *DiskScraper) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.vmDiskReadBytes
	ch <- s.vmDiskWriteBytes
	ch <- s.vmDiskReadOps
	ch <- s.vmDiskWriteOps
}

// Collect implements Scraper
func (s *DiskScraper) Collect(ctx context.Context, conn *libvirt.Connect, ch chan<- prometheus.Metric) error {
	// Get all domains
	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to list domains: %w", err)
	}
	defer func() {
		for _, domain := range domains {
			domain.Free()
		}
	}()

	for _, domain := range domains {
		domainInfo, err := domain.GetInfo()
		if err != nil {
			s.logError(domain, "get domain info", err)
			continue
		}

		// Only collect metrics for running domains
		if domainInfo.State != libvirt.DOMAIN_RUNNING {
			continue
		}

		domainName, err := domain.GetName()
		if err != nil {
			s.logError(domain, "get domain name", err)
			continue
		}

		domainUUID, err := domain.GetUUIDString()
		if err != nil {
			s.logError(domain, "get domain UUID", err)
			continue
		}

		// Use the approach from the original code to collect disk stats for common block devices
		blockDevices := []string{"vda", "vdb", "hda", "hdb", "sda", "sdb"}
		for _, device := range blockDevices {
			stats, err := domain.BlockStats(device)
			if err == nil {
				ch <- prometheus.MustNewConstMetric(s.vmDiskReadBytes, prometheus.CounterValue, float64(stats.RdBytes), domainName, domainUUID, device)
				ch <- prometheus.MustNewConstMetric(s.vmDiskWriteBytes, prometheus.CounterValue, float64(stats.WrBytes), domainName, domainUUID, device)
				ch <- prometheus.MustNewConstMetric(s.vmDiskReadOps, prometheus.CounterValue, float64(stats.RdReq), domainName, domainUUID, device)
				ch <- prometheus.MustNewConstMetric(s.vmDiskWriteOps, prometheus.CounterValue, float64(stats.WrReq), domainName, domainUUID, device)
			}
		}
	}

	return nil
}

func (s *DiskScraper) logError(domain libvirt.Domain, operation string, err error) {
	name, _ := domain.GetName()
	uuid, _ := domain.GetUUIDString()
	log.Printf("Error %s for domain %s (%s): %v", operation, name, uuid, err)
}

// NetworkScraper collects network I/O statistics
type NetworkScraper struct {
	vmNetworkRxBytes *prometheus.Desc
	vmNetworkTxBytes *prometheus.Desc
	vmNetworkRxPkts  *prometheus.Desc
	vmNetworkTxPkts  *prometheus.Desc
}

// NewNetworkScraper creates a new NetworkScraper
func NewNetworkScraper() *NetworkScraper {
	return &NetworkScraper{
		vmNetworkRxBytes: prometheus.NewDesc(
			"libvirt_vm_network_rx_bytes_total",
			"Total network bytes received by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkTxBytes: prometheus.NewDesc(
			"libvirt_vm_network_tx_bytes_total",
			"Total network bytes transmitted by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkRxPkts: prometheus.NewDesc(
			"libvirt_vm_network_rx_packets_total",
			"Total network packets received by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkTxPkts: prometheus.NewDesc(
			"libvirt_vm_network_tx_packets_total",
			"Total network packets transmitted by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
	}
}

// Name implements Scraper
func (s *NetworkScraper) Name() string {
	return "network"
}

// Help implements Scraper
func (s *NetworkScraper) Help() string {
	return "Collect network I/O statistics"
}

// Version implements Scraper
func (s *NetworkScraper) Version() float64 {
	return 1.0
}

// Describe implements Scraper
func (s *NetworkScraper) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.vmNetworkRxBytes
	ch <- s.vmNetworkTxBytes
	ch <- s.vmNetworkRxPkts
	ch <- s.vmNetworkTxPkts
}

// Collect implements Scraper
func (s *NetworkScraper) Collect(ctx context.Context, conn *libvirt.Connect, ch chan<- prometheus.Metric) error {
	// Get all domains
	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to list domains: %w", err)
	}
	defer func() {
		for _, domain := range domains {
			domain.Free()
		}
	}()

	for _, domain := range domains {
		domainInfo, err := domain.GetInfo()
		if err != nil {
			s.logError(domain, "get domain info", err)
			continue
		}

		// Only collect metrics for running domains
		if domainInfo.State != libvirt.DOMAIN_RUNNING {
			continue
		}

		domainName, err := domain.GetName()
		if err != nil {
			s.logError(domain, "get domain name", err)
			continue
		}

		domainUUID, err := domain.GetUUIDString()
		if err != nil {
			s.logError(domain, "get domain UUID", err)
			continue
		}

		// Use the approach from the original code to collect network stats for common interfaces
		netInterfaces := []string{"vnet0", "vnet1", "eth0", "eth1"}
		for _, iface := range netInterfaces {
			stats, err := domain.InterfaceStats(iface)
			if err == nil {
				ch <- prometheus.MustNewConstMetric(s.vmNetworkRxBytes, prometheus.CounterValue, float64(stats.RxBytes), domainName, domainUUID, iface)
				ch <- prometheus.MustNewConstMetric(s.vmNetworkTxBytes, prometheus.CounterValue, float64(stats.TxBytes), domainName, domainUUID, iface)
				ch <- prometheus.MustNewConstMetric(s.vmNetworkRxPkts, prometheus.CounterValue, float64(stats.RxPackets), domainName, domainUUID, iface)
				ch <- prometheus.MustNewConstMetric(s.vmNetworkTxPkts, prometheus.CounterValue, float64(stats.TxPackets), domainName, domainUUID, iface)
			}
		}
	}

	return nil
}

func (s *NetworkScraper) logError(domain libvirt.Domain, operation string, err error) {
	name, _ := domain.GetName()
	uuid, _ := domain.GetUUIDString()
	log.Printf("Error %s for domain %s (%s): %v", operation, name, uuid, err)
}