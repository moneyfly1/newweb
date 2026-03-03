#!/bin/bash

# 生产环境快速部署脚本

echo "=========================================="
echo "CBoard 生产环境部署"
echo "=========================================="
echo ""

# 1. 拉取最新代码
echo "步骤 1/5: 拉取最新代码..."
git pull origin main
if [ $? -ne 0 ]; then
    echo "❌ 拉取代码失败"
    exit 1
fi
echo "✅ 代码已更新"
echo ""

# 2. 编译
echo "步骤 2/5: 编译项目..."
go build -o cboard-server cmd/server/main.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi
echo "✅ 编译成功"
echo ""

# 3. 停止旧服务
echo "步骤 3/5: 停止旧服务..."
OLD_PID=$(ps aux | grep cboard-server | grep -v grep | awk '{print $2}')
if [ -n "$OLD_PID" ]; then
    kill $OLD_PID
    sleep 2
    echo "✅ 旧服务已停止 (PID: $OLD_PID)"
else
    echo "ℹ️  没有运行中的服务"
fi
echo ""

# 4. 启动新服务
echo "步骤 4/5: 启动新服务..."
nohup ./cboard-server > /dev/null 2>&1 &
NEW_PID=$!
sleep 2

# 检查服务是否启动成功
if ps -p $NEW_PID > /dev/null; then
    echo "✅ 新服务已启动 (PID: $NEW_PID)"
else
    echo "❌ 服务启动失败"
    exit 1
fi
echo ""

# 5. 验证服务
echo "步骤 5/5: 验证服务..."
sleep 3
if curl -s http://localhost:8000/api/v1/health > /dev/null 2>&1; then
    echo "✅ 服务运行正常"
else
    echo "⚠️  健康检查失败，但服务可能仍在启动中"
fi
echo ""

echo "=========================================="
echo "部署完成！"
echo "=========================================="
echo ""
echo "查看日志:"
echo "  tail -f logs/app-\$(date +%Y-%m-%d).log"
echo ""
echo "查看支付日志:"
echo "  tail -f logs/app-\$(date +%Y-%m-%d).log | grep PAYMENT"
echo ""
echo "查看回调日志:"
echo "  tail -f logs/app-\$(date +%Y-%m-%d).log | grep CALLBACK"
echo ""
echo "停止服务:"
echo "  kill $NEW_PID"
echo ""
