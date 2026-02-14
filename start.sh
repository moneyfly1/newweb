#!/bin/bash

# CBoard v2 服务管理脚本
# Go 后端 + Vue 前端

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

BACKEND_PORT=9000
BACKEND_BIN="./cboard"
BACKEND_PID_FILE="backend.pid"
BACKEND_LOG="backend.log"

# 函数：停止进程
stop_process() {
    local pid_file=$1
    local name=$2
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            echo -e "${YELLOW}停止 $name (PID: $pid)...${NC}"
            kill "$pid"
            sleep 2
            if kill -0 "$pid" 2>/dev/null; then
                echo -e "${YELLOW}强制停止 $name...${NC}"
                kill -9 "$pid"
            fi
            echo -e "${GREEN}✓ $name 已停止${NC}"
        else
            echo -e "${YELLOW}○ $name 未运行${NC}"
        fi
        rm -f "$pid_file"
    else
        echo -e "${YELLOW}○ $name 未运行${NC}"
    fi
}

# 函数：构建后端
build_backend() {
    echo -e "${YELLOW}构建后端...${NC}"
    go build -o cboard ./cmd/server/
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ 后端构建失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ 后端构建完成${NC}"
}

# 函数：构建前端
build_frontend() {
    echo -e "${YELLOW}构建前端...${NC}"
    cd frontend
    if [ ! -d "node_modules" ]; then
        echo -e "${YELLOW}安装前端依赖...${NC}"
        npm install
    fi
    npx vite build
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ 前端构建失败${NC}"
        cd ..
        exit 1
    fi
    cd ..
    echo -e "${GREEN}✓ 前端构建完成${NC}"
}

# 函数：启动后端
start_backend() {
    # 清理端口
    pids=$(lsof -ti:$BACKEND_PORT 2>/dev/null || true)
    if [ -n "$pids" ]; then
        echo -e "${YELLOW}清理端口 $BACKEND_PORT...${NC}"
        kill -9 $pids 2>/dev/null || true
        sleep 1
    fi

    echo -e "${GREEN}启动后端服务...${NC}"
    nohup $BACKEND_BIN > $BACKEND_LOG 2>&1 &
    local pid=$!
    echo $pid > $BACKEND_PID_FILE
    sleep 2

    if kill -0 $pid 2>/dev/null; then
        echo -e "${GREEN}✓ 后端已启动 (PID: $pid, 端口: $BACKEND_PORT)${NC}"
    else
        echo -e "${RED}✗ 后端启动失败，查看 $BACKEND_LOG${NC}"
        exit 1
    fi
}

# 函数：停止所有服务
stop_services() {
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}   停止 CBoard v2 服务${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""

    stop_process "$BACKEND_PID_FILE" "后端服务"

    # 清理残留
    pkill -9 -f "./cboard" 2>/dev/null || true
    pids=$(lsof -ti:$BACKEND_PORT 2>/dev/null || true)
    if [ -n "$pids" ]; then
        kill -9 $pids 2>/dev/null || true
    fi

    echo ""
    echo -e "${GREEN}✓ 所有服务已停止${NC}"
    echo ""
}

# 函数：启动所有服务（构建+启动）
start_services() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   启动 CBoard v2 服务${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    build_backend
    echo ""
    build_frontend
    echo ""
    start_backend

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   ✅ 启动完成${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "  后端: http://localhost:$BACKEND_PORT"
    echo -e "  前端: 由 Nginx 代理 frontend/dist"
    echo -e "  日志: $BACKEND_LOG"
    echo ""
}

# 函数：重启后端（重新构建+重启）
restart_backend() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   重启后端服务${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    stop_process "$BACKEND_PID_FILE" "后端服务"
    pkill -9 -f "./cboard" 2>/dev/null || true
    pids=$(lsof -ti:$BACKEND_PORT 2>/dev/null || true)
    [ -n "$pids" ] && kill -9 $pids 2>/dev/null || true
    sleep 1

    > $BACKEND_LOG 2>/dev/null || true
    echo -e "${GREEN}✓ 日志已清空${NC}"
    echo ""

    build_backend
    echo ""
    start_backend

    echo ""
    echo -e "${GREEN}✓ 后端重启完成${NC}"
    echo ""
}

# 函数：重新构建前端
rebuild_frontend() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   重新构建前端${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    build_frontend

    echo ""
    echo -e "${GREEN}✓ 前端构建完成，刷新浏览器即可看到更新${NC}"
    echo ""
}

# 函数：重启所有（构建+重启）
restart_all() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   重启 CBoard v2 全部服务${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    stop_process "$BACKEND_PID_FILE" "后端服务"
    pkill -9 -f "./cboard" 2>/dev/null || true
    pids=$(lsof -ti:$BACKEND_PORT 2>/dev/null || true)
    [ -n "$pids" ] && kill -9 $pids 2>/dev/null || true
    sleep 1

    > $BACKEND_LOG 2>/dev/null || true

    build_backend
    echo ""
    build_frontend
    echo ""
    start_backend

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   ✅ 全部重启完成${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "  后端: http://localhost:$BACKEND_PORT"
    echo -e "  日志: $BACKEND_LOG"
    echo ""
}

# 函数：查看状态
show_status() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   服务状态${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            echo -e "${GREEN}✓ 后端运行中 (PID: $pid, 端口: $BACKEND_PORT)${NC}"
        else
            echo -e "${RED}✗ 后端未运行 (PID文件存在但进程不存在)${NC}"
        fi
    else
        echo -e "${YELLOW}○ 后端未运行${NC}"
    fi
    echo ""
}

# 函数：查看日志
show_logs() {
    if [ ! -f "$BACKEND_LOG" ]; then
        echo -e "${RED}日志文件不存在${NC}"
        exit 1
    fi
    echo -e "${YELLOW}按 Ctrl+C 退出日志${NC}"
    echo ""
    tail -f "$BACKEND_LOG"
}

# 函数：显示帮助
show_menu() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}   CBoard v2 服务管理${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "  ./start.sh start            - 构建并启动所有服务"
    echo -e "  ./start.sh stop             - 停止所有服务"
    echo -e "  ./start.sh restart          - 重新构建并重启所有服务"
    echo -e "  ./start.sh restart-backend  - 重新构建并重启后端"
    echo -e "  ./start.sh rebuild          - 重新构建前端"
    echo -e "  ./start.sh status           - 查看服务状态"
    echo -e "  ./start.sh logs             - 实时查看后端日志"
    echo ""
}

# 主逻辑
case "${1:-}" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_all
        ;;
    restart-backend|restart-back)
        restart_backend
        ;;
    rebuild|rebuild-frontend)
        rebuild_frontend
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    *)
        show_menu
        ;;
esac
