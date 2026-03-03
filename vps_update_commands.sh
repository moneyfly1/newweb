#!/bin/bash
# VPS 服务器更新脚本
# 执行方式：bash vps_update_commands.sh

set -e  # 遇到错误立即停止

echo "=========================================="
echo "  cboard 支付回调修复更新"
echo "=========================================="
echo ""

# 检查是否在正确的目录
if [ ! -f "cboard.db" ]; then
    echo "❌ 错误：未找到 cboard.db 文件"
    echo "请先 cd 到 cboard 安装目录（通常是 /root/cboard）"
    exit 1
fi

echo "✓ 当前目录：$(pwd)"
echo ""

# 1. 备份数据库
echo "📦 步骤 1/7: 备份数据库..."
BACKUP_FILE="cboard.db.backup.$(date +%Y%m%d_%H%M%S)"
cp cboard.db "$BACKUP_FILE"
echo "✓ 数据库已备份到: $BACKUP_FILE"
echo ""

# 2. 停止服务
echo "🛑 步骤 2/7: 停止服务..."
if pgrep -f "./cboard" > /dev/null; then
    pkill -f "./cboard"
    sleep 2
    echo "✓ 服务已停止"
else
    echo "✓ 服务未运行"
fi
echo ""

# 3. 拉取最新代码
echo "📥 步骤 3/7: 拉取最新代码..."
git pull origin main
echo "✓ 代码已更新"
echo ""

# 4. 执行数据库迁移
echo "🗄️  步骤 4/7: 执行数据库迁移..."
if [ -f "migrations/add_payment_nonces.sql" ]; then
    sqlite3 cboard.db < migrations/add_payment_nonces.sql
    echo "✓ 数据库迁移完成"
else
    echo "❌ 错误：未找到迁移脚本"
    exit 1
fi
echo ""

# 5. 验证表创建
echo "🔍 步骤 5/7: 验证表创建..."
TABLE_EXISTS=$(sqlite3 cboard.db "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='payment_nonces';")
if [ "$TABLE_EXISTS" = "1" ]; then
    echo "✓ payment_nonces 表已创建"
else
    echo "❌ 错误：表创建失败"
    exit 1
fi
echo ""

# 6. 重新编译
echo "🔨 步骤 6/7: 重新编译..."
go build -o cboard cmd/server/main.go
echo "✓ 编译完成"
echo ""

# 7. 启动服务
echo "🚀 步骤 7/7: 启动服务..."
mkdir -p logs
nohup ./cboard > logs/app.log 2>&1 &
sleep 3

# 验证服务启动
if pgrep -f "./cboard" > /dev/null; then
    PID=$(pgrep -f "./cboard")
    echo "✓ 服务已启动，PID: $PID"
else
    echo "❌ 错误：服务启动失败"
    echo "查看日志："
    tail -20 logs/app.log
    exit 1
fi
echo ""

# 8. 最终验证
echo "=========================================="
echo "  更新完成！正在验证..."
echo "=========================================="
echo ""

# 检查 API 响应
echo "测试 API 响应..."
sleep 2
if curl -s http://localhost:9000/api/v1/config > /dev/null 2>&1; then
    echo "✓ API 响应正常"
else
    echo "⚠️  警告：API 未响应，请检查日志"
fi
echo ""

# 显示最新订单的 transaction_id 格式
echo "检查最新支付事务格式..."
LATEST_TX=$(sqlite3 cboard.db "SELECT transaction_id FROM payment_transactions ORDER BY id DESC LIMIT 1;" 2>/dev/null || echo "无数据")
echo "最新 transaction_id: $LATEST_TX"
echo "（新订单应为 PAY 开头）"
echo ""

echo "=========================================="
echo "  ✅ 更新成功！"
echo "=========================================="
echo ""
echo "📋 后续操作："
echo "1. 创建新订单测试支付功能"
echo "2. 监控日志：tail -f logs/app.log | grep -E 'CreatePayment|Alipay'"
echo "3. 查看进程：ps aux | grep cboard"
echo ""
echo "📁 备份文件：$BACKUP_FILE"
echo "如需回滚，请查看 UPDATE_GUIDE.md"
echo ""
