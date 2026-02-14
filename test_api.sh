#!/bin/bash
# CBoard V2 API 功能测试脚本
# 用法: ./test_api.sh [BASE_URL]
# 默认: http://localhost:8000

set -e

BASE="${1:-http://localhost:8000}/api/v1"
PASS=0
FAIL=0
SKIP=0
TOKEN=""
ADMIN_TOKEN=""
ORDER_NO=""
SUB_URL=""
INVITE_ID=""
DEVICE_ID=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

check() {
  local name="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -q "$expected"; then
    echo -e "  ${GREEN}[PASS]${NC} $name"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}[FAIL]${NC} $name"
    echo -e "    Expected: $expected"
    echo -e "    Got: $(echo "$actual" | head -c 200)"
    FAIL=$((FAIL+1))
  fi
}

check_code() {
  local name="$1" response="$2" expected_code="${3:-0}"
  local code=$(echo "$response" | jq -r '.code // empty' 2>/dev/null)
  if [ "$code" = "$expected_code" ]; then
    echo -e "  ${GREEN}[PASS]${NC} $name"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}[FAIL]${NC} $name (code=$code)"
    echo -e "    Response: $(echo "$response" | head -c 300)"
    FAIL=$((FAIL+1))
  fi
}

check_http() {
  local name="$1" status="$2" expected="${3:-200}"
  if [ "$status" = "$expected" ]; then
    echo -e "  ${GREEN}[PASS]${NC} $name (HTTP $status)"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}[FAIL]${NC} $name (HTTP $status, expected $expected)"
    FAIL=$((FAIL+1))
  fi
}

skip() {
  echo -e "  ${YELLOW}[SKIP]${NC} $1 - $2"
  SKIP=$((SKIP+1))
}

api() {
  curl -s -w "\n%{http_code}" "$@" 2>/dev/null
}

api_json() {
  curl -s "$@" 2>/dev/null
}

# Generate unique test user
TS=$(date +%s)
TEST_EMAIL="test_${TS}@example.com"
TEST_PASS="TestPass123!"
TEST_USER="testuser_${TS}"

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  CBoard V2 API 功能测试${NC}"
echo -e "${CYAN}  Base URL: $BASE${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

# ==================== 1. 公开接口 ====================
echo -e "${CYAN}[1] 公开接口测试${NC}"

R=$(api_json "$BASE/config")
check_code "GET /config - 获取公开配置" "$R"

R=$(api_json "$BASE/packages")
check_code "GET /packages - 获取套餐列表" "$R"

R=$(api_json "$BASE/announcements")
check_code "GET /announcements - 获取公告" "$R"

R=$(api_json "$BASE/payment/methods")
check_code "GET /payment/methods - 获取支付方式" "$R"
PM_COUNT=$(echo "$R" | jq -r '.data.methods | length' 2>/dev/null)
BAL_ENABLED=$(echo "$R" | jq -r '.data.balance_enabled' 2>/dev/null)
echo -e "    支付方式数量: $PM_COUNT, 余额支付: $BAL_ENABLED"

echo ""

# ==================== 2. 注册/登录 ====================
echo -e "${CYAN}[2] 认证流程测试${NC}"

# Register (may need verification code depending on config)
R=$(api_json -X POST "$BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASS\",\"username\":\"$TEST_USER\"}")
REG_CODE=$(echo "$R" | jq -r '.code' 2>/dev/null)
if [ "$REG_CODE" = "0" ]; then
  check_code "POST /auth/register - 注册用户" "$R"
else
  echo -e "  ${YELLOW}[INFO]${NC} 注册返回: $(echo "$R" | jq -r '.message' 2>/dev/null) (可能需要验证码)"
fi

# Login
R=$(api_json -X POST "$BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASS\"}")
TOKEN=$(echo "$R" | jq -r '.data.access_token // empty' 2>/dev/null)
if [ -n "$TOKEN" ]; then
  check_code "POST /auth/login - 用户登录" "$R"
  echo -e "    Token: ${TOKEN:0:20}..."
else
  echo -e "  ${YELLOW}[INFO]${NC} 登录失败，尝试使用管理员账号"
  # Try admin login
  R=$(api_json -X POST "$BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@admin.com","password":"admin123"}')
  TOKEN=$(echo "$R" | jq -r '.data.access_token // empty' 2>/dev/null)
  if [ -n "$TOKEN" ]; then
    echo -e "  ${GREEN}[PASS]${NC} POST /auth/login - 管理员登录"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}[FAIL]${NC} 无法登录，后续测试将跳过需认证的接口"
    FAIL=$((FAIL+1))
  fi
fi

AUTH="-H \"Authorization: Bearer $TOKEN\""
echo ""

# ==================== 3. 用户接口 ====================
echo -e "${CYAN}[3] 用户接口测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/users/me" -H "Authorization: Bearer $TOKEN")
  check_code "GET /users/me - 获取当前用户" "$R"

  R=$(api_json "$BASE/users/dashboard-info" -H "Authorization: Bearer $TOKEN")
  check_code "GET /users/dashboard-info - 仪表盘信息" "$R"

  R=$(api_json "$BASE/users/login-history" -H "Authorization: Bearer $TOKEN")
  check_code "GET /users/login-history - 登录历史" "$R"
  LOGIN_COUNT=$(echo "$R" | jq -r '.data.items | length // .data | length' 2>/dev/null)
  echo -e "    登录记录数: $LOGIN_COUNT"

  R=$(api_json "$BASE/users/notification-settings" -H "Authorization: Bearer $TOKEN")
  check_code "GET /users/notification-settings - 通知设置" "$R"

  R=$(api_json "$BASE/users/privacy-settings" -H "Authorization: Bearer $TOKEN")
  check_code "GET /users/privacy-settings - 隐私设置" "$R"

  R=$(api_json "$BASE/users/devices" -H "Authorization: Bearer $TOKEN")
  check_code "GET /users/devices - 用户设备" "$R"
else
  skip "用户接口" "未登录"
fi

echo ""

# ==================== 4. 套餐购买流程 ====================
echo -e "${CYAN}[4] 套餐购买流程测试${NC}"

if [ -n "$TOKEN" ]; then
  # Get first package
  R=$(api_json "$BASE/packages")
  PKG_ID=$(echo "$R" | jq -r '.data[0].id // .data.items[0].id // empty' 2>/dev/null)
  PKG_NAME=$(echo "$R" | jq -r '.data[0].name // .data.items[0].name // empty' 2>/dev/null)

  if [ -n "$PKG_ID" ]; then
    echo -e "    使用套餐: $PKG_NAME (ID: $PKG_ID)"

    # Create order
    R=$(api_json -X POST "$BASE/orders" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"package_id\":$PKG_ID}")
    check_code "POST /orders - 创建订单" "$R"
    ORDER_NO=$(echo "$R" | jq -r '.data.order_no // empty' 2>/dev/null)
    ORDER_ID=$(echo "$R" | jq -r '.data.id // empty' 2>/dev/null)
    echo -e "    订单号: $ORDER_NO"

    # List orders
    R=$(api_json "$BASE/orders?page=1&page_size=10" -H "Authorization: Bearer $TOKEN")
    check_code "GET /orders - 订单列表" "$R"
    HAS_PKG_NAME=$(echo "$R" | jq -r '.data.items[0].package_name // empty' 2>/dev/null)
    if [ -n "$HAS_PKG_NAME" ]; then
      echo -e "  ${GREEN}[PASS]${NC} 订单包含 package_name 字段: $HAS_PKG_NAME"
      PASS=$((PASS+1))
    else
      echo -e "  ${RED}[FAIL]${NC} 订单缺少 package_name 字段"
      FAIL=$((FAIL+1))
    fi

    # Filter orders by status
    R=$(api_json "$BASE/orders?status=pending" -H "Authorization: Bearer $TOKEN")
    check_code "GET /orders?status=pending - 按状态筛选" "$R"

    # Get order status
    if [ -n "$ORDER_NO" ]; then
      R=$(api_json "$BASE/orders/$ORDER_NO/status" -H "Authorization: Bearer $TOKEN")
      check_code "GET /orders/:orderNo/status - 订单状态" "$R"
    fi

    # Try balance payment
    if [ -n "$ORDER_NO" ]; then
      R=$(api_json -X POST "$BASE/orders/$ORDER_NO/pay" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"payment_method":"balance"}')
      PAY_CODE=$(echo "$R" | jq -r '.code' 2>/dev/null)
      if [ "$PAY_CODE" = "0" ]; then
        echo -e "  ${GREEN}[PASS]${NC} POST /orders/:orderNo/pay - 余额支付成功"
        PASS=$((PASS+1))
      else
        PAY_MSG=$(echo "$R" | jq -r '.message' 2>/dev/null)
        echo -e "  ${YELLOW}[INFO]${NC} 余额支付: $PAY_MSG (余额不足是正常的)"
      fi
    fi

    # Test external payment creation
    if [ -n "$ORDER_ID" ] && [ "$PM_COUNT" != "0" ] && [ "$PM_COUNT" != "null" ]; then
      PM_ID=$(api_json "$BASE/payment/methods" | jq -r '.data.methods[0].id // empty' 2>/dev/null)
      if [ -n "$PM_ID" ]; then
        # Create a new order for external payment test
        R2=$(api_json -X POST "$BASE/orders" \
          -H "Authorization: Bearer $TOKEN" \
          -H "Content-Type: application/json" \
          -d "{\"package_id\":$PKG_ID}")
        ORDER_ID2=$(echo "$R2" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$ORDER_ID2" ]; then
          R=$(api_json -X POST "$BASE/payment" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "{\"order_id\":$ORDER_ID2,\"payment_method_id\":$PM_ID}")
          EPAY_CODE=$(echo "$R" | jq -r '.code' 2>/dev/null)
          if [ "$EPAY_CODE" = "0" ]; then
            PAYMENT_URL=$(echo "$R" | jq -r '.data.payment_url // empty' 2>/dev/null)
            echo -e "  ${GREEN}[PASS]${NC} POST /payment - 创建外部支付"
            PASS=$((PASS+1))
            if [ -n "$PAYMENT_URL" ]; then
              echo -e "    支付URL: ${PAYMENT_URL:0:60}..."
            fi
          else
            echo -e "  ${YELLOW}[INFO]${NC} 外部支付: $(echo "$R" | jq -r '.message' 2>/dev/null)"
          fi
        fi
      fi
    fi

    # Cancel remaining test order
    if [ -n "$ORDER_NO" ]; then
      R=$(api_json -X POST "$BASE/orders/$ORDER_NO/cancel" \
        -H "Authorization: Bearer $TOKEN")
      # May already be paid or cancelled, that's ok
    fi
  else
    skip "套餐购买" "无可用套餐"
  fi
else
  skip "套餐购买" "未登录"
fi

echo ""

# ==================== 5. 订阅接口 ====================
echo -e "${CYAN}[5] 订阅接口测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/subscriptions/user-subscription" -H "Authorization: Bearer $TOKEN")
  SUB_CODE=$(echo "$R" | jq -r '.code' 2>/dev/null)
  if [ "$SUB_CODE" = "0" ]; then
    check_code "GET /subscriptions/user-subscription - 获取订阅" "$R"
    SUB_URL=$(echo "$R" | jq -r '.data.subscription_url // empty' 2>/dev/null)
    UNIV_URL=$(echo "$R" | jq -r '.data.universal_url // empty' 2>/dev/null)
    CLASH_URL=$(echo "$R" | jq -r '.data.clash_url // empty' 2>/dev/null)
    echo -e "    订阅URL: ${SUB_URL:0:20}..."
    echo -e "    通用链接: ${UNIV_URL:0:40}..."
    echo -e "    Clash链接: ${CLASH_URL:0:40}..."
  else
    echo -e "  ${YELLOW}[INFO]${NC} 暂无订阅 (需先购买套餐)"
  fi

  R=$(api_json "$BASE/subscriptions/devices" -H "Authorization: Bearer $TOKEN")
  check_code "GET /subscriptions/devices - 订阅设备列表" "$R"

  # Test subscription link access
  if [ -n "$SUB_URL" ]; then
    RESP=$(api "$BASE/subscribe/$SUB_URL")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    check_http "GET /subscribe/:url - Clash订阅" "$HTTP_CODE"

    RESP=$(api "$BASE/subscribe/universal/$SUB_URL")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    check_http "GET /subscribe/universal/:url - 通用订阅" "$HTTP_CODE"
  fi

  # Convert to balance
  R=$(api_json -X POST "$BASE/subscriptions/convert-to-balance" -H "Authorization: Bearer $TOKEN")
  CVT_CODE=$(echo "$R" | jq -r '.code' 2>/dev/null)
  if [ "$CVT_CODE" = "0" ]; then
    echo -e "  ${GREEN}[PASS]${NC} POST /subscriptions/convert-to-balance - 转换余额"
    PASS=$((PASS+1))
    echo -e "    转换金额: $(echo "$R" | jq -r '.data.converted_amount' 2>/dev/null)"
  else
    echo -e "  ${YELLOW}[INFO]${NC} 转换余额: $(echo "$R" | jq -r '.message' 2>/dev/null)"
  fi

  # Send subscription email
  R=$(api_json -X POST "$BASE/subscriptions/send-subscription-email" -H "Authorization: Bearer $TOKEN")
  echo -e "  ${YELLOW}[INFO]${NC} 发送订阅邮件: $(echo "$R" | jq -r '.message' 2>/dev/null)"
else
  skip "订阅接口" "未登录"
fi

echo ""

# ==================== 6. 设备管理 ====================
echo -e "${CYAN}[6] 设备管理测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/devices" -H "Authorization: Bearer $TOKEN")
  check_code "GET /devices - 设备列表" "$R"
  DEVICE_ID=$(echo "$R" | jq -r '(.data[0].id // .data.items[0].id) // empty' 2>/dev/null)

  if [ -n "$DEVICE_ID" ] && [ "$DEVICE_ID" != "null" ]; then
    R=$(api_json -X DELETE "$BASE/devices/$DEVICE_ID" -H "Authorization: Bearer $TOKEN")
    check_code "DELETE /devices/:id - 删除设备" "$R"
  else
    echo -e "  ${YELLOW}[INFO]${NC} 无设备可删除"
  fi

  # Also test subscription device deletion
  R=$(api_json "$BASE/subscriptions/devices" -H "Authorization: Bearer $TOKEN")
  SUB_DEV_ID=$(echo "$R" | jq -r '(.data[0].id // .data.items[0].id) // empty' 2>/dev/null)
  if [ -n "$SUB_DEV_ID" ] && [ "$SUB_DEV_ID" != "null" ]; then
    R=$(api_json -X DELETE "$BASE/subscriptions/devices/$SUB_DEV_ID" -H "Authorization: Bearer $TOKEN")
    check_code "DELETE /subscriptions/devices/:id - 删除订阅设备" "$R"
  else
    echo -e "  ${YELLOW}[INFO]${NC} 无订阅设备可删除"
  fi
else
  skip "设备管理" "未登录"
fi

echo ""

# ==================== 7. 节点接口 ====================
echo -e "${CYAN}[7] 节点接口测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/nodes" -H "Authorization: Bearer $TOKEN")
  check_code "GET /nodes - 节点列表" "$R"
  NODE_COUNT=$(echo "$R" | jq -r '.data.items | length' 2>/dev/null)
  echo -e "    节点数量: $NODE_COUNT"

  # Check protocol field exists
  HAS_PROTOCOL=$(echo "$R" | jq -r '.data.items[0].protocol // empty' 2>/dev/null)
  if [ -n "$HAS_PROTOCOL" ]; then
    echo -e "  ${GREEN}[PASS]${NC} 节点包含 protocol 字段: $HAS_PROTOCOL"
    PASS=$((PASS+1))
  elif [ "$NODE_COUNT" = "0" ] || [ "$NODE_COUNT" = "null" ]; then
    echo -e "  ${YELLOW}[INFO]${NC} 无节点数据，跳过 protocol 字段检查"
  else
    echo -e "  ${RED}[FAIL]${NC} 节点缺少 protocol 字段"
    FAIL=$((FAIL+1))
  fi

  R=$(api_json "$BASE/nodes/stats" -H "Authorization: Bearer $TOKEN")
  check_code "GET /nodes/stats - 节点统计" "$R"
else
  skip "节点接口" "未登录"
fi

echo ""

# ==================== 8. 邀请码 ====================
echo -e "${CYAN}[8] 邀请码测试${NC}"

if [ -n "$TOKEN" ]; then
  # Create invite code
  R=$(api_json -X POST "$BASE/invites" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"max_uses":5,"expires_in_days":7,"inviter_reward":1,"invitee_reward":1}')
  check_code "POST /invites - 创建邀请码" "$R"
  INVITE_ID=$(echo "$R" | jq -r '.data.id // empty' 2>/dev/null)
  INVITE_CODE=$(echo "$R" | jq -r '.data.code // empty' 2>/dev/null)
  echo -e "    邀请码: $INVITE_CODE (ID: $INVITE_ID)"

  # List invite codes
  R=$(api_json "$BASE/invites" -H "Authorization: Bearer $TOKEN")
  check_code "GET /invites - 邀请码列表" "$R"

  # Get stats
  R=$(api_json "$BASE/invites/stats" -H "Authorization: Bearer $TOKEN")
  check_code "GET /invites/stats - 邀请统计" "$R"

  # Validate invite code
  if [ -n "$INVITE_CODE" ]; then
    R=$(api_json "$BASE/invites/validate/$INVITE_CODE")
    check_code "GET /invites/validate/:code - 验证邀请码" "$R"
  fi

  # Delete invite code
  if [ -n "$INVITE_ID" ]; then
    R=$(api_json -X DELETE "$BASE/invites/$INVITE_ID" -H "Authorization: Bearer $TOKEN")
    check_code "DELETE /invites/:id - 删除邀请码" "$R"
  fi
else
  skip "邀请码" "未登录"
fi

echo ""

# ==================== 9. 充值接口 ====================
echo -e "${CYAN}[9] 充值接口测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/recharge" -H "Authorization: Bearer $TOKEN")
  check_code "GET /recharge - 充值记录列表" "$R"

  R=$(api_json -X POST "$BASE/recharge" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"amount":10}')
  check_code "POST /recharge - 创建充值" "$R"
  RCH_ID=$(echo "$R" | jq -r '.data.id // empty' 2>/dev/null)

  if [ -n "$RCH_ID" ]; then
    R=$(api_json -X POST "$BASE/recharge/$RCH_ID/cancel" -H "Authorization: Bearer $TOKEN")
    check_code "POST /recharge/:id/cancel - 取消充值" "$R"
  fi
else
  skip "充值接口" "未登录"
fi

echo ""

# ==================== 10. 工单接口 ====================
echo -e "${CYAN}[10] 工单接口测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/tickets" -H "Authorization: Bearer $TOKEN")
  check_code "GET /tickets - 工单列表" "$R"

  R=$(api_json -X POST "$BASE/tickets" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"title":"测试工单","content":"这是一个测试工单","type":"technical"}')
  check_code "POST /tickets - 创建工单" "$R"
  TICKET_ID=$(echo "$R" | jq -r '.data.id // empty' 2>/dev/null)

  if [ -n "$TICKET_ID" ]; then
    R=$(api_json "$BASE/tickets/$TICKET_ID" -H "Authorization: Bearer $TOKEN")
    check_code "GET /tickets/:id - 工单详情" "$R"

    R=$(api_json -X POST "$BASE/tickets/$TICKET_ID/reply" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"content":"测试回复"}')
    check_code "POST /tickets/:id/reply - 回复工单" "$R"

    R=$(api_json -X PUT "$BASE/tickets/$TICKET_ID" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"status":"closed"}')
    check_code "PUT /tickets/:id - 关闭工单" "$R"
  fi
else
  skip "工单接口" "未登录"
fi

echo ""

# ==================== 11. 通知接口 ====================
echo -e "${CYAN}[11] 通知接口测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/notifications" -H "Authorization: Bearer $TOKEN")
  check_code "GET /notifications - 通知列表" "$R"

  R=$(api_json "$BASE/notifications/unread-count" -H "Authorization: Bearer $TOKEN")
  check_code "GET /notifications/unread-count - 未读数" "$R"
else
  skip "通知接口" "未登录"
fi

echo ""

# ==================== 12. 卡密兑换 ====================
echo -e "${CYAN}[12] 卡密兑换测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json "$BASE/redeem/history" -H "Authorization: Bearer $TOKEN")
  check_code "GET /redeem/history - 兑换历史" "$R"
else
  skip "卡密兑换" "未登录"
fi

echo ""

# ==================== 13. 密码修改 ====================
echo -e "${CYAN}[13] 用户设置测试${NC}"

if [ -n "$TOKEN" ]; then
  R=$(api_json -X PUT "$BASE/users/notification-settings" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"email_notifications":true}')
  check_code "PUT /users/notification-settings - 更新通知设置" "$R"

  R=$(api_json -X PUT "$BASE/users/privacy-settings" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"abnormal_login_alert_enabled":true}')
  check_code "PUT /users/privacy-settings - 更新隐私设置" "$R"
else
  skip "用户设置" "未登录"
fi

echo ""

# ==================== 结果汇总 ====================
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  测试结果汇总${NC}"
echo -e "${CYAN}========================================${NC}"
TOTAL=$((PASS+FAIL+SKIP))
echo -e "  总计: $TOTAL"
echo -e "  ${GREEN}通过: $PASS${NC}"
echo -e "  ${RED}失败: $FAIL${NC}"
echo -e "  ${YELLOW}跳过: $SKIP${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
  echo -e "${GREEN}所有测试通过!${NC}"
  exit 0
else
  echo -e "${RED}有 $FAIL 个测试失败${NC}"
  exit 1
fi
