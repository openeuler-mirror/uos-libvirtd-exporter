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
  if ! ping -c 1 "$REMOTE_HOST" >/dev/null 2>&1; then
    echo "ERROR: Cannot ping remote host $REMOTE_HOST"
    exit 1
  fi
  
  if ! nc -zv "$REMOTE_HOST" 22 >/dev/null 2>&1; then
    echo "ERROR: Cannot connect to SSH port on $REMOTE_HOST"
    exit 1
  fi
fi

# 启动应用
echo "Starting UOS Libvirt Exporter"
exec /usr/local/bin/uos-libvirtd-exporter "$@"