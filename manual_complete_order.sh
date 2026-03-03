#!/bin/bash

# 手动完成订单 ORD17725062420taFVX 并激活订阅

ORDER_NO="ORD17725062420taFVX"
ORDER_ID=270
TRANSACTION_ID=23
USER_ID=1  # 请根据实际情况修改

echo "========== 手动完成订单 =========="
echo "订单号: $ORDER_NO"
echo "订单ID: $ORDER_ID"
echo ""

# 1. 更新订单状态
echo "1. 更新订单状态为 paid..."
sqlite3 /Users/apple/v2/cboard.db "UPDATE orders SET status='paid', payment_time=datetime('now'), payment_method_name='支付宝', payment_transaction_id='$TRANSACTION_ID' WHERE id=$ORDER_ID;"

# 2. 更新支付交易状态
echo "2. 更新支付交易状态..."
sqlite3 /Users/apple/v2/cboard.db "UPDATE payment_transactions SET status='paid', external_transaction_id='MANUAL_$(date +%Y%m%d%H%M%S)', transaction_id='$ORDER_NO' WHERE id=$TRANSACTION_ID;"

# 3. 查看订单信息
echo ""
echo "3. 查看更新后的订单信息:"
sqlite3 /Users/apple/v2/cboard.db "SELECT id, order_no, status, payment_time, payment_method_name FROM orders WHERE id=$ORDER_ID;"

# 4. 查看支付交易信息
echo ""
echo "4. 查看更新后的支付交易信息:"
sqlite3 /Users/apple/v2/cboard.db "SELECT id, transaction_id, status, external_transaction_id FROM payment_transactions WHERE id=$TRANSACTION_ID;"

echo ""
echo "========== 完成 =========="
echo ""
echo "⚠️  注意：订单状态已更新，但订阅尚未激活"
echo "请执行以下步骤激活订阅："
echo ""
echo "1. 重启服务器："
echo "   pkill -f cboard"
echo "   cd /Users/apple/v2"
echo "   ./cboard-server"
echo ""
echo "2. 访问以下URL手动触发订阅激活："
echo "   curl -X POST http://localhost:8000/api/v1/admin/orders/$ORDER_ID/activate \\"
echo "     -H 'Authorization: Bearer YOUR_ADMIN_TOKEN'"
echo ""
echo "或者在管理后台找到该订单，点击'激活订阅'按钮"
