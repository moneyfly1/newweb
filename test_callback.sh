#!/bin/bash

# 测试支付宝回调
# 使用订单号 ORD17725062420taFVX

ORDER_NO="ORD17725062420taFVX"
TRADE_NO="2026030322001234567890123456"
TOTAL_AMOUNT="0.05"

# 注意：这个测试会失败，因为签名不正确
# 但可以验证服务器是否能接收到回调

curl -X POST "https://go.moneyfly.top/api/v1/payment/notify/alipay" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "out_trade_no=${ORDER_NO}" \
  -d "trade_no=${TRADE_NO}" \
  -d "trade_status=TRADE_SUCCESS" \
  -d "total_amount=${TOTAL_AMOUNT}" \
  -d "sign=test_signature" \
  -v

echo ""
echo "如果返回 'verify fail'，说明服务器收到了回调但签名验证失败（正常）"
echo "如果返回 404，说明路由配置有问题"
echo "如果连接失败，说明服务器无法访问"
