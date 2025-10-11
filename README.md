# UOS Libvirt Exporter

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.11-blue.svg)](https://golang.org/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Exporter-green.svg)](https://prometheus.io/)

#### 介绍

UOS Libvirt Exporter 是一个专业的 Prometheus 监控导出器，用于收集和暴露基于 libvirt 的虚拟机（KVM/QEMU）运行状态和性能指标。该工具专为统信UOS操作系统和openEuler社区设计，支持本地和远程 libvirt 实例监控。

### 主要特性

- 🚀 **高性能采集** - 基于 Go 语言开发，支持并发采集和智能缓存
- 🔍 **全面监控** - 覆盖虚拟机状态、CPU、内存、磁盘I/O、网络I/O等关键指标
- 🔌 **灵活连接** - 支持本地和远程 libvirt 连接（qemu:///system, qemu+tcp://host/system）
- 🛡️ **安全可靠** - 支持 TLS/SASL 认证，完善的错误处理和重连机制
- 📊 **Prometheus 原生** - 遵循 Prometheus 最佳实践，支持标签化指标
- ⚙️ **易于部署** - 提供 systemd 服务、Docker 容器等多种部署方式

### 监控指标

- **虚拟机状态** - 运行状态、CPU数量、内存使用情况
- **CPU 性能** - CPU使用时间、vCPU分配情况
- **内存监控** - 当前内存、最大内存、内存使用率
- **磁盘I/O** - 读写字节数、请求次数、I/O时间
- **网络I/O** - 收发字节数、数据包数量、错误统计
- **运行时长** - 虚拟机运行时间统计
- **元数据** - 构建信息、连接状态等

#### 软件架构

```
Prometheus Server ──HTTP──> UOS Libvirt Exporter ──libvirt API──> Libvirtd (QEMU/KVM)
                                      │
                                      └──> 虚拟机指标采集与暴露
```

#### 安装教程

##### 1. 二进制安装

```bash
# 下载最新版本
wget https://github.com/openeuler/uos-libvirtd-exporter/releases/latest/download/uos-libvirtd-exporter-linux-amd64.tar.gz

# 解压
tar -xzf uos-libvirtd-exporter-linux-amd64.tar.gz

# 安装
sudo mv uos-libvirtd-exporter /usr/local/bin/
sudo chmod +x /usr/local/bin/uos-libvirtd-exporter
```

##### 2. 源码编译安装

```bash
# 克隆仓库
git clone https://github.com/openeuler/uos-libvirtd-exporter.git
cd uos-libvirtd-exporter

# 下载依赖
go mod download

# 构建
make build

# 安装
sudo make install
```

##### 3. Docker 部署

```bash
# 使用 Docker 运行
docker run -d \
  --name uos-libvirtd-exporter \
  -p 9177:9177 \
  -v /var/run/libvirt/libvirt-sock:/var/run/libvirt/libvirt-sock:ro \
  openeuler/uos-libvirtd-exporter:latest
```

##### 4. Systemd 服务部署

```bash
# 复制服务文件
sudo cp uos-libvirtd-exporter.service /etc/systemd/system/

# 重新加载 systemd
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable uos-libvirtd-exporter
sudo systemctl start uos-libvirtd-exporter
```

#### 使用说明

##### 基本使用

```bash
# 默认配置运行（连接本地 libvirt）
uos-libvirtd-exporter

# 指定 libvirt URI
uos-libvirtd-exporter -libvirt.uri=qemu:///system

# 指定监听地址和端口
uos-libvirtd-exporter -web.listen-address=:9177

# 指定指标路径
uos-libvirtd-exporter -web.telemetry-path=/metrics
```

##### 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-libvirt.uri` | `qemu:///system` | Libvirt 连接 URI |
| `-web.listen-address` | `:9177` | 监听地址和端口 |
| `-web.telemetry-path` | `/metrics` | 指标路径 |

##### Prometheus 配置

在 Prometheus 配置文件中添加:

```yaml
scrape_configs:
  - job_name: 'libvirt'
    static_configs:
      - targets: ['localhost:9177']
    scrape_interval: 30s
    scrape_timeout: 25s
```

#### 参与贡献

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

#### 许可证

本项目采用 Apache License 2.0 许可证，详见 [LICENSE](LICENSE) 文件。
