#!/bin/bash
# 测试支付宝配置和回调地址

echo "========================================="
echo "支付宝配置测试"
echo "========================================="
echo ""

DB_FILE="cboard.db"

echo "1. 检查支付宝配置..."
echo "-----------------------------------"
sqlite3 $DB_FILE "SELECT key, value FROM system_configs WHERE key IN ('pay_alipay_app_id', 'pay_alipay_notify_url', 'site_url') ORDER BY key;"
echo ""

echo "2. 检查最近的订单..."
echo "-----------------------------------"
sqlite3 $DB_FILE "SELECT id, order_no, user_id, amount, status, payment_method_name, created_at FROM orders ORDER BY id DESC LIMIT 3;"
echo ""

echo "3. 检查支付事务..."
echo "-----------------------------------"
sqlite3 $DB_FILE "SELECT id, transaction_id, order_id, amount, status, created_at FROM payment_transactions ORDER BY id DESC LIMIT 3;"
echo ""

echo "4. 检查回调记录..."
echo "-----------------------------------"
CALLBACK_COUNT=$(sqlite3 $DB_FILE "SELECT COUNT(*) FROM payment_callbacks;")
echo "回调记录总数: $CALLBACK_COUNT"
if [ "$CALLBACK_COUNT" -gt 0 ]; then
    echo ""
    echo "最近的回调："
    sqlite3 $DB_FILE "SELECT id, callback_type, processed, processing_result, created_at FROM payment_callbacks ORDER BY id DESC LIMIT 3;"
else
    echo "❌ 没有收到任何回调"
fi
echo ""

echo "5. 测试回调地址可访问性..."
echo "-----------------------------------"
NOTIFY_URL=$(sqlite3 $DB_FILE "SELECT value FROM system_configs WHERE key = 'pay_alipay_notify_url';")
if [ -n "$NOTIFY_URL" ]; then
    echo "配置的回调地址: $NOTIFY_URL"
    echo ""
    echo "测试 POST 请求（模拟支付宝回调）："
    curl -X POST "$NOTIFY_URL" \
        -d "out_trade_no=TEST123" \
        -d "trade_no=TEST456" \
        -d "trade_status=TRADE_SUCCESS" \
        -d "total_amount=0.01" \
        -d "sign=test_sign" \
        -w "\nHTTP Status: %{http_code}\n" \
        -s -o /dev/null
else
    echo "❌ 回调地址未配置"
fi
echo ""

echo "========================================="
echo "诊断建议"
echo "========================================="

# 检查服务是否运行
if systemctl is-active --quiet cboard; then
    echo "✓ cboard 服务正在运行"
    echo ""
    echo "查看实时日志："
    echo "  journalctl -u cboard -f"
else
    echo "❌ cboard 服务未运行"
    echo ""
    echo "启动服务："
    echo "  systemctl start cboard"
fi
echo ""

echo "如果支付后没有回调："
echo "1. 检查服务器防火墙是否开放 80/443 端口"
echo "2. 检查 Nginx 是否正确转发到后端"
echo "3. 查看支付宝开放平台的回调日志"
echo "4. 确认回调地址可以从外网访问"
echo ""
