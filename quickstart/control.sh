#!/bin/bash

# UOS Libvirt Exporter 控制脚本
# 用于管理 quickstart 环境中的服务启停

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 脚本信息
SCRIPT_VERSION="1.0.0"
SCRIPT_NAME="UOS Libvirt Exporter Control Script"

# 服务状态
SERVICES=("uos-libvirtd-exporter" "uos-prometheus" "uos-grafana")

# 显示脚本使用说明
show_help() {
    echo -e "${CYAN}==============================================================================${NC}"
    echo -e "${CYAN}                    ${SCRIPT_NAME}${NC}"
    echo -e "${CYAN}==============================================================================${NC}"
    echo ""
    echo -e "${YELLOW}使用方法:${NC}"
    echo "  ./control.sh [选项]"
    echo ""
    echo -e "${YELLOW}选项:${NC}"
    echo "  start     启动所有服务"
    echo "  stop      停止所有服务"
    echo "  restart   重启所有服务"
    echo "  status    查看服务状态"
    echo "  logs      查看服务日志"
    echo "  clean     清理所有服务和数据"
    echo "  help      显示此帮助信息"
    echo ""
    echo -e "${YELLOW}示例:${NC}"
    echo "  ./control.sh start"
    echo "  ./control.sh stop"
    echo ""
    echo -e "${CYAN}==============================================================================${NC}"
}

# 检查 docker-compose 是否可用
check_docker_compose() {
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo -e "${RED}错误: 未找到 docker-compose 或 docker compose${NC}"
        echo "请先安装 Docker 和 Docker Compose"
        exit 1
    fi
}

# 检查是否在正确的目录中
check_directory() {
    if [ ! -f "docker-compose.yml" ]; then
        echo -e "${RED}错误: 未找到 docker-compose.yml 文件${NC}"
        echo "请在 quickstart 目录中运行此脚本"
        exit 1
    fi
}

# 启动服务
start_services() {
    echo -e "${BLUE}正在启动 UOS Libvirt Exporter 环境...${NC}"
    echo ""
    
    # 检查服务是否已在运行
    if docker compose ps | grep -q "running"; then
        echo -e "${YELLOW}服务已在运行中${NC}"
        show_status
        return 0
    fi
    
    # 启动服务
    if docker compose up -d; then
        echo ""
        echo -e "${GREEN}✓ 服务启动成功!${NC}"
        echo ""
        show_urls
    else
        echo -e "${RED}✗ 服务启动失败${NC}"
        exit 1
    fi
}

# 停止服务
stop_services() {
    echo -e "${BLUE}正在停止服务...${NC}"
    
    if docker compose down; then
        echo -e "${GREEN}✓ 服务已停止${NC}"
    else
        echo -e "${RED}✗ 停止服务时出错${NC}"
        exit 1
    fi
}

# 重启服务
restart_services() {
    echo -e "${BLUE}正在重启服务...${NC}"
    stop_services
    sleep 2
    start_services
}

# 显示服务状态
show_status() {
    echo -e "${BLUE}服务状态:${NC}"
    echo -e "${CYAN}------------------------------------------------------------------------------${NC}"
    docker compose ps
    echo -e "${CYAN}------------------------------------------------------------------------------${NC}"
}

# 显示访问URL
show_urls() {
    echo -e "${BLUE}访问信息:${NC}"
    echo -e "${CYAN}------------------------------------------------------------------------------${NC}"
    echo -e "${GREEN}Grafana:${NC}     http://localhost:3000  (默认用户/密码: admin/admin)"
    echo -e "${GREEN}Prometheus:${NC}  http://localhost:9091"
    echo -e "${GREEN}Exporter:${NC}    http://localhost:9177/metrics"
    echo -e "${CYAN}------------------------------------------------------------------------------${NC}"
    echo ""
    echo -e "${YELLOW}提示:${NC}"
    echo "  1. 首次访问 Grafana 时需要修改默认密码"
    echo "  2. Prometheus 已预配置为目标 'uos-libvirtd-exporter'"
    echo "  3. 可能需要等待几秒钟让服务完全启动"
}

# 查看日志
show_logs() {
    echo -e "${BLUE}选择要查看日志的服务:${NC}"
    echo "1) uos-libvirtd-exporter"
    echo "2) uos-prometheus"
    echo "3) uos-grafana"
    echo "4) 所有服务"
    echo ""
    echo -n "请输入选项 (1-4): "
    read -r choice
    
    case $choice in
        1)
            echo -e "${BLUE}uos-libvirtd-exporter 日志:${NC}"
            docker compose logs -f uos-libvirtd-exporter
            ;;
        2)
            echo -e "${BLUE}uos-prometheus 日志:${NC}"
            docker compose logs -f uos-prometheus
            ;;
        3)
            echo -e "${BLUE}uos-grafana 日志:${NC}"
            docker compose logs -f uos-grafana
            ;;
        4)
            echo -e "${BLUE}所有服务日志:${NC}"
            docker compose logs -f
            ;;
        *)
            echo -e "${RED}无效选项${NC}"
            ;;
    esac
}

# 清理所有数据
clean_services() {
    echo -e "${YELLOW}警告: 此操作将删除所有服务和数据卷${NC}"
    echo -n "确认继续? (y/N): "
    read -r confirm
    
    if [[ $confirm == [yY] ]]; then
        echo -e "${BLUE}正在清理所有服务和数据...${NC}"
        docker compose down -v
        echo -e "${GREEN}✓ 清理完成${NC}"
    else
        echo -e "${BLUE}操作已取消${NC}"
    fi
}

# 主函数
main() {
    # 检查依赖和目录
    check_docker_compose
    check_directory
    
    # 如果没有参数，显示帮助
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi
    
    # 处理命令行参数
    case "$1" in
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        clean)
            clean_services
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}未知选项: $1${NC}"
            echo "使用 './control.sh help' 查看帮助信息"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"