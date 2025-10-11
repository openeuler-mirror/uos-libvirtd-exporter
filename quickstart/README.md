# UOS Libvirt Exporter 快速启动指南

本目录包含使用 Docker Compose 快速部署 uos-libvirtd-exporter、Prometheus 和 Grafana 的完整监控解决方案所需的所有文件。

## 目录结构

```
quickstart/
├── grafana/                 # Grafana 配置文件
│   ├── dashboards/          # 预加载的仪表板
│   │   └── libvirt-dashboard.json
│   ├── provisioning/        # Grafana 预配配置
│   │   └── dashboards/
│   │       └── dashboard.yml
│   └── config.ini           # Grafana 配置文件
├── Dockerfile               # uos-libvirtd-exporter 镜像构建文件
├── dashboard.json           # Grafana 仪表板配置文件
├── docker-compose.yml       # Docker Compose 编排文件
└── prometheus.yml           # Prometheus 配置文件
```

## 快速开始

### 前提条件

1. 确保已安装 Docker 和 Docker Compose
2. 确保主机上运行了 libvirtd 服务
3. 确保当前用户有权限访问 libvirt 套接字（通常需要将用户添加到 libvirt 组）

### 部署环境

```bash
# 克隆项目
git clone https://github.com/openeuler/uos-libvirtd-exporter.git
cd uos-libvirtd-exporter/quickstart

# 启动所有服务
docker-compose up -d
```

这将启动三个容器：
1. **uos-libvirtd-exporter** - 从本地 Dockerfile 构建的镜像，连接到宿主机的 libvirt 套接字
2. **Prometheus** - 时间序列数据库，用于存储指标数据
3. **Grafana** - 数据可视化平台，预配置了仪表板

### 访问服务

服务启动后，可以通过以下地址访问：

- **Grafana**: http://localhost:3000 (默认用户/密码: admin/admin)
- **Prometheus**: http://localhost:9090
- **Libvirt Exporter Metrics**: http://localhost:9100/metrics

### 验证部署

检查服务是否正常运行:

```bash
# 检查容器状态
docker-compose ps

# 检查 libvirt-exporter 指标
curl http://localhost:9100/metrics

# 检查 Prometheus 目标状态
curl http://localhost:9090/api/v1/targets
```

## 仪表板说明

Grafana 已预配置了 Libvirt 虚拟机监控仪表板，包含以下面板：

### 概览部分
- 运行中的虚拟机数量
- 总虚拟机数
- CPU 使用率 (%)

### 虚拟机状态部分
- 虚拟机状态统计（运行中/已停止）
- 虚拟机详细状态表格

### CPU 和内存部分
- CPU 使用时间
- 当前内存使用量

### 磁盘 I/O 部分
- 磁盘读取速度
- 磁盘写入速度

### 网络 I/O 部分
- 网络接收速度
- 网络发送速度

## 自定义配置

### 修改抓取间隔

编辑 [prometheus.yml](prometheus.yml) 文件:
```yaml
global:
  scrape_interval: 10s # 更改为所需间隔
```

然后重启 Prometheus:
```bash
docker-compose restart prometheus
```

### 使用不同的端口

编辑 [docker-compose.yml](docker-compose.yml) 文件，修改 ports 部分:
```yaml
ports:
  - "9101:9100" # 将主机端口从 9100 更改为 9101
```

## 故障排除

### 权限问题

如果 exporter 无法连接到 libvirt，请确保：

1. 用户在 libvirt 组中:
   ```bash
   sudo usermod -a -G libvirt $USER
   ```

2. libvirt 套接字文件权限正确:
   ```bash
   ls -la /var/run/libvirt/libvirt-sock
   ```

### 容器无法启动

检查日志以获取更多信息:
```bash
docker-compose logs libvirtd-exporter
```

### 没有数据显示在 Grafana 中

1. 检查 Prometheus 目标状态是否为 UP
2. 验证 Prometheus 是否能抓取指标:
   ```bash
   curl http://localhost:9090/api/v1/query?query=libvirt_domain_state
   ```

## 停止环境

```bash
# 停止所有服务但保留数据
docker-compose down

# 停止服务并清除所有数据
docker-compose down -v
```

## 架构图

```
+------------------+      +-----------------+      +------------+
|                  |      |                 |      |            |
|  Libvirt Daemon  |<---->| Exporter Agent  |<---->| Prometheus |
|  (Host System)   |      | (Container)     |      | (Container)|
+------------------+      +-----------------+      +------------+
                                                    |
                                                    v
                                              +------------+
                                              |            |
                                              |  Grafana   |
                                              | (Container)|
                                              +------------+
```

## 相关链接

- [UOS Libvirt Exporter GitHub](https://github.com/openeuler/uos-libvirtd-exporter)
- [Grafana](https://grafana.com/)
- [Prometheus](https://prometheus.io/)
- [Libvirt](https://libvirt.org/)