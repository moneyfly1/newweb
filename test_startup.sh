#!/bin/bash
# 服务器启动测试脚本

echo "========================================="
echo "CBoard 启动测试"
echo "========================================="
echo ""

# 检查 Go 版本
echo "1. 检查 Go 版本..."
go version
echo ""

# 检查依赖
echo "2. 检查依赖..."
go mod verify
echo ""

# 尝试编译
echo "3. 尝试编译..."
go build -o cboard cmd/server/main.go
if [ $? -eq 0 ]; then
    echo "✓ 编译成功"
else
    echo "❌ 编译失败"
    exit 1
fi
echo ""

# 检查数据库文件
echo "4. 检查数据库..."
if [ -f "cboard.db" ]; then
    echo "✓ 数据库文件存在"
    sqlite3 cboard.db "PRAGMA integrity_check;" | head -1
else
    echo "❌ 数据库文件不存在"
fi
echo ""

# 尝试启动（5秒后自动停止）
echo "5. 尝试启动服务（5秒测试）..."
timeout 5 ./cboard 2>&1 | head -100 &
PID=$!
sleep 5
kill $PID 2>/dev/null
echo ""

echo "========================================="
echo "测试完成"
echo "========================================="
