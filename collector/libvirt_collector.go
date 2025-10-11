package collector

import (
	"time"

	"libvirt.org/go/libvirt"
)

// LibvirtMetricsCollector implements MetricsCollector to fetch raw metrics from libvirt
type LibvirtMetricsCollector struct{}

// NewLibvirtMetricsCollector creates a new LibvirtMetricsCollector
func NewLibvirtMetricsCollector() *LibvirtMetricsCollector {
	return &LibvirtMetricsCollector{}
}

// CollectDomainInfo collects basic domain information from libvirt
func (mc *LibvirtMetricsCollector) CollectDomainInfo(conn *libvirt.Connect, domain *libvirt.Domain) (*DomainInfoMetrics, error) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	metrics := &DomainInfoMetrics{
		Name:          domainName,
		UUID:          domainUUID,
		MemoryCurrent: float64(domainInfo.Memory) * 1024,
		MemoryMax:     float64(domainInfo.MaxMem) * 1024,
		CPUTime:       float64(domainInfo.CpuTime) / 1e9,
	}

	// VM status metric
	if domainInfo.State == libvirt.DOMAIN_RUNNING {
		metrics.Status = 1.0
	} else {
		metrics.Status = 0.0
	}

	// Only collect uptime for running domains
	if domainInfo.State == libvirt.DOMAIN_RUNNING {
		domainTime, _, err := domain.GetTime(0)
		if err == nil {
			metrics.Uptime = time.Since(time.Unix(int64(domainTime/1000), 0)).Seconds()
			metrics.HasUptime = true
		}
	}

	return metrics, nil
}

// CollectDiskStats collects disk I/O statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectDiskStats(conn *libvirt.Connect, domain *libvirt.Domain) ([]DiskMetrics, error) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	// Only collect metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		return []DiskMetrics{}, nil
	}

	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	var metrics []DiskMetrics

	// Use the approach from the original code to collect disk stats for common block devices
	blockDevices := []string{"vda", "vdb", "hda", "hdb", "sda", "sdb"}
	for _, device := range blockDevices {
		stats, err := domain.BlockStats(device)
		if err == nil {
			m := DiskMetrics{
				Name:       domainName,
				UUID:       domainUUID,
				Device:     device,
				ReadBytes:  uint64(stats.RdBytes),
				WriteBytes: uint64(stats.WrBytes),
				ReadOps:    uint64(stats.RdReq),
				WriteOps:   uint64(stats.WrReq),
			}
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

// CollectNetworkStats collects network I/O statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectNetworkStats(conn *libvirt.Connect, domain *libvirt.Domain) ([]NetworkMetrics, error) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	// Only collect metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		return []NetworkMetrics{}, nil
	}

	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	var metrics []NetworkMetrics

	// Use the approach from the original code to collect network stats for common interfaces
	netInterfaces := []string{"vnet0", "vnet1", "eth0", "eth1"}
	for _, iface := range netInterfaces {
		stats, err := domain.InterfaceStats(iface)
		if err == nil {
			m := NetworkMetrics{
				Name:      domainName,
				UUID:      domainUUID,
				Interface: iface,
				RxBytes:   uint64(stats.RxBytes),
				TxBytes:   uint64(stats.TxBytes),
				RxPackets: uint64(stats.RxPackets),
				TxPackets: uint64(stats.TxPackets),
			}
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}
