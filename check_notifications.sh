#!/bin/bash
# 通知功能快速测试脚本

DB_FILE="cboard.db"

echo "=========================================="
echo "通知功能配置检查"
echo "=========================================="
echo ""

# 检查数据库文件
if [ ! -f "$DB_FILE" ]; then
    echo "❌ 数据库文件不存在: $DB_FILE"
    exit 1
fi

echo "📋 1. 检查管理员通知开关"
echo "----------------------------------------"
sqlite3 "$DB_FILE" <<EOF
.mode column
.headers on
SELECT key, value FROM system_configs
WHERE key IN (
    'notify_new_order',
    'notify_payment_success',
    'notify_recharge_success',
    'notify_new_ticket',
    'notify_new_user',
    'notify_subscription_reset',
    'notify_abnormal_login'
)
ORDER BY key;
EOF
echo ""

echo "📋 2. 检查通知渠道配置"
echo "----------------------------------------"
sqlite3 "$DB_FILE" <<EOF
.mode column
.headers on
SELECT key,
    CASE
        WHEN key LIKE '%token%' OR key LIKE '%key%' THEN '***已配置***'
        ELSE value
    END as value
FROM system_configs
WHERE key IN (
    'notify_email_enabled',
    'notify_admin_email',
    'notify_telegram_enabled',
    'notify_telegram_bot_token',
    'notify_telegram_chat_id',
    'notify_bark_enabled',
    'notify_bark_server',
    'notify_bark_device_key'
)
ORDER BY key;
EOF
echo ""

echo "📋 3. 检查用户通知开关"
echo "----------------------------------------"
sqlite3 "$DB_FILE" <<EOF
.mode column
.headers on
SELECT key, value FROM system_configs
WHERE key IN (
    'user_notify_welcome',
    'user_notify_payment',
    'user_notify_expiry',
    'user_notify_expired',
    'user_notify_reset',
    'user_notify_account_status'
)
ORDER BY key;
EOF
echo ""

echo "📋 4. 检查用户个人通知设置（前 5 个用户）"
echo "----------------------------------------"
sqlite3 "$DB_FILE" <<EOF
.mode column
.headers on
SELECT
    id,
    username,
    email_notifications as email_notify,
    notify_order,
    notify_expiry,
    notify_subscription as notify_sub,
    abnormal_login_alert_enabled as abnormal_login
FROM users
ORDER BY id
LIMIT 5;
EOF
echo ""

echo "📋 5. 检查最近的邮件队列"
echo "----------------------------------------"
sqlite3 "$DB_FILE" <<EOF
.mode column
.headers on
SELECT
    id,
    to_email,
    subject,
    status,
    email_type,
    datetime(created_at) as created
FROM email_queue
ORDER BY created_at DESC
LIMIT 10;
EOF
echo ""

echo "📋 6. 检查通知相关系统日志"
echo "----------------------------------------"
sqlite3 "$DB_FILE" <<EOF
.mode column
.headers on
SELECT
    level,
    module,
    message,
    datetime(created_at) as created
FROM system_logs
WHERE module = 'notify'
ORDER BY created_at DESC
LIMIT 10;
EOF
echo ""

echo "=========================================="
echo "✅ 检查完成"
echo "=========================================="
echo ""
echo "💡 提示："
echo "1. 如果开关值为空，表示未配置（默认关闭）"
echo "2. 开关值为 'true' 或 '1' 表示开启"
echo "3. 开关值为 'false' 或 '0' 表示关闭"
echo "4. 测试 Telegram 通知：后台设置 → 通知设置 → 测试 Telegram"
echo ""
