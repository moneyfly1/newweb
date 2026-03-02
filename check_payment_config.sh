#!/bin/bash
# 支付配置检查脚本

echo "========================================="
echo "支付配置检查"
echo "========================================="
echo ""

DB_FILE="cboard.db"

echo "1. 检查系统配置..."
echo "-----------------------------------"
sqlite3 $DB_FILE "SELECT key, value FROM system_configs WHERE key IN ('site_url', 'domain_name');"
echo ""

echo "2. 检查支付宝配置..."
echo "-----------------------------------"
echo "从 system_configs 表检查："
sqlite3 $DB_FILE "SELECT
  CASE WHEN value IS NULL OR value = '' THEN '❌ 未配置' ELSE '✓ 已配置' END as status,
  key
FROM system_configs WHERE key IN ('pay_alipay_app_id', 'pay_alipay_private_key', 'pay_alipay_public_key', 'pay_alipay_notify_url')
ORDER BY key;"
echo ""
echo "详细配置："
sqlite3 $DB_FILE "SELECT key,
  CASE
    WHEN key LIKE '%private_key%' THEN SUBSTR(value, 1, 50) || '...(长度:' || LENGTH(value) || ')'
    WHEN key LIKE '%public_key%' THEN SUBSTR(value, 1, 50) || '...(长度:' || LENGTH(value) || ')'
    ELSE value
  END as value
FROM system_configs WHERE key LIKE 'pay_alipay%' ORDER BY key;"
echo ""

echo "3. 检查最近的订单..."
echo "-----------------------------------"
sqlite3 $DB_FILE "SELECT id, order_no, amount, status, payment_method_name, created_at FROM orders ORDER BY created_at DESC LIMIT 3;"
echo ""

echo "4. 检查最近的支付事务..."
echo "-----------------------------------"
sqlite3 $DB_FILE "SELECT id, transaction_id, order_id, amount, status, created_at FROM payment_transactions ORDER BY created_at DESC LIMIT 3;"
echo ""

echo "5. 检查支付回调记录..."
echo "-----------------------------------"
CALLBACK_COUNT=$(sqlite3 $DB_FILE "SELECT COUNT(*) FROM payment_callbacks;")
echo "回调记录总数: $CALLBACK_COUNT"
if [ "$CALLBACK_COUNT" -gt 0 ]; then
  sqlite3 $DB_FILE "SELECT id, callback_type, processed, created_at FROM payment_callbacks ORDER BY created_at DESC LIMIT 3;"
else
  echo "❌ 没有收到任何支付回调"
fi
echo ""

echo "========================================="
echo "问题诊断"
echo "========================================="

SITE_URL=$(sqlite3 $DB_FILE "SELECT value FROM system_configs WHERE key = 'site_url';")
if [[ "$SITE_URL" == *"localhost"* ]] || [[ "$SITE_URL" == *"127.0.0.1"* ]]; then
  echo "❌ site_url 配置为本地地址，支付宝无法回调"
  echo "   请在系统设置中修改为实际域名（如 https://go.moneyfly.top）"
fi

APP_ID=$(sqlite3 $DB_FILE "SELECT value FROM system_configs WHERE key = 'pay_alipay_app_id';")
if [ -z "$APP_ID" ]; then
  echo "❌ 支付宝 AppID 未配置"
  echo "   请在系统设置中配置 pay_alipay_app_id"
else
  echo "✓ 支付宝配置完整"
fi

echo ""
echo "========================================="
echo "修复建议"
echo "========================================="
echo "1. 登录管理后台 -> 系统管理 -> 系统设置"
echo "   修改 site_url 为实际域名（如 https://go.moneyfly.top）"
echo ""
echo "2. 登录管理后台 -> 订单管理 -> 支付方式"
echo "   配置支付宝的 AppID、应用私钥、支付宝公钥"
echo ""
echo "3. 重启服务: systemctl restart cboard"
echo ""
