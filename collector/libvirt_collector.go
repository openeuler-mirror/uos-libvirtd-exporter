package collector

import (
	"encoding/xml"
	"log"
	"time"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

// LibvirtMetricsCollector implements MetricsCollector to fetch raw metrics from libvirt
type LibvirtMetricsCollector struct{}

// NewLibvirtMetricsCollector creates a new LibvirtMetricsCollector
func NewLibvirtMetricsCollector() *LibvirtMetricsCollector {
	return &LibvirtMetricsCollector{}
}

// CollectDomainInfo collects basic domain information from libvirt
func (mc *LibvirtMetricsCollector) CollectDomainInfo(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) (*DomainInfoMetrics, error) {
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

	// Check if domain is persistent
	persistent, err := domain.IsPersistent()
	if err != nil {
		return nil, err
	}

	// Check if domain has managed save
	managedSave, err := domain.HasManagedSaveImage(0)
	if err != nil {
		return nil, err
	}

	// Check if domain is set to autostart
	autostart, err := domain.GetAutostart()
	if err != nil {
		return nil, err
	}

	metrics := &DomainInfoMetrics{
		Name:          domainName,
		UUID:          domainUUID,
		MemoryCurrent: float64(domainInfo.Memory) * 1024,
		MemoryMax:     float64(domainInfo.MaxMem) * 1024,
		CPUTime:       float64(domainInfo.CpuTime) / 1e9,
		Autostart:     autostart,
		Persistent:    persistent,
		ManagedSave:   managedSave,
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
			metrics.BootTime = time.Unix(int64(domainTime/1000), 0)
			metrics.Uptime = time.Since(metrics.BootTime).Seconds()
			metrics.HasUptime = true
		}
	}

	return metrics, nil
}

// CollectCPUStats collects CPU statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectCPUStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) (*CPUStatsMetrics, error) {
	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	// Get vCPU counts
	maxVCPUs, err := domain.GetMaxVcpus()
	if err != nil {
		return nil, err
	}

	// Get current vCPU count
	vcpuInfo, err := domain.GetVcpus()
	if err != nil {
		// If we can't get vcpu info, use a default
		vcpuInfo = make([]libvirt.DomainVcpuInfo, 0)
	}

	metrics := &CPUStatsMetrics{
		Name:         domainName,
		UUID:         domainUUID,
		VCPUsMax:     uint(maxVCPUs),
		VCPUsCurrent: uint(len(vcpuInfo)),
		CPUTime:      domainInfo.CpuTime,
	}

	return metrics, nil
}

// CollectMemoryStats collects memory statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectMemoryStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) (*MemoryStatsMetrics, error) {
	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	// Get memory stats
	memStats, err := domain.MemoryStats(uint32(libvirt.DOMAIN_MEMORY_STAT_NR), 0)
	if err != nil {
		return nil, err
	}

	metrics := &MemoryStatsMetrics{
		Name: domainName,
		UUID: domainUUID,
	}

	// Parse memory stats
	for _, stat := range memStats {
		switch stat.Tag {
		case int32(libvirt.DOMAIN_MEMORY_STAT_ACTUAL_BALLOON):
			metrics.BalloonSize = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_UNUSED):
			metrics.Unused = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_AVAILABLE):
			metrics.Available = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_RSS):
			metrics.RSS = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_SWAP_IN):
			metrics.SwapIn = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_SWAP_OUT):
			metrics.SwapOut = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_MAJOR_FAULT):
			metrics.MajorFaults = stat.Val
		case int32(libvirt.DOMAIN_MEMORY_STAT_MINOR_FAULT):
			metrics.MinorFaults = stat.Val
		}
	}

	// Calculate total memory
	if metrics.Available > 0 && metrics.Unused > 0 {
		metrics.Total = metrics.Available
	}

	return metrics, nil
}

// CollectDiskStats collects disk I/O statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectDiskStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) ([]DiskMetrics, error) {
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

	// Try to discover devices dynamically
	devices := mc.discoverBlockDevices(domain)

	for _, device := range devices {
		// Get detailed block stats
		stats, err := domain.BlockStatsFlags(device, 0)
		if err != nil {
			// If we can't get extended stats, try basic stats
			basicStats, err := domain.BlockStats(device)
			if err != nil {
				continue
			}

			m := DiskMetrics{
				Name:       domainName,
				UUID:       domainUUID,
				Device:     device,
				Path:       "/dev/" + device,
				ReadBytes:  uint64(basicStats.RdBytes),
				WriteBytes: uint64(basicStats.WrBytes),
				ReadOps:    uint64(basicStats.RdReq),
				WriteOps:   uint64(basicStats.WrReq),
			}
			metrics = append(metrics, m)
		} else {
			m := DiskMetrics{
				Name:        domainName,
				UUID:        domainUUID,
				Device:      device,
				Path:        "/dev/" + device,
				ReadBytes:   uint64(stats.RdBytes),
				WriteBytes:  uint64(stats.WrBytes),
				ReadOps:     uint64(stats.RdReq),
				WriteOps:    uint64(stats.WrReq),
				ReadTimeNs:  uint64(stats.RdTotalTimes),
				WriteTimeNs: uint64(stats.WrTotalTimes),
			}
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

// CollectNetworkStats collects network I/O statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectNetworkStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) ([]NetworkMetrics, error) {
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

	// Try to discover interfaces dynamically
	interfaces := mc.discoverNetworkInterfaces(domain)

	for _, ifaceName := range interfaces {
		// Get interface stats
		stats, err := domain.InterfaceStats(ifaceName)
		if err != nil {
			continue
		}

		m := NetworkMetrics{
			Name:      domainName,
			UUID:      domainUUID,
			Interface: ifaceName,
			RxBytes:   uint64(stats.RxBytes),
			TxBytes:   uint64(stats.TxBytes),
			RxPackets: uint64(stats.RxPackets),
			TxPackets: uint64(stats.TxPackets),
			RxErrors:  uint64(stats.RxErrs),
			TxErrors:  uint64(stats.TxErrs),
			RxDrops:   uint64(stats.RxDrop),
			TxDrops:   uint64(stats.TxDrop),
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// discoverBlockDevices attempts to discover available block devices for a domain using XML parsing
func (mc *LibvirtMetricsCollector) discoverBlockDevices(domain *libvirt.Domain) []string {
	var devices []string

	// Get domain XML description
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		log.Printf("Warning: Failed to get domain XML: %v", err)
		return mc.fallbackBlockDeviceDiscovery(domain)
	}

	// Parse the XML
	var domainXML libvirtxml.Domain
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		log.Printf("Warning: Failed to parse domain XML: %v", err)
		return mc.fallbackBlockDeviceDiscovery(domain)
	}

	// Extract disk devices from XML
	if domainXML.Devices != nil {
		for _, disk := range domainXML.Devices.Disks {
			if disk.Target != nil && disk.Target.Dev != "" {
				devices = append(devices, disk.Target.Dev)
			}
		}
	}

	// If XML parsing didn't find any devices, fall back to trial-and-error
	if len(devices) == 0 {
		return mc.fallbackBlockDeviceDiscovery(domain)
	}

	return devices
}

// fallbackBlockDeviceDiscovery uses trial-and-error method as fallback
func (mc *LibvirtMetricsCollector) fallbackBlockDeviceDiscovery(domain *libvirt.Domain) []string {
	var devices []string

	// Common block device patterns in virtualized environments
	commonDevices := []string{
		// VirtIO block devices (KVM/QEMU)
		"vda", "vdb", "vdc", "vdd", "vde", "vdf",
		// SCSI devices
		"sda", "sdb", "sdc", "sdd", "sde", "sdf",
		// IDE devices
		"hda", "hdb", "hdc", "hdd",
		// NVMe devices
		"nvme0n1", "nvme1n1", "nvme2n1",
		// Xen devices
		"xvda", "xvdb", "xvdc", "xvdd",
	}

	// Test each device to see if it exists
	for _, device := range commonDevices {
		// Try to get stats - if successful, device exists
		_, err := domain.BlockStatsFlags(device, 0)
		if err == nil {
			devices = append(devices, device)
			continue
		}
		// Try basic stats as fallback
		_, err = domain.BlockStats(device)
		if err == nil {
			devices = append(devices, device)
		}
	}

	return devices
}

// discoverNetworkInterfaces attempts to discover available network interfaces for a domain using XML parsing
func (mc *LibvirtMetricsCollector) discoverNetworkInterfaces(domain *libvirt.Domain) []string {
	var interfaces []string

	// Get domain XML description
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		log.Printf("Warning: Failed to get domain XML for interfaces: %v", err)
		return mc.fallbackNetworkInterfaceDiscovery(domain)
	}

	// Parse the XML
	var domainXML libvirtxml.Domain
	if err := xml.Unmarshal([]byte(xmlDesc), &domainXML); err != nil {
		log.Printf("Warning: Failed to parse domain XML for interfaces: %v", err)
		return mc.fallbackNetworkInterfaceDiscovery(domain)
	}

	// Extract network interfaces from XML
	if domainXML.Devices != nil {
		for _, iface := range domainXML.Devices.Interfaces {
			if iface.Target != nil && iface.Target.Dev != "" {
				interfaces = append(interfaces, iface.Target.Dev)
			}
		}
	}

	// If XML parsing didn't find any interfaces, fall back to trial-and-error
	if len(interfaces) == 0 {
		return mc.fallbackNetworkInterfaceDiscovery(domain)
	}

	return interfaces
}

// fallbackNetworkInterfaceDiscovery uses trial-and-error method as fallback
func (mc *LibvirtMetricsCollector) fallbackNetworkInterfaceDiscovery(domain *libvirt.Domain) []string {
	var interfaces []string

	// Common network interface patterns
	commonInterfaces := []string{
		// Standard Ethernet
		"eth0", "eth1", "eth2", "eth3", "eth4", "eth5",
		// Predictable interface names (systemd)
		"ens3", "ens4", "ens5", "ens6", "ens7", "ens8",
		"enp0s3", "enp0s4", "enp0s5", "enp0s6", "enp0s7", "enp0s8",
		"eno1", "eno2", "eno3", "eno4",
		// libvirt virtual interfaces
		"vnet0", "vnet1", "vnet2", "vnet3", "vnet4", "vnet5",
		// VLAN interfaces
		"eth0.1", "eth0.2", "eth1.1", "eth1.2",
		// Bridge interfaces
		"br0", "br1", "br2", "virbr0", "virbr1",
		// Wireless
		"wlan0", "wlan1", "wlp0s3", "wlp0s4",
	}

	// Test each interface to see if it exists
	for _, iface := range commonInterfaces {
		_, err := domain.InterfaceStats(iface)
		if err == nil {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces
}

// CollectDeviceStats collects device statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectDeviceStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) (*DeviceMetrics, error) {
	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	metrics := &DeviceMetrics{
		Name: domainName,
		UUID: domainUUID,
	}

	// Check for TPM
	xmlDesc, err := domain.GetXMLDesc(0)
	if err == nil {
		// Simple check for TPM in XML
		if len([]byte(xmlDesc)) > 0 {
			metrics.HasTPM = false // Would need to parse XML to determine this accurately
			metrics.HasRNG = false // Would need to parse XML to determine this accurately
		}
	}

	return metrics, nil
}

// CollectJobStats collects job statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectJobStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) (*DomainJobMetrics, error) {
	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	metrics := &DomainJobMetrics{
		Name: domainName,
		UUID: domainUUID,
	}

	// Try to get job info
	jobInfo, err := domain.GetJobInfo()
	if err == nil && jobInfo.Type != libvirt.DOMAIN_JOB_NONE {
		metrics.Type = jobTypeToString(jobInfo.Type)
		if jobInfo.DataTotal > 0 {
			metrics.Progress = float64(jobInfo.DataProcessed) / float64(jobInfo.DataTotal)
		}
		metrics.Remaining = jobInfo.DataRemaining
		metrics.Transferred = jobInfo.DataProcessed
		metrics.Total = jobInfo.DataTotal
		if jobInfo.DiskBpsSet {
			metrics.SpeedBps = jobInfo.DiskBps
		}
	}

	return metrics, nil
}

// CollectSnapshotStats collects snapshot statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectSnapshotStats(
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) (*SnapshotMetrics, error) {
	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	// List snapshots to get count
	snapshots, err := domain.ListAllSnapshots(0)
	if err != nil {
		return nil, err
	}

	// Free snapshots
	for _, snapshot := range snapshots {
		snapshot.Free()
	}

	metrics := &SnapshotMetrics{
		Name:  domainName,
		UUID:  domainUUID,
		Count: len(snapshots),
	}

	return metrics, nil
}

// CollectConnectionStats collects connection and host level statistics
func (mc *LibvirtMetricsCollector) CollectConnectionStats(
	conn *libvirt.Connect,
) (*ConnectionMetrics, error) {
	// Get connection URI
	uri, err := conn.GetURI()
	if err != nil {
		uri = "unknown"
	}

	// Check if connection is alive
	isAlive, err := conn.IsAlive()
	if err != nil {
		isAlive = false
	}

	// Get capabilities XML
	capabilities, err := conn.GetCapabilities()
	if err != nil {
		capabilities = ""
	}

	// Get active domains count
	activeDomains, _, err := conn.ListDomains()
	if err != nil {
		activeDomains = []uint{}
	}

	// Get inactive domains count
	inactiveDomains, err := conn.ListDefinedDomains()
	if err != nil {
		inactiveDomains = []string{}
	}

	metrics := &ConnectionMetrics{
		URI:                 uri,
		IsAlive:             isAlive,
		Capabilities:        capabilities,
		ActiveDomainCount:   len(activeDomains),
		InactiveDomainCount: len(inactiveDomains),
	}

	return metrics, nil
}

// CollectHostStats collects host level statistics
func (mc *LibvirtMetricsCollector) CollectHostStats(
	conn *libvirt.Connect,
) (*HostMetrics, error) {
	// Get hostname
	hostname, err := conn.GetHostname()
	if err != nil {
		hostname = "unknown"
	}

	// Get node info
	nodeInfo, err := conn.GetNodeInfo()
	if err != nil {
		return nil, err
	}

	// Get free memory
	freeMemory, err := conn.GetFreeMemory()
	if err != nil {
		freeMemory = 0
	}

	// Get libvirt version
	libvirtVersion, err := conn.GetLibVersion()
	if err != nil {
		libvirtVersion = 0
	}

	// Get hypervisor version
	hypervisorVersion, err := conn.GetVersion()
	if err != nil {
		hypervisorVersion = 0
	}

	metrics := &HostMetrics{
		Name:              hostname,
		TotalCPUCount:     uint64(nodeInfo.Cpus),
		OnlineCPUCount:    uint64(nodeInfo.Cpus), // Simplified, assuming all CPUs are online
		TotalMemoryBytes:  uint64(nodeInfo.Memory) * 1024, // Convert from KB to bytes
		FreeMemoryBytes:   freeMemory,
		Hostname:          hostname,
		LibvirtVersion:    libvirtVersion,
		HypervisorVersion: hypervisorVersion,
	}

	return metrics, nil
}

// Helper function to convert job type to string
func jobTypeToString(jobType libvirt.DomainJobType) string {
	switch jobType {
	case libvirt.DOMAIN_JOB_BOUNDED:
		return "bounded"
	case libvirt.DOMAIN_JOB_UNBOUNDED:
		return "unbounded"
	case libvirt.DOMAIN_JOB_COMPLETED:
		return "completed"
	default:
		return "none"
	}
}
