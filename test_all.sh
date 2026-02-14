#!/bin/bash
# ============================================================
# CBoard v2 全功能 API 测试脚本 (完整版)
# 覆盖所有后端 API 端点 + 前端构建 + 后端编译
# ============================================================

set -e

BASE="http://localhost:9000/api/v1"
PASS=0
FAIL=0
SKIP=0
ERRORS=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ============================================================
# Helper functions
# ============================================================
assert_status() {
  local name="$1" expected="$2" actual="$3"
  if [ "$actual" -eq "$expected" ]; then
    echo -e "  ${GREEN}✓${NC} $name (HTTP $actual)"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} $name (期望 $expected, 实际 $actual)"
    FAIL=$((FAIL+1))
    ERRORS="${ERRORS}\n  ✗ $name (期望 $expected, 实际 $actual)"
  fi
}

assert_status_in() {
  local name="$1" actual="$2"
  shift 2
  local expected_list=("$@")
  for e in "${expected_list[@]}"; do
    if [ "$actual" -eq "$e" ]; then
      echo -e "  ${GREEN}✓${NC} $name (HTTP $actual)"
      PASS=$((PASS+1))
      return
    fi
  done
  echo -e "  ${RED}✗${NC} $name (期望 ${expected_list[*]}, 实际 $actual)"
  FAIL=$((FAIL+1))
  ERRORS="${ERRORS}\n  ✗ $name (期望 ${expected_list[*]}, 实际 $actual)"
}

assert_code0() {
  local name="$1" body="$2"
  local code=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code','?'))" 2>/dev/null)
  if [ "$code" = "0" ]; then
    echo -e "  ${GREEN}✓${NC} $name (code=0)"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} $name (code=$code)"
    FAIL=$((FAIL+1))
    ERRORS="${ERRORS}\n  ✗ $name (code=$code)"
  fi
}

assert_json_field() {
  local name="$1" body="$2" field="$3"
  local val=$(echo "$body" | python3 -c "
import sys,json
d=json.load(sys.stdin)
keys='$field'.split('.')
for k in keys:
    if isinstance(d,dict): d=d.get(k)
    else: d=None
print('__NONE__' if d is None else d)
" 2>/dev/null)
  if [ "$val" != "__NONE__" ] && [ -n "$val" ]; then
    echo -e "  ${GREEN}✓${NC} $name ($field=$val)"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} $name ($field 缺失)"
    FAIL=$((FAIL+1))
    ERRORS="${ERRORS}\n  ✗ $name ($field 缺失)"
  fi
}

skip_test() {
  echo -e "  ${YELLOW}⊘${NC} $1 (跳过)"
  SKIP=$((SKIP+1))
}

section() {
  echo ""
  echo -e "${CYAN}━━━ $1 ━━━${NC}"
}

# GET/POST/PUT/DELETE helpers
api_get() {
  curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" "$BASE$1" 2>/dev/null
}
api_post() {
  curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "$BASE$1" 2>/dev/null
}
api_put() {
  curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -X PUT -d "$2" "$BASE$1" 2>/dev/null
}
api_delete() {
  curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOKEN" "$BASE$1" -X DELETE 2>/dev/null
}
api_post_noauth() {
  curl -s -w "\n%{http_code}" -H "Content-Type: application/json" -d "$2" "$BASE$1" 2>/dev/null
}

split_response() {
  local resp="$1"
  BODY=$(echo "$resp" | sed '$d')
  STATUS=$(echo "$resp" | tail -1)
}

# ============================================================
echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   CBoard v2 全功能测试 (完整版)          ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"

# ============================================================
# 0. 环境检查
# ============================================================
section "0. 环境检查"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/config" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "000" ]; then
  echo -e "${RED}后端未运行 (localhost:9000)，请先启动后端${NC}"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} 后端运行中"

# ============================================================
# 1. 注册模块
# ============================================================
section "1. 注册模块 (Register)"

# 1.1 正常注册
REG_EMAIL="testreg_$(date +%s)@test.com"
REG_USER="testreg_$$"
RESP=$(api_post_noauth "/auth/register" "{\"username\":\"$REG_USER\",\"email\":\"$REG_EMAIL\",\"password\":\"Test123456\"}")
split_response "$RESP"
assert_status_in "注册新用户" "$STATUS" 200 429
if [ "$STATUS" -eq 200 ]; then
  assert_json_field "注册返回 access_token" "$BODY" "data.access_token"
  assert_json_field "注册返回 user.email" "$BODY" "data.user.email"
  REG_TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null)
  REG_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['user']['id'])" 2>/dev/null)
fi

# 1.2 重复注册
RESP=$(api_post_noauth "/auth/register" "{\"username\":\"$REG_USER\",\"email\":\"$REG_EMAIL\",\"password\":\"Test123456\"}")
split_response "$RESP"
assert_status_in "重复注册应拒绝" "$STATUS" 400 409 429
# PLACEHOLDER_REGISTER_MORE

# 1.3 缺少用户名
RESP=$(api_post_noauth "/auth/register" "{\"email\":\"bad@test.com\",\"password\":\"Test123456\"}")
split_response "$RESP"
assert_status_in "注册缺少用户名" "$STATUS" 400 429

# 1.4 密码太短
RESP=$(api_post_noauth "/auth/register" "{\"username\":\"shortpw\",\"email\":\"shortpw@test.com\",\"password\":\"12\"}")
split_response "$RESP"
assert_status_in "注册密码太短" "$STATUS" 400 429

# 1.5 带邀请码注册 (无效码)
RESP=$(api_post_noauth "/auth/register" "{\"username\":\"invtest_$$\",\"email\":\"invtest_$$@test.com\",\"password\":\"Test123456\",\"invite_code\":\"INVALID_CODE\"}")
split_response "$RESP"
assert_status_in "无效邀请码注册" "$STATUS" 400 429

# ============================================================
# 2. 登录模块
# ============================================================
section "2. 登录模块 (Login)"

# 2.1 正确凭据
RESP=$(api_post_noauth "/auth/login" '{"email":"admin@example.com","password":"admin123"}')
split_response "$RESP"
assert_status "登录 (正确凭据)" 200 "$STATUS"
assert_json_field "登录返回 access_token" "$BODY" "data.access_token"
assert_json_field "登录返回 user.is_admin" "$BODY" "data.user.is_admin"
assert_json_field "登录返回 user.email" "$BODY" "data.user.email"
assert_json_field "登录返回 refresh_token" "$BODY" "data.refresh_token"

TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null)
REFRESH=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['refresh_token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}无法获取 token，终止测试${NC}"
  exit 1
fi

# 2.2 错误密码
RESP=$(api_post_noauth "/auth/login" '{"email":"admin@example.com","password":"wrong"}')
split_response "$RESP"
assert_status_in "登录 (错误密码)" "$STATUS" 401 429

# 2.3 缺少参数
RESP=$(api_post_noauth "/auth/login" '{"email":"admin@example.com"}')
split_response "$RESP"
assert_status_in "登录 (缺少密码)" "$STATUS" 400 429

# 2.4 不存在的用户
RESP=$(api_post_noauth "/auth/login" '{"email":"nonexist@nowhere.com","password":"whatever"}')
split_response "$RESP"
assert_status_in "登录 (不存在用户)" "$STATUS" 401 429

# 2.5 刷新 Token
RESP=$(api_post_noauth "/auth/refresh" "{\"refresh_token\":\"$REFRESH\"}")
split_response "$RESP"
assert_status "刷新 Token" 200 "$STATUS"
assert_json_field "刷新返回新 access_token" "$BODY" "data.access_token"
NEW_TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null)
[ -n "$NEW_TOKEN" ] && TOKEN="$NEW_TOKEN"

# 2.6 无效 refresh token
RESP=$(api_post_noauth "/auth/refresh" '{"refresh_token":"invalid_token_xxx"}')
split_response "$RESP"
assert_status "无效 refresh token" 401 "$STATUS"

# 2.7 无 Token 访问受保护路由
RESP=$(curl -s -w "\n%{http_code}" "$BASE/users/me" 2>/dev/null)
split_response "$RESP"
assert_status "无 Token 访问应返回 401" 401 "$STATUS"

# ============================================================
# 3. 忘记密码 / 重置密码
# ============================================================
section "3. 忘记密码 / 重置密码"

# 3.1 发送忘记密码邮件 (不管邮箱是否存在都返回200)
RESP=$(api_post_noauth "/auth/forgot-password" '{"email":"admin@example.com"}')
split_response "$RESP"
assert_status_in "忘记密码 (存在邮箱)" "$STATUS" 200 429

RESP=$(api_post_noauth "/auth/forgot-password" '{"email":"nonexist@nowhere.com"}')
split_response "$RESP"
assert_status_in "忘记密码 (不存在邮箱)" "$STATUS" 200 429

# 3.2 重置密码 (无效验证码)
RESP=$(api_post_noauth "/auth/reset-password" '{"email":"admin@example.com","code":"000000","password":"NewPass123"}')
split_response "$RESP"
assert_status_in "重置密码 (无效验证码)" "$STATUS" 400 401

# 3.3 发送验证码
RESP=$(api_post_noauth "/auth/verification/send" '{"email":"admin@example.com","purpose":"register"}')
split_response "$RESP"
assert_status_in "发送验证码" "$STATUS" 200 429 500

# 3.4 验证码验证 (无效)
RESP=$(api_post_noauth "/auth/verification/verify" '{"email":"admin@example.com","code":"000000"}')
split_response "$RESP"
assert_status_in "验证码验证 (无效)" "$STATUS" 400 401

# ============================================================
# 4. 用户资料模块
# ============================================================
section "4. 用户资料模块 (User Profile)"

RESP=$(api_get "/users/me")
split_response "$RESP"
assert_status "获取当前用户" 200 "$STATUS"
assert_code0 "当前用户 code=0" "$BODY"
assert_json_field "用户有 email" "$BODY" "data.email"
assert_json_field "用户有 username" "$BODY" "data.username"

# 4.2 更新用户资料
RESP=$(api_put "/users/me" '{"nickname":"测试昵称","avatar":"https://example.com/avatar.png"}')
split_response "$RESP"
assert_status "更新用户资料" 200 "$STATUS"

# 4.3 修改密码 (错误旧密码)
RESP=$(api_post "/users/change-password" '{"old_password":"wrongold","new_password":"NewPass123"}')
split_response "$RESP"
assert_status_in "修改密码 (错误旧密码)" "$STATUS" 400 401

# 4.4 修改密码 (新密码太短)
RESP=$(api_post "/users/change-password" '{"old_password":"admin123","new_password":"12"}')
split_response "$RESP"
assert_status "修改密码 (新密码太短)" 400 "$STATUS"

# 4.5 修改密码 (正确) 然后改回来
RESP=$(api_post "/users/change-password" '{"old_password":"admin123","new_password":"Admin1234"}')
split_response "$RESP"
assert_status "修改密码 (正确)" 200 "$STATUS"
# 改回原密码
RESP=$(api_post "/users/change-password" '{"old_password":"Admin1234","new_password":"admin123"}')
split_response "$RESP"
assert_status "恢复原密码" 200 "$STATUS"

# 4.6 更新偏好设置
RESP=$(api_put "/users/preferences" '{"theme":"dark","language":"zh-CN","timezone":"Asia/Shanghai"}')
split_response "$RESP"
assert_status "更新偏好设置" 200 "$STATUS"

# 4.7 获取/更新通知设置
RESP=$(api_get "/users/notification-settings")
split_response "$RESP"
assert_status "获取通知设置" 200 "$STATUS"

RESP=$(api_put "/users/notification-settings" '{"email_notifications":true,"abnormal_login_alert_enabled":true,"push_notifications":false}')
split_response "$RESP"
assert_status "更新通知设置" 200 "$STATUS"

# 4.8 获取/更新隐私设置
RESP=$(api_get "/users/privacy-settings")
split_response "$RESP"
assert_status "获取隐私设置" 200 "$STATUS"

RESP=$(api_put "/users/privacy-settings" '{"data_sharing":false,"analytics":true}')
split_response "$RESP"
assert_status "更新隐私设置" 200 "$STATUS"

# 4.9 仪表盘信息
RESP=$(api_get "/users/dashboard-info")
split_response "$RESP"
assert_status "获取用户仪表盘" 200 "$STATUS"
assert_code0 "用户仪表盘 code=0" "$BODY"

# 4.10 登录历史
RESP=$(api_get "/users/login-history?page=1&page_size=5")
split_response "$RESP"
assert_status "获取登录历史" 200 "$STATUS"

# 4.11 用户等级
RESP=$(api_get "/users/my-level")
split_response "$RESP"
assert_status "获取用户等级" 200 "$STATUS"

# 4.12 活动记录
RESP=$(api_get "/users/activities?page=1&page_size=5")
split_response "$RESP"
assert_status "获取活动记录" 200 "$STATUS"

# 4.13 用户设备
RESP=$(api_get "/users/devices")
split_response "$RESP"
assert_status "获取用户设备" 200 "$STATUS"

# 4.14 订阅重置记录
RESP=$(api_get "/users/subscription-resets")
split_response "$RESP"
assert_status "获取订阅重置记录" 200 "$STATUS"
# PLACEHOLDER_PUBLIC

# ============================================================
# 5. 公开路由
# ============================================================
section "5. 公开路由 (Public)"

RESP=$(curl -s -w "\n%{http_code}" "$BASE/config" 2>/dev/null)
split_response "$RESP"
assert_status "获取公开配置" 200 "$STATUS"
assert_code0 "公开配置 code=0" "$BODY"

RESP=$(curl -s -w "\n%{http_code}" "$BASE/packages" 2>/dev/null)
split_response "$RESP"
assert_status "获取公开套餐列表" 200 "$STATUS"
assert_code0 "公开套餐 code=0" "$BODY"

RESP=$(curl -s -w "\n%{http_code}" "$BASE/announcements" 2>/dev/null)
split_response "$RESP"
assert_status "获取公开公告" 200 "$STATUS"

RESP=$(curl -s -w "\n%{http_code}" "$BASE/payment/methods" 2>/dev/null)
split_response "$RESP"
assert_status "获取支付方式" 200 "$STATUS"

# 验证邀请码 (无效)
RESP=$(curl -s -w "\n%{http_code}" "$BASE/invites/validate/INVALID_CODE" 2>/dev/null)
split_response "$RESP"
assert_status_in "验证无效邀请码" "$STATUS" 200 400 404

# ============================================================
# 6. 用户订阅模块
# ============================================================
section "6. 用户订阅模块 (Subscription)"

RESP=$(api_get "/subscriptions/user-subscription")
split_response "$RESP"
if [ "$STATUS" -eq 200 ]; then
  assert_status "获取用户订阅" 200 "$STATUS"
  assert_json_field "订阅有 universal_url" "$BODY" "data.universal_url"
  assert_json_field "订阅有 clash_url" "$BODY" "data.clash_url"
  assert_json_field "订阅有 expire_time" "$BODY" "data.expire_time"
  assert_json_field "订阅有 device_limit" "$BODY" "data.device_limit"
  assert_json_field "订阅有 universal_count" "$BODY" "data.universal_count"
  assert_json_field "订阅有 clash_count" "$BODY" "data.clash_count"
  assert_json_field "订阅有 status" "$BODY" "data.status"

  SUB_TOKEN=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)['data']
url=d.get('universal_url','')
print(url.split('/')[-1] if url else d.get('subscription_url',''))
" 2>/dev/null)
else
  assert_status "获取用户订阅 (404=无订阅)" 404 "$STATUS"
  SUB_TOKEN=""
fi

# 订阅设备
RESP=$(api_get "/subscriptions/devices")
split_response "$RESP"
assert_status "获取订阅设备列表" 200 "$STATUS"

# 测试订阅链接
if [ -n "$SUB_TOKEN" ]; then
  RESP=$(curl -s -w "\n%{http_code}" "$BASE/subscribe/$SUB_TOKEN" 2>/dev/null)
  split_response "$RESP"
  assert_status "Clash 订阅链接" 200 "$STATUS"
  if echo "$BODY" | grep -q "proxies:"; then
    echo -e "  ${GREEN}✓${NC} Clash 订阅返回有效 YAML"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} Clash 订阅未返回有效 YAML"
    FAIL=$((FAIL+1))
    ERRORS="${ERRORS}\n  ✗ Clash 订阅未返回有效 YAML"
  fi

  # Clash 格式路由
  RESP=$(curl -s -w "\n%{http_code}" "$BASE/subscribe/clash/$SUB_TOKEN" 2>/dev/null)
  split_response "$RESP"
  assert_status "Clash 格式订阅" 200 "$STATUS"

  # 通用订阅
  RESP=$(curl -s -w "\n%{http_code}" "$BASE/subscribe/universal/$SUB_TOKEN" 2>/dev/null)
  split_response "$RESP"
  assert_status "通用订阅链接 (Base64)" 200 "$STATUS"
  DECODED=$(echo "$BODY" | base64 -d 2>/dev/null || echo "")
  if echo "$DECODED" | grep -qE "(vmess://|vless://|trojan://|ss://|hysteria|ℹ️)"; then
    echo -e "  ${GREEN}✓${NC} 通用订阅 Base64 解码包含内容"
    PASS=$((PASS+1))
  else
    echo -e "  ${YELLOW}⊘${NC} 通用订阅 Base64 解码无协议链接 (可能无在线节点)"
    SKIP=$((SKIP+1))
  fi

  # 无效订阅链接
  RESP=$(curl -s -w "\n%{http_code}" "$BASE/subscribe/invalid_token_12345" 2>/dev/null)
  split_response "$RESP"
  assert_status "无效订阅链接应返回 200" 200 "$STATUS"

  # 重置订阅
  RESP=$(api_post "/subscriptions/reset-subscription" '{}')
  split_response "$RESP"
  assert_status "重置订阅 URL" 200 "$STATUS"

  # 发送订阅邮件 (功能开发中)
  RESP=$(api_post "/subscriptions/send-subscription-email" '{}')
  split_response "$RESP"
  assert_status_in "发送订阅邮件" "$STATUS" 200 501
else
  skip_test "Clash 订阅链接 (无订阅)"
  skip_test "Clash 格式订阅 (无订阅)"
  skip_test "通用订阅链接 (无订阅)"
  skip_test "无效订阅链接 (无订阅)"
  skip_test "重置订阅 URL (无订阅)"
  skip_test "发送订阅邮件 (无订阅)"
fi

# 转换为余额 (可能失败因为条件不满足)
RESP=$(api_post "/subscriptions/convert-to-balance" '{}')
split_response "$RESP"
assert_status_in "转换订阅为余额" "$STATUS" 200 400 404
# PLACEHOLDER_ORDERS

# ============================================================
# 7. 用户订单 & 购买流程
# ============================================================
section "7. 用户订单 & 购买流程"

RESP=$(api_get "/orders?page=1&page_size=10")
split_response "$RESP"
assert_status "用户订单列表" 200 "$STATUS"

# 获取套餐列表找一个可用套餐
PKG_RESP=$(curl -s "$BASE/packages" 2>/dev/null)
FIRST_PKG_ID=$(echo "$PKG_RESP" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items=d.get('data',d.get('items',[]))
if isinstance(items,list) and items:
    print(items[0].get('id',''))
else:
    print('')
" 2>/dev/null)

if [ -n "$FIRST_PKG_ID" ] && [ "$FIRST_PKG_ID" != "" ]; then
  # 创建订单
  RESP=$(api_post "/orders" "{\"package_id\":$FIRST_PKG_ID}")
  split_response "$RESP"
  assert_status "创建订单" 200 "$STATUS"
  ORDER_NO=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('order_no',''))" 2>/dev/null)

  if [ -n "$ORDER_NO" ] && [ "$ORDER_NO" != "" ]; then
    # 查询订单状态
    RESP=$(api_get "/orders/$ORDER_NO/status")
    split_response "$RESP"
    assert_status "查询订单状态" 200 "$STATUS"

    # 余额支付
    RESP=$(api_post "/orders/$ORDER_NO/pay" '{"payment_method":"balance"}')
    split_response "$RESP"
    assert_status_in "余额支付订单" "$STATUS" 200 400

    # 创建另一个订单用于取消
    RESP=$(api_post "/orders" "{\"package_id\":$FIRST_PKG_ID}")
    split_response "$RESP"
    ORDER_NO2=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('order_no',''))" 2>/dev/null)
    if [ -n "$ORDER_NO2" ] && [ "$ORDER_NO2" != "" ]; then
      RESP=$(api_post "/orders/$ORDER_NO2/cancel" '{}')
      split_response "$RESP"
      assert_status "取消订单" 200 "$STATUS"
    else
      skip_test "取消订单 (创建失败)"
    fi

    # 带优惠券创建订单
    RESP=$(api_post "/orders" "{\"package_id\":$FIRST_PKG_ID,\"coupon_code\":\"INVALID_COUPON\"}")
    split_response "$RESP"
    assert_status_in "无效优惠券创建订单" "$STATUS" 200 400
  else
    skip_test "查询订单状态 (创建失败)"
    skip_test "余额支付订单 (创建失败)"
    skip_test "取消订单 (创建失败)"
    skip_test "无效优惠券创建订单 (创建失败)"
  fi
else
  skip_test "创建订单 (无套餐)"
  skip_test "查询订单状态 (无套餐)"
  skip_test "余额支付订单 (无套餐)"
  skip_test "取消订单 (无套餐)"
  skip_test "无效优惠券创建订单 (无套餐)"
fi

# ============================================================
# 8. 用户工单系统
# ============================================================
section "8. 用户工单系统"

RESP=$(api_get "/tickets?page=1&page_size=10")
split_response "$RESP"
assert_status "用户工单列表" 200 "$STATUS"

# 创建工单
RESP=$(api_post "/tickets" '{"title":"测试工单","content":"这是自动测试创建的工单","type":"other","priority":"low"}')
split_response "$RESP"
assert_status "创建工单" 200 "$STATUS"
TEST_TICKET_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_TICKET_ID" ] && [ "$TEST_TICKET_ID" != "" ]; then
  RESP=$(api_get "/tickets/$TEST_TICKET_ID")
  split_response "$RESP"
  assert_status "获取工单详情" 200 "$STATUS"

  RESP=$(api_post "/tickets/$TEST_TICKET_ID/reply" '{"content":"自动测试回复"}')
  split_response "$RESP"
  assert_status "回复工单" 200 "$STATUS"

  RESP=$(api_put "/tickets/$TEST_TICKET_ID" '{"status":"closed"}')
  split_response "$RESP"
  assert_status "关闭工单" 200 "$STATUS"
else
  skip_test "获取工单详情 (创建失败)"
  skip_test "回复工单 (创建失败)"
  skip_test "关闭工单 (创建失败)"
fi

# ============================================================
# 9. 用户通知系统
# ============================================================
section "9. 用户通知系统"

RESP=$(api_get "/notifications?page=1&page_size=10")
split_response "$RESP"
assert_status "通知列表" 200 "$STATUS"

RESP=$(api_get "/notifications/unread-count")
split_response "$RESP"
assert_status "未读通知数" 200 "$STATUS"

# 标记全部已读
RESP=$(api_put "/notifications/read-all" '{}')
split_response "$RESP"
assert_status "标记全部已读" 200 "$STATUS"

# 获取通知列表看有没有可操作的
NOTIF_ID=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items=d.get('data',{}).get('items',d.get('data',[]))
if isinstance(items,list) and items:
    print(items[0].get('id',''))
else:
    print('')
" 2>/dev/null)
# 重新获取通知列表
RESP=$(api_get "/notifications?page=1&page_size=10")
split_response "$RESP"
NOTIF_ID=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items=d.get('data',{}).get('items',d.get('data',[]))
if isinstance(items,list) and items:
    print(items[0].get('id',''))
else:
    print('')
" 2>/dev/null)

if [ -n "$NOTIF_ID" ] && [ "$NOTIF_ID" != "" ]; then
  RESP=$(api_put "/notifications/$NOTIF_ID/read" '{}')
  split_response "$RESP"
  assert_status "标记单条已读" 200 "$STATUS"

  RESP=$(api_delete "/notifications/$NOTIF_ID")
  split_response "$RESP"
  assert_status "删除通知" 200 "$STATUS"
else
  skip_test "标记单条已读 (无通知)"
  skip_test "删除通知 (无通知)"
fi
# PLACEHOLDER_INVITE

# ============================================================
# 10. 用户邀请系统
# ============================================================
section "10. 用户邀请系统"

RESP=$(api_get "/invites")
split_response "$RESP"
assert_status "邀请码列表" 200 "$STATUS"

RESP=$(api_get "/invites/stats")
split_response "$RESP"
assert_status "邀请统计" 200 "$STATUS"

RESP=$(api_get "/invites/my-codes")
split_response "$RESP"
assert_status "我的邀请码" 200 "$STATUS"

# 创建邀请码
RESP=$(api_post "/invites" '{"inviter_reward":1.0,"invitee_reward":0.5,"max_uses":10,"expires_in_days":30}')
split_response "$RESP"
assert_status "创建邀请码" 200 "$STATUS"
INVITE_CODE=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('code',''))" 2>/dev/null)

# 验证刚创建的邀请码
if [ -n "$INVITE_CODE" ] && [ "$INVITE_CODE" != "" ]; then
  RESP=$(curl -s -w "\n%{http_code}" "$BASE/invites/validate/$INVITE_CODE" 2>/dev/null)
  split_response "$RESP"
  assert_status "验证有效邀请码" 200 "$STATUS"
else
  skip_test "验证有效邀请码 (创建失败)"
fi

# ============================================================
# 11. 用户节点
# ============================================================
section "11. 用户节点"

RESP=$(api_get "/nodes?page=1&page_size=10")
split_response "$RESP"
assert_status "用户节点列表" 200 "$STATUS"

RESP=$(api_get "/nodes/stats")
split_response "$RESP"
assert_status "节点统计" 200 "$STATUS"

# ============================================================
# 12. 用户设备
# ============================================================
section "12. 用户设备"

RESP=$(api_get "/devices")
split_response "$RESP"
assert_status "设备列表" 200 "$STATUS"

# ============================================================
# 13. 用户充值
# ============================================================
section "13. 用户充值"

RESP=$(api_get "/recharge")
split_response "$RESP"
assert_status "充值记录" 200 "$STATUS"

# 创建充值
RESP=$(api_post "/recharge" '{"amount":10.0}')
split_response "$RESP"
assert_status "创建充值" 200 "$STATUS"
RECHARGE_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$RECHARGE_ID" ] && [ "$RECHARGE_ID" != "" ]; then
  RESP=$(api_post "/recharge/$RECHARGE_ID/cancel" '{}')
  split_response "$RESP"
  assert_status "取消充值" 200 "$STATUS"
else
  skip_test "取消充值 (创建失败)"
fi

# 充值金额无效
RESP=$(api_post "/recharge" '{"amount":-5}')
split_response "$RESP"
assert_status "充值金额无效" 400 "$STATUS"

# ============================================================
# 14. 用户兑换 & 优惠券
# ============================================================
section "14. 用户兑换 & 优惠券"

RESP=$(api_get "/redeem/history")
split_response "$RESP"
assert_status "兑换历史" 200 "$STATUS"

# 兑换无效卡密
RESP=$(api_post "/redeem" '{"code":"INVALID_REDEEM_CODE"}')
split_response "$RESP"
assert_status_in "兑换无效卡密" "$STATUS" 400 404

RESP=$(api_get "/coupons/my")
split_response "$RESP"
assert_status "我的优惠券" 200 "$STATUS"

# 验证优惠券
RESP=$(api_post "/coupons/verify" '{"code":"INVALID_COUPON"}')
split_response "$RESP"
assert_status_in "验证无效优惠券" "$STATUS" 200 400 404
# PLACEHOLDER_ADMIN

# ============================================================
# 15. 管理员 - 仪表盘 & 统计
# ============================================================
section "15. 管理员仪表盘 & 统计"

RESP=$(api_get "/admin/dashboard")
split_response "$RESP"
assert_status "管理员仪表盘" 200 "$STATUS"
assert_code0 "仪表盘 code=0" "$BODY"
assert_json_field "仪表盘有 total_users" "$BODY" "data.total_users"

RESP=$(api_get "/admin/stats")
split_response "$RESP"
assert_status "管理员统计" 200 "$STATUS"

RESP=$(api_get "/admin/stats/revenue")
split_response "$RESP"
assert_status "收入统计" 200 "$STATUS"

RESP=$(api_get "/admin/stats/users")
split_response "$RESP"
assert_status "用户统计" 200 "$STATUS"

RESP=$(api_get "/admin/stats/regions")
split_response "$RESP"
assert_status "地区统计" 200 "$STATUS"

RESP=$(api_get "/admin/monitoring")
split_response "$RESP"
assert_status "系统监控" 200 "$STATUS"

# ============================================================
# 16. 管理员 - 用户管理
# ============================================================
section "16. 管理员用户管理"

RESP=$(api_get "/admin/users?page=1&page_size=10")
split_response "$RESP"
assert_status "管理员用户列表" 200 "$STATUS"
assert_code0 "用户列表 code=0" "$BODY"
assert_json_field "用户列表有 items" "$BODY" "data.items"
assert_json_field "用户列表有 total" "$BODY" "data.total"

# 分页第2页
RESP=$(api_get "/admin/users?page=2&page_size=5")
split_response "$RESP"
assert_status "用户列表第2页" 200 "$STATUS"

# 创建测试用户
TEST_EMAIL="test_$(date +%s)@test.com"
RESP=$(api_post "/admin/users" "{\"email\":\"$TEST_EMAIL\",\"username\":\"testuser_$$\",\"password\":\"Test123456\"}")
split_response "$RESP"
assert_status "创建测试用户" 200 "$STATUS"
TEST_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_USER_ID" ] && [ "$TEST_USER_ID" != "" ]; then
  RESP=$(api_get "/admin/users/$TEST_USER_ID")
  split_response "$RESP"
  assert_status "获取用户详情" 200 "$STATUS"
  assert_json_field "用户详情有 email" "$BODY" "data.user.email"

  RESP=$(api_put "/admin/users/$TEST_USER_ID" '{"notes":"自动测试备注","is_active":true}')
  split_response "$RESP"
  assert_status "更新用户" 200 "$STATUS"

  RESP=$(api_post "/admin/users/$TEST_USER_ID/toggle-active" '{}')
  split_response "$RESP"
  assert_status "切换用户状态" 200 "$STATUS"

  RESP=$(api_post "/admin/users/$TEST_USER_ID/reset-password" '{"password":"NewPass123"}')
  split_response "$RESP"
  assert_status "重置用户密码" 200 "$STATUS"

  RESP=$(api_delete "/admin/users/$TEST_USER_ID")
  split_response "$RESP"
  assert_status "删除测试用户" 200 "$STATUS"
else
  skip_test "获取用户详情 (创建失败)"
  skip_test "更新用户 (创建失败)"
  skip_test "切换用户状态 (创建失败)"
  skip_test "重置用户密码 (创建失败)"
  skip_test "删除测试用户 (创建失败)"
fi

# 异常用户
RESP=$(api_get "/admin/users/abnormal")
split_response "$RESP"
assert_status "异常用户列表" 200 "$STATUS"

# ============================================================
# 17. 管理员 - 节点管理
# ============================================================
section "17. 管理员节点管理"

RESP=$(api_get "/admin/nodes?page=1&page_size=20")
split_response "$RESP"
assert_status "管理员节点列表" 200 "$STATUS"
assert_code0 "节点列表 code=0" "$BODY"
assert_json_field "节点列表有 items" "$BODY" "data.items"
assert_json_field "节点列表有 total" "$BODY" "data.total"

# 分页验证
NODE_TOTAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['total'])" 2>/dev/null)
NODE_ITEMS=$(echo "$BODY" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['data']['items']))" 2>/dev/null)
if [ "$NODE_TOTAL" -le 20 ] 2>/dev/null || [ "$NODE_ITEMS" -le 20 ] 2>/dev/null; then
  echo -e "  ${GREEN}✓${NC} 节点分页正确 (total=$NODE_TOTAL, 本页=$NODE_ITEMS)"
  PASS=$((PASS+1))
else
  echo -e "  ${RED}✗${NC} 节点分页异常"
  FAIL=$((FAIL+1))
fi

# 更新节点
FIRST_NODE_ID=$(echo "$BODY" | python3 -c "
import sys,json
items=json.load(sys.stdin)['data']['items']
print(items[0]['id'] if items else '')
" 2>/dev/null)
if [ -n "$FIRST_NODE_ID" ] && [ "$FIRST_NODE_ID" != "" ]; then
  RESP=$(api_put "/admin/nodes/$FIRST_NODE_ID" '{"name":"测试更新节点名","region":"测试地区","is_active":true}')
  split_response "$RESP"
  assert_status "更新节点" 200 "$STATUS"
fi

# 导入节点
RESP=$(api_post "/admin/nodes/import" '{"type":"links","links":"ss://YWVzLTI1Ni1nY206dGVzdHBhc3M=@1.2.3.4:8388#TestNode"}')
split_response "$RESP"
assert_status "导入节点链接" 200 "$STATUS"
# PLACEHOLDER_ADMIN2

# ============================================================
# 18. 管理员 - 节点自动更新
# ============================================================
section "18. 节点自动更新 (Config Update)"

RESP=$(api_get "/admin/config-update/status")
split_response "$RESP"
assert_status "获取更新状态" 200 "$STATUS"

RESP=$(api_get "/admin/config-update/config")
split_response "$RESP"
assert_status "获取更新配置" 200 "$STATUS"

RESP=$(api_put "/admin/config-update/config" '{"urls":["https://example.com/sub"],"keywords":["hk","us","jp"],"enabled":false,"interval":60}')
split_response "$RESP"
assert_status "保存更新配置" 200 "$STATUS"

RESP=$(api_get "/admin/config-update/logs")
split_response "$RESP"
assert_status "获取更新日志" 200 "$STATUS"

RESP=$(api_post "/admin/config-update/logs/clear" '{}')
split_response "$RESP"
assert_status "清除更新日志" 200 "$STATUS"

# ============================================================
# 19. 管理员 - 订阅管理
# ============================================================
section "19. 管理员订阅管理"

RESP=$(api_get "/admin/subscriptions?page=1&page_size=10")
split_response "$RESP"
assert_status "管理员订阅列表" 200 "$STATUS"
assert_code0 "订阅列表 code=0" "$BODY"
assert_json_field "订阅列表有 items" "$BODY" "data.items"
assert_json_field "订阅列表有 total" "$BODY" "data.total"

FIRST_SUB_ID=$(echo "$BODY" | python3 -c "
import sys,json
items=json.load(sys.stdin)['data']['items']
print(items[0]['id'] if items else '')
" 2>/dev/null)

if [ -n "$FIRST_SUB_ID" ] && [ "$FIRST_SUB_ID" != "" ]; then
  RESP=$(api_get "/admin/subscriptions/$FIRST_SUB_ID")
  split_response "$RESP"
  assert_status "获取订阅详情" 200 "$STATUS"
  assert_json_field "订阅详情有 universal_url" "$BODY" "data.universal_url"
  assert_json_field "订阅详情有 clash_url" "$BODY" "data.clash_url"

  RESP=$(api_post "/admin/subscriptions/$FIRST_SUB_ID/extend" '{"days":1}')
  split_response "$RESP"
  assert_status "延长订阅 (+1天)" 200 "$STATUS"

  RESP=$(api_put "/admin/subscriptions/$FIRST_SUB_ID" '{"device_limit":5}')
  split_response "$RESP"
  assert_status "更新设备限制" 200 "$STATUS"

  RESP=$(api_post "/admin/subscriptions/$FIRST_SUB_ID/reset" '{}')
  split_response "$RESP"
  assert_status "管理员重置订阅" 200 "$STATUS"

  RESP=$(api_get "/admin/subscriptions?search=admin")
  split_response "$RESP"
  assert_status "搜索订阅" 200 "$STATUS"

  RESP=$(api_get "/admin/subscriptions?status=active")
  split_response "$RESP"
  assert_status "按状态筛选订阅" 200 "$STATUS"
else
  skip_test "获取订阅详情 (无订阅)"
  skip_test "延长订阅 (无订阅)"
  skip_test "更新设备限制 (无订阅)"
  skip_test "管理员重置订阅 (无订阅)"
  skip_test "搜索订阅 (无订阅)"
  skip_test "按状态筛选订阅 (无订阅)"
fi

# ============================================================
# 20. 管理员 - 套餐管理
# ============================================================
section "20. 管理员套餐管理"

RESP=$(api_get "/admin/packages?page=1&page_size=10")
split_response "$RESP"
assert_status "管理员套餐列表" 200 "$STATUS"

RESP=$(api_post "/admin/packages" '{"name":"测试套餐","price":9.99,"duration_days":30,"device_limit":3,"description":"自动测试套餐","is_active":true}')
split_response "$RESP"
assert_status "创建套餐" 200 "$STATUS"
TEST_PKG_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_PKG_ID" ] && [ "$TEST_PKG_ID" != "" ]; then
  RESP=$(api_put "/admin/packages/$TEST_PKG_ID" '{"name":"测试套餐-已更新","price":19.99}')
  split_response "$RESP"
  assert_status "更新套餐" 200 "$STATUS"

  RESP=$(api_delete "/admin/packages/$TEST_PKG_ID")
  split_response "$RESP"
  assert_status "删除套餐" 200 "$STATUS"
else
  skip_test "更新套餐 (创建失败)"
  skip_test "删除套餐 (创建失败)"
fi

# ============================================================
# 21. 管理员 - 订单管理
# ============================================================
section "21. 管理员订单管理"

RESP=$(api_get "/admin/orders?page=1&page_size=10")
split_response "$RESP"
assert_status "管理员订单列表" 200 "$STATUS"
assert_code0 "订单列表 code=0" "$BODY"

# 获取第一个订单详情
FIRST_ORDER_ID=$(echo "$BODY" | python3 -c "
import sys,json
items=json.load(sys.stdin)['data']['items']
print(items[0]['id'] if items else '')
" 2>/dev/null)
if [ -n "$FIRST_ORDER_ID" ] && [ "$FIRST_ORDER_ID" != "" ]; then
  RESP=$(api_get "/admin/orders/$FIRST_ORDER_ID")
  split_response "$RESP"
  assert_status "获取订单详情" 200 "$STATUS"
else
  skip_test "获取订单详情 (无订单)"
fi

# ============================================================
# 22. 管理员 - 优惠券管理
# ============================================================
section "22. 管理员优惠券管理"

RESP=$(api_get "/admin/coupons?page=1&page_size=10")
split_response "$RESP"
assert_status "优惠券列表" 200 "$STATUS"

RESP=$(api_post "/admin/coupons" '{"code":"TESTCOUPON_'$$'","discount_type":"percentage","discount_value":10,"max_uses":100,"is_active":true}')
split_response "$RESP"
assert_status "创建优惠券" 200 "$STATUS"
TEST_COUPON_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_COUPON_ID" ] && [ "$TEST_COUPON_ID" != "" ]; then
  RESP=$(api_put "/admin/coupons/$TEST_COUPON_ID" '{"status":"disabled"}')
  split_response "$RESP"
  assert_status "更新优惠券" 200 "$STATUS"

  RESP=$(api_delete "/admin/coupons/$TEST_COUPON_ID")
  split_response "$RESP"
  assert_status "删除优惠券" 200 "$STATUS"
else
  skip_test "更新优惠券 (创建失败)"
  skip_test "删除优惠券 (创建失败)"
fi
# PLACEHOLDER_ADMIN3

# ============================================================
# 23. 管理员 - 公告管理
# ============================================================
section "23. 管理员公告管理"

RESP=$(api_get "/admin/announcements?page=1&page_size=10")
split_response "$RESP"
assert_status "公告列表" 200 "$STATUS"

RESP=$(api_post "/admin/announcements" '{"title":"测试公告","content":"这是自动测试创建的公告","is_active":true}')
split_response "$RESP"
assert_status "创建公告" 200 "$STATUS"
TEST_ANN_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_ANN_ID" ] && [ "$TEST_ANN_ID" != "" ]; then
  RESP=$(api_put "/admin/announcements/$TEST_ANN_ID" '{"title":"测试公告-已更新","is_active":false}')
  split_response "$RESP"
  assert_status "更新公告" 200 "$STATUS"

  RESP=$(api_delete "/admin/announcements/$TEST_ANN_ID")
  split_response "$RESP"
  assert_status "删除公告" 200 "$STATUS"
else
  skip_test "更新公告 (创建失败)"
  skip_test "删除公告 (创建失败)"
fi

# ============================================================
# 24. 管理员 - 工单管理
# ============================================================
section "24. 管理员工单管理"

RESP=$(api_get "/admin/tickets?page=1&page_size=10")
split_response "$RESP"
assert_status "管理员工单列表" 200 "$STATUS"

# 获取第一个工单
ADMIN_TICKET_ID=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items=d.get('data',{}).get('items',d.get('data',[]))
if isinstance(items,list) and items:
    print(items[0].get('id',''))
else:
    print('')
" 2>/dev/null)

if [ -n "$ADMIN_TICKET_ID" ] && [ "$ADMIN_TICKET_ID" != "" ]; then
  RESP=$(api_get "/admin/tickets/$ADMIN_TICKET_ID")
  split_response "$RESP"
  assert_status "管理员获取工单详情" 200 "$STATUS"

  RESP=$(api_post "/admin/tickets/$ADMIN_TICKET_ID/reply" '{"content":"管理员自动测试回复"}')
  split_response "$RESP"
  assert_status "管理员回复工单" 200 "$STATUS"

  RESP=$(api_put "/admin/tickets/$ADMIN_TICKET_ID" '{"status":"closed"}')
  split_response "$RESP"
  assert_status "管理员关闭工单" 200 "$STATUS"
else
  skip_test "管理员获取工单详情 (无工单)"
  skip_test "管理员回复工单 (无工单)"
  skip_test "管理员关闭工单 (无工单)"
fi

# ============================================================
# 25. 管理员 - 用户等级
# ============================================================
section "25. 管理员用户等级"

RESP=$(api_get "/admin/user-levels?page=1&page_size=10")
split_response "$RESP"
assert_status "用户等级列表" 200 "$STATUS"

RESP=$(api_post "/admin/user-levels" '{"name":"测试等级","level":99,"discount":0.8,"description":"自动测试等级"}')
split_response "$RESP"
assert_status "创建用户等级" 200 "$STATUS"
TEST_LEVEL_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_LEVEL_ID" ] && [ "$TEST_LEVEL_ID" != "" ]; then
  RESP=$(api_put "/admin/user-levels/$TEST_LEVEL_ID" '{"name":"测试等级-已更新"}')
  split_response "$RESP"
  assert_status "更新用户等级" 200 "$STATUS"

  RESP=$(api_delete "/admin/user-levels/$TEST_LEVEL_ID")
  split_response "$RESP"
  assert_status "删除用户等级" 200 "$STATUS"
else
  skip_test "更新用户等级 (创建失败)"
  skip_test "删除用户等级 (创建失败)"
fi

# ============================================================
# 26. 管理员 - 卡密管理
# ============================================================
section "26. 管理员卡密管理"

RESP=$(api_get "/admin/redeem-codes?page=1&page_size=10")
split_response "$RESP"
assert_status "卡密列表" 200 "$STATUS"

RESP=$(api_post "/admin/redeem-codes" '{"type":"duration","value":30,"quantity":2,"name":"测试卡密"}')
split_response "$RESP"
assert_status "创建卡密" 200 "$STATUS"

# 提取卡密 ID 用于删除
REDEEM_ID=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin).get('data',{})
codes=d.get('codes',[])
if codes and isinstance(codes[0],dict):
    print(codes[0].get('id',''))
elif d.get('id'):
    print(d['id'])
else:
    print('')
" 2>/dev/null)

# ============================================================
# 27. 管理员 - 专线节点
# ============================================================
section "27. 管理员专线节点"

RESP=$(api_get "/admin/custom-nodes?page=1&page_size=10")
split_response "$RESP"
assert_status "专线节点列表" 200 "$STATUS"

RESP=$(api_post "/admin/custom-nodes" '{"name":"测试专线","type":"vmess","server":"1.2.3.4","port":443,"config":"{}","is_active":true}')
split_response "$RESP"
assert_status "创建专线节点" 200 "$STATUS"
TEST_CN_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

if [ -n "$TEST_CN_ID" ] && [ "$TEST_CN_ID" != "" ]; then
  RESP=$(api_put "/admin/custom-nodes/$TEST_CN_ID" '{"name":"测试专线-已更新"}')
  split_response "$RESP"
  assert_status "更新专线节点" 200 "$STATUS"

  RESP=$(api_get "/admin/custom-nodes/$TEST_CN_ID/link")
  split_response "$RESP"
  assert_status "获取专线节点链接" 200 "$STATUS"

  RESP=$(api_get "/admin/custom-nodes/$TEST_CN_ID/users")
  split_response "$RESP"
  assert_status "获取专线节点用户" 200 "$STATUS"

  # 分配专线给用户
  RESP=$(api_post "/admin/custom-nodes/$TEST_CN_ID/assign" '{"user_ids":[1]}')
  split_response "$RESP"
  assert_status_in "分配专线给用户" "$STATUS" 200 400

  RESP=$(api_delete "/admin/custom-nodes/$TEST_CN_ID")
  split_response "$RESP"
  assert_status "删除专线节点" 200 "$STATUS"
else
  skip_test "更新专线节点 (创建失败)"
  skip_test "获取专线节点链接 (创建失败)"
  skip_test "获取专线节点用户 (创建失败)"
  skip_test "分配专线给用户 (创建失败)"
  skip_test "删除专线节点 (创建失败)"
fi

# 导入专线链接
RESP=$(api_post "/admin/custom-nodes/import-links" '{"links":"ss://YWVzLTI1Ni1nY206dGVzdA==@5.6.7.8:8388#CustomTest"}')
split_response "$RESP"
assert_status "导入专线链接" 200 "$STATUS"
# PLACEHOLDER_ADMIN4

# ============================================================
# 28. 管理员 - 邮件队列
# ============================================================
section "28. 管理员邮件队列"

RESP=$(api_get "/admin/email-queue?page=1&page_size=10")
split_response "$RESP"
assert_status "邮件队列列表" 200 "$STATUS"

# 按状态筛选
RESP=$(api_get "/admin/email-queue?status=pending&page=1&page_size=10")
split_response "$RESP"
assert_status "邮件队列 (pending)" 200 "$STATUS"

RESP=$(api_get "/admin/email-queue?status=sent&page=1&page_size=10")
split_response "$RESP"
assert_status "邮件队列 (sent)" 200 "$STATUS"

RESP=$(api_get "/admin/email-queue?status=failed&page=1&page_size=10")
split_response "$RESP"
assert_status "邮件队列 (failed)" 200 "$STATUS"

# ============================================================
# 29. 管理员 - 系统设置 (全部7个Tab)
# ============================================================
section "29. 管理员系统设置"

# 获取所有设置
RESP=$(api_get "/admin/settings")
split_response "$RESP"
assert_status "获取系统设置" 200 "$STATUS"
assert_code0 "系统设置 code=0" "$BODY"

# 保存基本设置
RESP=$(api_put "/admin/settings" '{"site_name":"CBoard测试","domain_name":"test.example.com"}')
split_response "$RESP"
assert_status "保存基本设置" 200 "$STATUS"

# 保存注册设置
RESP=$(api_put "/admin/settings" '{"enable_registration":"true","require_email_verification":"false","require_invite_code":"false"}')
split_response "$RESP"
assert_status "保存注册设置" 200 "$STATUS"

# 保存邮件设置
RESP=$(api_put "/admin/settings" '{"smtp_host":"smtp.example.com","smtp_port":"587","smtp_username":"test@example.com","smtp_password":"testpass","smtp_from":"test@example.com"}')
split_response "$RESP"
assert_status "保存邮件设置" 200 "$STATUS"

# 发送测试邮件 (可能失败因为SMTP未配置)
RESP=$(api_post "/admin/settings/test-email" '{"email":"test@example.com"}')
split_response "$RESP"
assert_status_in "发送测试邮件" "$STATUS" 200 500

# 保存支付设置
RESP=$(api_put "/admin/settings" '{"enable_balance_payment":"true"}')
split_response "$RESP"
assert_status "保存支付设置" 200 "$STATUS"

# 保存通知设置
RESP=$(api_put "/admin/settings" '{"telegram_bot_token":"","telegram_chat_id":""}')
split_response "$RESP"
assert_status "保存通知设置" 200 "$STATUS"

# 保存安全设置
RESP=$(api_put "/admin/settings" '{"login_rate_limit":"10","enable_captcha":"false"}')
split_response "$RESP"
assert_status "保存安全设置" 200 "$STATUS"

# ============================================================
# 30. 管理员 - 日志模块 (全部6种)
# ============================================================
section "30. 管理员日志模块"

RESP=$(api_get "/admin/logs/audit?page=1&page_size=5")
split_response "$RESP"
assert_status "审计日志" 200 "$STATUS"

RESP=$(api_get "/admin/logs/login?page=1&page_size=5")
split_response "$RESP"
assert_status "登录日志" 200 "$STATUS"

RESP=$(api_get "/admin/logs/registration?page=1&page_size=5")
split_response "$RESP"
assert_status "注册日志" 200 "$STATUS"

RESP=$(api_get "/admin/logs/subscription?page=1&page_size=5")
split_response "$RESP"
assert_status "订阅日志" 200 "$STATUS"

RESP=$(api_get "/admin/logs/balance?page=1&page_size=5")
split_response "$RESP"
assert_status "余额日志" 200 "$STATUS"

RESP=$(api_get "/admin/logs/commission?page=1&page_size=5")
split_response "$RESP"
assert_status "佣金日志" 200 "$STATUS"

# ============================================================
# 31. 管理员 - 备份
# ============================================================
section "31. 管理员备份"

RESP=$(api_get "/admin/backup")
split_response "$RESP"
assert_status "备份列表" 200 "$STATUS"

RESP=$(api_post "/admin/backup" '{}')
split_response "$RESP"
assert_status_in "创建备份" "$STATUS" 200 500

# ============================================================
# 32. 清理注册的测试用户
# ============================================================
section "32. 清理测试数据"

if [ -n "$REG_USER_ID" ] && [ "$REG_USER_ID" != "" ]; then
  RESP=$(api_delete "/admin/users/$REG_USER_ID")
  split_response "$RESP"
  assert_status "清理注册测试用户" 200 "$STATUS"
else
  skip_test "清理注册测试用户 (无ID)"
fi

# ============================================================
# 33. 登出测试
# ============================================================
section "33. 登出测试"

RESP=$(api_post "/auth/logout" '{}')
split_response "$RESP"
assert_status "登出" 200 "$STATUS"

RESP=$(api_get "/users/me")
split_response "$RESP"
assert_status "登出后访问应返回 401" 401 "$STATUS"

# ============================================================
# 34. 前端构建测试
# ============================================================
section "34. 前端构建"

cd /Users/apple/v2/frontend
BUILD_OUTPUT=$(npx vite build 2>&1)
BUILD_EXIT=$?
if [ $BUILD_EXIT -eq 0 ]; then
  echo -e "  ${GREEN}✓${NC} 前端构建成功"
  PASS=$((PASS+1))
else
  echo -e "  ${RED}✗${NC} 前端构建失败"
  echo "$BUILD_OUTPUT" | tail -5
  FAIL=$((FAIL+1))
  ERRORS="${ERRORS}\n  ✗ 前端构建失败"
fi
cd /Users/apple/v2

# ============================================================
# 35. 后端编译测试
# ============================================================
section "35. 后端编译"

cd /Users/apple/v2
BUILD_OUTPUT=$(go build ./... 2>&1)
BUILD_EXIT=$?
if [ $BUILD_EXIT -eq 0 ]; then
  echo -e "  ${GREEN}✓${NC} 后端编译成功"
  PASS=$((PASS+1))
else
  echo -e "  ${RED}✗${NC} 后端编译失败"
  echo "$BUILD_OUTPUT" | tail -5
  FAIL=$((FAIL+1))
  ERRORS="${ERRORS}\n  ✗ 后端编译失败"
fi

# ============================================================
# 测试结果汇总
# ============================================================
echo ""
echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║              测试结果汇总                ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
echo ""
TOTAL=$((PASS+FAIL+SKIP))
echo -e "  总计: $TOTAL 项"
echo -e "  ${GREEN}通过: $PASS${NC}"
echo -e "  ${RED}失败: $FAIL${NC}"
echo -e "  ${YELLOW}跳过: $SKIP${NC}"
echo ""

if [ $FAIL -gt 0 ]; then
  echo -e "${RED}失败项:${NC}"
  echo -e "$ERRORS"
  echo ""
  exit 1
else
  echo -e "${GREEN}所有测试通过!${NC}"
  exit 0
fi
