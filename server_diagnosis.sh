#!/bin/bash

# 服务器诊断脚本 - 请在服务器上运行

echo "=========================================="
echo "支付回调诊断 - 服务器端"
echo "=========================================="
echo ""

# 1. 检查最近的订单
echo "1. 最近的订单（最近30分钟）："
echo "----------------------------------------"
sqlite3 /path/to/cboard.db "SELECT id, order_no, status, datetime(created_at, 'localtime') as created FROM orders WHERE datetime(created_at, 'localtime') > datetime('now', 'localtime', '-30 minutes') ORDER BY id DESC LIMIT 5;"

echo ""
echo "2. 支付交易记录："
echo "----------------------------------------"
sqlite3 /path/to/cboard.db "SELECT pt.id, pt.order_id, pt.transaction_id, pt.status, pt.external_transaction_id FROM payment_transactions pt WHERE pt.order_id IN (SELECT id FROM orders WHERE datetime(created_at, 'localtime') > datetime('now', 'localtime', '-30 minutes')) ORDER BY pt.id DESC;"

echo ""
echo "3. 支付回调记录："
echo "----------------------------------------"
sqlite3 /path/to/cboard.db "SELECT id, callback_type, processed, processing_result, datetime(created_at, 'localtime') as created FROM payment_callbacks ORDER BY id DESC LIMIT 10;"

echo ""
echo "4. 检查回调地址配置："
echo "----------------------------------------"
sqlite3 /path/to/cboard.db "SELECT key, value FROM system_configs WHERE key IN ('pay_alipay_notify_url', 'pay_alipay_return_url', 'site_url');"

echo ""
echo "5. 测试回调端点："
echo "----------------------------------------"
SITE_URL=$(sqlite3 /path/to/cboard.db "SELECT value FROM system_configs WHERE key='site_url';")
echo "回调地址: ${SITE_URL}/api/v1/payment/notify/alipay"
curl -X POST "${SITE_URL}/api/v1/payment/notify/alipay" -d "test=1" -v 2>&1 | grep -E "HTTP|verify"

echo ""
echo "6. 检查服务器进程："
echo "----------------------------------------"
ps aux | grep cboard | grep -v grep

echo ""
echo "7. 查看最近的服务器日志（如果有）："
echo "----------------------------------------"
if [ -f "/var/log/cboard/app.log" ]; then
    tail -50 /var/log/cboard/app.log | grep -E "alipay|payment|回调"
elif [ -f "./logs/app.log" ]; then
    tail -50 ./logs/app.log | grep -E "alipay|payment|回调"
else
    echo "未找到日志文件"
fi

echo ""
echo "=========================================="
echo "请将以上输出发送给我进行分析"
echo "=========================================="
