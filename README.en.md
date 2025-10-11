# UOS Libvirt Exporter

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.11-blue.svg)](https://golang.org/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Exporter-green.svg)](https://prometheus.io/)

#### Description

UOS Libvirt Exporter is a professional Prometheus monitoring exporter for collecting and exposing libvirt-based virtual machine (KVM/QEMU) runtime status and performance metrics. This tool is designed specifically for the UOS operating system and openEuler community, supporting monitoring of both local and remote libvirt instances.

### Key Features

- 🚀 **High-performance collection** - Developed in Go language with concurrent collection and intelligent caching
- 🔍 **Comprehensive monitoring** - Covers key metrics including VM status, CPU, memory, disk I/O, and network I/O
- 🔌 **Flexible connection** - Supports local and remote libvirt connections (qemu:///system, qemu+tcp://host/system)
- 🛡️ **Secure and reliable** - Supports TLS/SASL authentication with comprehensive error handling and reconnection mechanisms
- 📊 **Prometheus native** - Follows Prometheus best practices with label-based metrics
- ⚙️ **Easy deployment** - Provides multiple deployment options including systemd service and Docker container

### Monitoring Metrics

- **VM Status** - Running status, CPU count, memory usage
- **CPU Performance** - CPU time usage, vCPU allocation
- **Memory Monitoring** - Current memory, maximum memory, memory usage ratio
- **Disk I/O** - Read/write bytes, request counts, I/O time
- **Network I/O** - Received/sent bytes, packet counts, error statistics
- **Uptime** - VM running time statistics
- **Metadata** - Build information, connection status, etc.

#### Software Architecture

```
Prometheus Server ──HTTP──> UOS Libvirt Exporter ──libvirt API──> Libvirtd (QEMU/KVM)
                                      │
                                      └──> VM metrics collection and exposure
```

#### Installation

##### 1. Binary Installation

```bash
# Download the latest version
wget https://github.com/openeuler/uos-libvirtd-exporter/releases/latest/download/uos-libvirtd-exporter-linux-amd64.tar.gz

# Extract
tar -xzf uos-libvirtd-exporter-linux-amd64.tar.gz

# Install
sudo mv uos-libvirtd-exporter /usr/local/bin/
sudo chmod +x /usr/local/bin/uos-libvirtd-exporter
```

##### 2. Building from Source

```bash
# Clone the repository
git clone https://github.com/openeuler/uos-libvirtd-exporter.git
cd uos-libvirtd-exporter

# Download dependencies
go mod download

# Build
make build

# Install
sudo make install
```

##### 3. Docker Deployment

```bash
# Run with Docker
docker run -d \
  --name uos-libvirtd-exporter \
  -p 9177:9177 \
  -v /var/run/libvirt/libvirt-sock:/var/run/libvirt/libvirt-sock:ro \
  openeuler/uos-libvirtd-exporter:latest
```

##### 4. Systemd Service Deployment

```bash
# Copy service file
sudo cp uos-libvirtd-exporter.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable uos-libvirtd-exporter
sudo systemctl start uos-libvirtd-exporter
```

#### Instructions

##### Basic Usage

```bash
# Run with default configuration (connect to local libvirt)
uos-libvirtd-exporter

# Specify libvirt URI
uos-libvirtd-exporter -libvirt.uri=qemu:///system

# Specify listen address and port
uos-libvirtd-exporter -web.listen-address=:9177

# Specify metrics path
uos-libvirtd-exporter -web.telemetry-path=/metrics
```

##### Configuration Parameters

| Parameter | Default Value | Description |
|-----------|---------------|-------------|
| `-libvirt.uri` | `qemu:///system` | Libvirt connection URI |
| `-web.listen-address` | `:9177` | Listen address and port |
| `-web.telemetry-path` | `/metrics` | Metrics path |

##### Prometheus Configuration

Add to your Prometheus configuration file:

```yaml
scrape_configs:
  - job_name: 'libvirt'
    static_configs:
      - targets: ['localhost:9177']
    scrape_interval: 30s
    scrape_timeout: 25s
```

#### Contribution

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

#### License

This project is licensed under the Apache License 2.0, see the [LICENSE](LICENSE) file for details.