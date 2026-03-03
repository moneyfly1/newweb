#!/bin/bash

# 批量完成待处理订单

DB="/Users/apple/v2/cboard.db"

echo "=========================================="
echo "批量完成待处理订单"
echo "=========================================="
echo ""

# 获取待处理订单列表
echo "查找待处理订单..."
ORDERS=$(sqlite3 $DB "SELECT id, order_no, user_id, package_id FROM orders WHERE status='pending' AND datetime(created_at, 'localtime') > datetime('now', 'localtime', '-30 minutes') ORDER BY id DESC;")

if [ -z "$ORDERS" ]; then
    echo "没有待处理的订单"
    exit 0
fi

echo "找到以下待处理订单："
echo "$ORDERS"
echo ""

# 处理每个订单
echo "$ORDERS" | while IFS='|' read -r order_id order_no user_id package_id; do
    echo "----------------------------------------"
    echo "处理订单: $order_no (ID: $order_id)"

    # 获取套餐信息
    PACKAGE_INFO=$(sqlite3 $DB "SELECT device_limit, duration_days FROM packages WHERE id=$package_id;")
    device_limit=$(echo $PACKAGE_INFO | cut -d'|' -f1)
    duration_days=$(echo $PACKAGE_INFO | cut -d'|' -f2)

    echo "  用户ID: $user_id"
    echo "  套餐ID: $package_id"
    echo "  设备限制: $device_limit"
    echo "  时长: $duration_days 天"

    # 1. 更新订单状态
    sqlite3 $DB "UPDATE orders SET status='paid', payment_time=datetime('now'), payment_method_name='支付宝-手动' WHERE id=$order_id;"

    # 2. 更新支付交易
    sqlite3 $DB "UPDATE payment_transactions SET status='paid', external_transaction_id='MANUAL_$(date +%Y%m%d%H%M%S)' WHERE order_id=$order_id;"

    # 3. 检查用户是否已有订阅
    SUB_EXISTS=$(sqlite3 $DB "SELECT COUNT(*) FROM subscriptions WHERE user_id=$user_id;")

    if [ "$SUB_EXISTS" -eq "0" ]; then
        # 创建新订阅
        echo "  创建新订阅..."
        SUB_URL=$(openssl rand -hex 16)
        sqlite3 $DB "INSERT INTO subscriptions (user_id, subscription_url, device_limit, current_devices, is_active, status, expire_time, package_id, created_at, updated_at) VALUES ($user_id, '$SUB_URL', $device_limit, 0, 1, 'active', datetime('now', '+$duration_days days'), $package_id, datetime('now'), datetime('now'));"
    else
        # 更新现有订阅
        echo "  更新现有订阅..."
        sqlite3 $DB "UPDATE subscriptions SET device_limit=$device_limit, expire_time=datetime(expire_time, '+$duration_days days'), is_active=1, status='active', package_id=$package_id WHERE user_id=$user_id;"
    fi

    echo "  ✓ 订单已完成"
done

echo ""
echo "=========================================="
echo "处理完成！"
echo "=========================================="
echo ""

# 显示结果
echo "已完成订单："
sqlite3 $DB "SELECT id, order_no, status, payment_time FROM orders WHERE status='paid' AND datetime(payment_time, 'localtime') > datetime('now', 'localtime', '-5 minutes');"

echo ""
echo "活跃订阅："
sqlite3 $DB "SELECT id, user_id, device_limit, datetime(expire_time, 'localtime') as expire_time, status FROM subscriptions WHERE is_active=1;"
