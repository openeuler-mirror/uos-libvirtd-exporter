#!/bin/bash
set -e

# 检查是否提供了远程主机地址
REMOTE_HOST=${LIBVIRT_REMOTE_HOST:-""}
if [ -n "$REMOTE_HOST" ]; then
  echo "Setting up SSH known hosts for $REMOTE_HOST"
  # 添加远程主机的 SSH 指纹
  ssh-keyscan "$REMOTE_HOST" >> /root/.ssh/known_hosts 2>/dev/null || true
fi

# 检查网络连接
if [ -n "$REMOTE_HOST" ]; then
  echo "Checking connectivity to $REMOTE_HOST"
  if ! nc -zv "$REMOTE_HOST" 22 >/dev/null 2>&1; then
    echo "ERROR: Cannot connect to SSH port on $REMOTE_HOST"
    exit 1
  fi
fi

# 设置默认的libvirt URI，如果通过环境变量指定了URI，则使用环境变量中的值
LIBVIRT_URI=${LIBVIRT_URI:-"qemu:///system"}

# 启动应用，将libvirt URI作为参数传递
echo "Starting UOS Libvirt Exporter with URI: $LIBVIRT_URI"
exec /usr/local/bin/uos-libvirtd-exporter --libvirt.uri="$LIBVIRT_URI" "$@"