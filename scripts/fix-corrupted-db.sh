#!/bin/bash
# 修复损坏的 SQLite 数据库
# 使用方法: ./fix-corrupted-db.sh

set -e

echo "=========================================="
echo "SQLite 数据库修复工具"
echo "=========================================="

# 检查是否在正确的目录
if [ ! -f "cboard.db" ]; then
    echo "错误: 未找到 cboard.db 文件"
    echo "请在包含 cboard.db 的目录中运行此脚本"
    exit 1
fi

# 停止应用服务
echo ""
echo "步骤 1: 停止应用服务..."
if systemctl is-active --quiet cboard; then
    sudo systemctl stop cboard
    echo "✓ 服务已停止"
else
    echo "✓ 服务未运行"
fi

# 备份损坏的数据库
echo ""
echo "步骤 2: 备份损坏的数据库..."
BACKUP_FILE="cboard.db.corrupted.$(date +%Y%m%d_%H%M%S)"
cp cboard.db "$BACKUP_FILE"
echo "✓ 已备份到: $BACKUP_FILE"

# 尝试使用 sqlite3 恢复数据
echo ""
echo "步骤 3: 尝试恢复数据..."
RECOVERED_FILE="cboard.db.recovered"

if command -v sqlite3 &> /dev/null; then
    echo "使用 sqlite3 尝试恢复..."

    # 方法 1: 使用 .dump 和 .read 恢复
    if sqlite3 cboard.db ".dump" > dump.sql 2>/dev/null; then
        echo "✓ 成功导出数据"
        sqlite3 "$RECOVERED_FILE" < dump.sql
        echo "✓ 成功导入到新数据库"

        # 验证恢复的数据库
        if sqlite3 "$RECOVERED_FILE" "PRAGMA integrity_check;" | grep -q "ok"; then
            echo "✓ 数据库完整性检查通过"

            # 替换损坏的数据库
            mv cboard.db "cboard.db.old"
            mv "$RECOVERED_FILE" cboard.db
            rm -f dump.sql

            echo ""
            echo "=========================================="
            echo "✓ 数据库恢复成功！"
            echo "=========================================="
        else
            echo "✗ 恢复的数据库仍有问题"
            rm -f "$RECOVERED_FILE" dump.sql
            echo ""
            echo "需要重新初始化数据库（将丢失所有数据）"
            read -p "是否继续？(y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -f cboard.db
                echo "✓ 已删除损坏的数据库"
                echo "应用启动时将自动创建新数据库"
            else
                echo "操作已取消"
                exit 1
            fi
        fi
    else
        echo "✗ 无法导出数据，数据库损坏严重"
        echo ""
        echo "需要重新初始化数据库（将丢失所有数据）"
        read -p "是否继续？(y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -f cboard.db
            echo "✓ 已删除损坏的数据库"
            echo "应用启动时将自动创建新数据库"
        else
            echo "操作已取消"
            exit 1
        fi
    fi
else
    echo "✗ 未安装 sqlite3 命令行工具"
    echo "正在安装..."

    if command -v apt-get &> /dev/null; then
        sudo apt-get update && sudo apt-get install -y sqlite3
    elif command -v yum &> /dev/null; then
        sudo yum install -y sqlite
    else
        echo "✗ 无法自动安装 sqlite3，请手动安装后重试"
        exit 1
    fi

    echo "✓ sqlite3 安装完成，请重新运行此脚本"
    exit 0
fi

# 清理 WAL 和 SHM 文件
echo ""
echo "步骤 4: 清理临时文件..."
rm -f cboard.db-wal cboard.db-shm
echo "✓ 已清理 WAL 和 SHM 文件"

# 设置正确的权限
echo ""
echo "步骤 5: 设置文件权限..."
chmod 644 cboard.db
echo "✓ 权限已设置"

# 启动应用服务
echo ""
echo "步骤 6: 启动应用服务..."
sudo systemctl start cboard
sleep 2

if systemctl is-active --quiet cboard; then
    echo "✓ 服务启动成功"
    echo ""
    echo "=========================================="
    echo "修复完成！"
    echo "=========================================="
    echo ""
    echo "查看日志: sudo journalctl -u cboard -f"
    echo "或: tail -f logs/app-$(date +%Y-%m-%d).log"
else
    echo "✗ 服务启动失败"
    echo "查看错误日志: sudo journalctl -u cboard -n 50"
    exit 1
fi
