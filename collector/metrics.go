package collector

import (
	"libvirt.org/go/libvirt"
	"time"
)

// DomainInfoMetrics represents the basic domain runtime information
type DomainInfoMetrics struct {
	Name          string  // domain name
	UUID          string  // domain uuid
	Status        float64 // domain state (running, paused, etc.)
	StateReason   string  // optional: state reason description
	CPUTime       float64 // accumulated CPU time (ns)
	Uptime        float64 // uptime in seconds
	HasUptime     bool
	MemoryCurrent float64   // current memory usage (bytes)
	MemoryMax     float64   // maximum configured memory (bytes)
	Autostart     bool      // domain autostart flag
	Persistent    bool      // whether domain is persistent
	ManagedSave   bool      // managed save image exists
	BootTime      time.Time // guest boot time
}

// CPUStatsMetrics represents vCPU and scheduling metrics
type CPUStatsMetrics struct {
	Name         string
	UUID         string
	VCPUsMax     uint   // maximum vCPU count
	VCPUsCurrent uint   // current active vCPUs
	CPUTime      uint64 // total CPU time (ns)
	UserTime     uint64 // guest user time (ns)
	SystemTime   uint64 // guest system time (ns)
	StealTime    uint64 // vCPU steal time (ns)
	Scheduler    string // scheduler type (e.g. "cfs", "rt")
	Quota        int64  // CPU quota in microseconds
	Period       int64  // CPU period in microseconds
	Affinity     string // CPU affinity bitmap string
}

// MemoryStatsMetrics represents guest memory balloon and usage metrics
type MemoryStatsMetrics struct {
	Name        string
	UUID        string
	BalloonSize uint64 // current balloon size (KB)
	Unused      uint64 // guest unused memory (KB)
	Available   uint64 // guest available memory (KB)
	RSS         uint64 // resident set size (KB)
	SwapIn      uint64 // swap in (KB)
	SwapOut     uint64 // swap out (KB)
	MajorFaults uint64 // major page faults
	MinorFaults uint64 // minor page faults
	Total       uint64 // total assigned memory (KB)
	NUMANodes   []NUMANodeMemory
}

// NUMANodeMemory represents per-node memory statistics
type NUMANodeMemory struct {
	NodeID  int
	UsedKB  uint64
	TotalKB uint64
	FreeKB  uint64
}

// DiskMetrics represents raw disk I/O and capacity metrics
type DiskMetrics struct {
	Name        string
	UUID        string
	Device      string
	Path        string
	ReadBytes   uint64
	WriteBytes  uint64
	ReadOps     uint64
	WriteOps    uint64
	ReadTimeNs  uint64
	WriteTimeNs uint64
	FlushOps    uint64
	FlushBytes  uint64
	Capacity    uint64 // total virtual disk size
	Allocation  uint64 // allocated bytes on host
	Physical    uint64 // physical bytes consumed on storage
	CacheMode   string
	BlockJob    *BlockJobMetrics
}

// BlockJobMetrics represents active disk job (e.g. commit, copy, mirror)
type BlockJobMetrics struct {
	Type      string  // "copy", "commit", "active-commit", etc.
	Progress  float64 // 0.0 - 1.0
	Bandwidth uint64  // bytes per second
}

// NetworkMetrics represents network interface statistics
type NetworkMetrics struct {
	Name        string
	UUID        string
	Interface   string
	MACAddress  string
	Type        string // bridge, macvtap, vhostuser, etc.
	RxBytes     uint64
	TxBytes     uint64
	RxPackets   uint64
	TxPackets   uint64
	RxErrors    uint64
	TxErrors    uint64
	RxDrops     uint64
	TxDrops     uint64
	BandwidthRx uint64 // bandwidth limit (bps)
	BandwidthTx uint64 // bandwidth limit (bps)
	Multiqueue  bool
}

// DeviceMetrics represents virtual devices attached to the domain
type DeviceMetrics struct {
	Name        string
	UUID        string
	HasTPM      bool
	HasRNG      bool
	PCIDevices  []PCIDevice
	USBDevices  []USBDevice
	VGPUDevices []VGPUDevice
	Snapshots   int
}

// PCIDevice represents a PCI passthrough device
type PCIDevice struct {
	Address string // e.g. "0000:00:02.0"
	Type    string // e.g. "GPU", "NIC"
	Driver  string // vfio-pci, etc.
}

// USBDevice represents a USB passthrough device
type USBDevice struct {
	Bus     int
	Device  int
	Product string
	Vendor  string
}

// VGPUDevice represents mediated device (vGPU)
type VGPUDevice struct {
	MdevUUID string
	Model    string // e.g. "nvidia-222"
}

// DomainJobMetrics represents job progress (e.g. migration, block copy)
type DomainJobMetrics struct {
	Name        string
	UUID        string
	Type        string  // "migration", "block-commit", etc.
	Progress    float64 // 0.0 ~ 1.0
	Remaining   uint64  // bytes remaining
	Transferred uint64  // bytes transferred
	Total       uint64  // total bytes
	SpeedBps    uint64  // current transfer speed (B/s)
}

// SnapshotMetrics represents snapshot statistics
type SnapshotMetrics struct {
	Name       string
	UUID       string
	Count      int
	LastCreate time.Time
	LastDelete time.Time
}

// ConnectionMetrics represents libvirt connection and host statistics
type ConnectionMetrics struct {
	Hostname            string
	LibvirtVersion      uint64
	HypervisorVersion   uint64
	DriverType          string
	IsAlive             bool
	ActiveDomains       int
	DefinedDomains      int
	FreeMemoryBytes     uint64
	TotalMemoryBytes    uint64
	TotalCPUs           int
	HostCPUUsagePercent float64
	StoragePools        []StoragePoolMetrics
	Networks            []NetworkPoolMetrics
	Interfaces          []HostInterfaceMetrics
}

// StoragePoolMetrics represents storage pool stats
type StoragePoolMetrics struct {
	Name       string
	Type       string
	State      string
	Capacity   uint64
	Allocation uint64
	Available  uint64
	Volumes    int
}

// NetworkPoolMetrics represents virtual network stats
type NetworkPoolMetrics struct {
	Name   string
	Active bool
	Bridge string
}

// HostInterfaceMetrics represents physical NIC stats on host
type HostInterfaceMetrics struct {
	Name      string
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
}

// ExporterMetrics represents exporter self-monitoring metrics
type ExporterMetrics struct {
	Up                bool      // exporter connected to libvirt successfully
	LastScrapeTime    time.Time // last successful scrape time
	ScrapeDurationSec float64   // time taken for last scrape
	ScrapeErrorsTotal uint64    // total scrape errors
	DomainsDiscovered int       // number of domains discovered
	CacheHits         uint64
	CacheMisses       uint64
	BuildVersion      string
	BuildCommit       string
}

// DomainMetrics aggregates all metrics for one domain
type DomainMetrics struct {
	Info     DomainInfoMetrics
	CPU      CPUStatsMetrics
	Memory   MemoryStatsMetrics
	Disks    []DiskMetrics
	Networks []NetworkMetrics
	Devices  DeviceMetrics
	Job      *DomainJobMetrics
	Snapshot SnapshotMetrics
}

// MetricsCollector defines interface for collecting raw metrics from libvirt
type MetricsCollector interface {
	CollectDomainInfo(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*DomainInfoMetrics, error)
	CollectCPUStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*CPUStatsMetrics, error)
	CollectMemoryStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*MemoryStatsMetrics, error)
	CollectDiskStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) ([]DiskMetrics, error)
	CollectNetworkStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) ([]NetworkMetrics, error)
	CollectDeviceStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*DeviceMetrics, error)
	CollectJobStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*DomainJobMetrics, error)
	CollectSnapshotStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*SnapshotMetrics, error)
}
