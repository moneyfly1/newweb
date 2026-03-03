#!/bin/bash

# 完整的手动订单处理脚本
# 订单号: ORD17725062420taFVX

DB="/Users/apple/v2/cboard.db"

echo "=========================================="
echo "手动处理订单: ORD17725062420taFVX"
echo "=========================================="
echo ""

# 1. 更新订单状态
echo "步骤 1: 更新订单状态为 paid"
sqlite3 $DB <<EOF
UPDATE orders
SET status='paid',
    payment_time=datetime('now'),
    payment_method_name='支付宝',
    payment_transaction_id='23'
WHERE id=270;
EOF
echo "✓ 订单状态已更新"

# 2. 更新支付交易
echo ""
echo "步骤 2: 更新支付交易状态"
sqlite3 $DB <<EOF
UPDATE payment_transactions
SET status='paid',
    transaction_id='ORD17725062420taFVX',
    external_transaction_id='MANUAL_$(date +%Y%m%d%H%M%S)'
WHERE id=23;
EOF
echo "✓ 支付交易已更新"

# 3. 更新订阅（续期30天，设备限制改为3）
echo ""
echo "步骤 3: 更新订阅（续期30天，设备限制改为3）"
sqlite3 $DB <<EOF
UPDATE subscriptions
SET device_limit=3,
    expire_time=datetime(expire_time, '+30 days'),
    is_active=1,
    status='active',
    package_id=4
WHERE user_id=1;
EOF
echo "✓ 订阅已更新"

# 4. 显示结果
echo ""
echo "=========================================="
echo "处理完成！查看结果："
echo "=========================================="

echo ""
echo "订单信息:"
sqlite3 $DB "SELECT id, order_no, status, payment_time, payment_method_name FROM orders WHERE id=270;"

echo ""
echo "支付交易:"
sqlite3 $DB "SELECT id, transaction_id, status, external_transaction_id FROM payment_transactions WHERE id=23;"

echo ""
echo "订阅信息:"
sqlite3 $DB "SELECT id, device_limit, datetime(expire_time, 'localtime') as expire_time, is_active, status FROM subscriptions WHERE user_id=1;"

echo ""
echo "=========================================="
echo "✅ 订单已完成，订阅已激活！"
echo "=========================================="
echo ""
echo "现在可以刷新页面查看订单状态和订阅信息"
