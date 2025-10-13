# Build stage
FROM golang:1.25-alpine3.21 AS builder

# 配置 USTC 镜像源（Alpine 3.21）
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine/|https://mirrors.ustc.edu.cn/alpine/|g' /etc/apk/repositories

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev libvirt-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o uos-libvirtd-exporter


# Runtime stage
FROM alpine:latest

# 配置 USTC 镜像源（适用于 alpine:latest，通常为最新版）
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine/|https://mirrors.ustc.edu.cn/alpine/|g' /etc/apk/repositories

# Install runtime dependencies
RUN apk add --no-cache libvirt-client openssh-client

# Create non-root user
RUN adduser -D -g '' exporter

# Copy binary from builder
COPY --from=builder /build/uos-libvirtd-exporter /usr/local/bin/uos-libvirtd-exporter

# Copy config file
COPY --from=builder /build/config.yaml /etc/uos-libvirtd-exporter/config.yaml

# Change ownership
RUN chown exporter:exporter /usr/local/bin/uos-libvirtd-exporter
RUN chown exporter:exporter /etc/uos-libvirtd-exporter/config.yaml

# Switch to non-root user
USER exporter

# Expose metrics port
EXPOSE 9177

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9177/ || exit 1

# Run the exporter
ENTRYPOINT ["/usr/local/bin/uos-libvirtd-exporter"]