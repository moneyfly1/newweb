#!/bin/bash
# 详细的启动检查脚本

echo "========================================="
echo "CBoard 启动诊断"
echo "========================================="
echo ""

# 检查端口占用
echo "1. 检查端口 8000 是否被占用..."
if lsof -i :8000 >/dev/null 2>&1; then
    echo "❌ 端口 8000 已被占用："
    lsof -i :8000
else
    echo "✓ 端口 8000 可用"
fi
echo ""

# 检查数据库
echo "2. 检查数据库..."
if [ -f "cboard.db" ]; then
    echo "✓ 数据库文件存在"
    DB_SIZE=$(ls -lh cboard.db | awk '{print $5}')
    echo "  大小: $DB_SIZE"

    # 检查数据库完整性
    INTEGRITY=$(sqlite3 cboard.db "PRAGMA integrity_check;" 2>&1 | head -1)
    if [ "$INTEGRITY" = "ok" ]; then
        echo "✓ 数据库完整性检查通过"
    else
        echo "❌ 数据库损坏: $INTEGRITY"
    fi
else
    echo "❌ 数据库文件不存在"
fi
echo ""

# 检查环境变量
echo "3. 检查环境变量..."
if [ -n "$SECRET_KEY" ]; then
    echo "✓ SECRET_KEY 已设置"
else
    echo "⚠ SECRET_KEY 未设置（将使用随机密钥）"
fi

if [ -n "$PORT" ]; then
    echo "✓ PORT 已设置: $PORT"
else
    echo "⚠ PORT 未设置（将使用默认端口 8000）"
fi
echo ""

# 检查可执行文件
echo "4. 检查可执行文件..."
if [ -f "./cboard" ]; then
    echo "✓ cboard 可执行文件存在"
    FILE_SIZE=$(ls -lh cboard | awk '{print $5}')
    echo "  大小: $FILE_SIZE"

    # 检查是否可执行
    if [ -x "./cboard" ]; then
        echo "✓ 文件具有执行权限"
    else
        echo "❌ 文件没有执行权限"
        chmod +x ./cboard
        echo "  已添加执行权限"
    fi
else
    echo "❌ cboard 可执行文件不存在"
    echo "  请先运行: go build -o cboard cmd/server/main.go"
fi
echo ""

# 尝试启动并捕获错误
echo "5. 尝试启动服务..."
echo "-----------------------------------"
timeout 3 ./cboard 2>&1 &
CBOARD_PID=$!
sleep 3

if ps -p $CBOARD_PID > /dev/null 2>&1; then
    echo "✓ 服务启动成功（PID: $CBOARD_PID）"
    kill $CBOARD_PID 2>/dev/null
    wait $CBOARD_PID 2>/dev/null
else
    echo "❌ 服务启动失败"
    echo ""
    echo "最后的错误日志："
    echo "-----------------------------------"
    ./cboard 2>&1 | head -50
fi
echo ""

echo "========================================="
echo "诊断完成"
echo "========================================="
