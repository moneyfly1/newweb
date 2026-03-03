# 本地测试支付回调问题及解决方案

## 问题根源

**核心问题：** 本地测试时，支付宝无法回调到 `localhost`

### 为什么会这样？

1. **回调地址配置**
   - 当前配置：`https://go.moneyfly.top/api/v1/payment/notify/alipay`
   - 这是生产环境的公网地址
   - 支付宝只能回调到公网可访问的地址

2. **本地环境限制**
   - 本地服务器运行在 `localhost:8000`
   - 支付宝服务器无法访问你的本地机器
   - 即使支付成功，回调也无法到达本地服务器

3. **结果**
   - 订单创建成功 ✓
   - 支付成功 ✓
   - 回调失败 ✗（支付宝回调到生产服务器，而不是本地）
   - 订单状态不更新 ✗

## 已完成的订单

以下订单已手动完成：

| 订单ID | 订单号 | 状态 | 用户ID |
|--------|--------|------|--------|
| 271 | ORD1772506563FQRc0F | paid | 1377 |
| 272 | ORD1772506572OLp5qW | paid | 1377 |
| 273 | ORD1772506591XMaGGc | paid | 1377 |

订阅已更新，续期90天（3个订单 × 30天）

## 解决方案

### 方案1：使用内网穿透（推荐用于本地开发）

使用 ngrok 将本地服务器暴露到公网：

```bash
# 1. 安装 ngrok
brew install ngrok

# 2. 启动本地服务器
cd /Users/apple/v2
./cboard-server

# 3. 在另一个终端启动 ngrok
ngrok http 8000

# 4. 复制 ngrok 提供的 HTTPS 地址
# 例如：https://abc123.ngrok.io

# 5. 更新数据库配置
sqlite3 /Users/apple/v2/cboard.db "UPDATE system_configs SET value='https://abc123.ngrok.io/api/v1/payment/notify/alipay' WHERE key='pay_alipay_notify_url';"

# 6. 重启服务器
pkill -f cboard
./cboard-server

# 7. 创建测试订单并支付
# 这次回调应该能到达本地服务器
```

**优点：**
- 可以真实测试支付回调流程
- 可以在本地调试回调处理逻辑
- 可以看到完整的日志输出

**缺点：**
- 需要安装额外工具
- ngrok 免费版的 URL 每次重启都会变化
- 需要每次更新数据库配置

### 方案2：使用生产环境测试（推荐用于功能测试）

直接在生产服务器上测试：

```bash
# 1. SSH 到生产服务器
ssh user@go.moneyfly.top

# 2. 查看服务器日志
tail -f /path/to/logs/app.log

# 3. 在浏览器中访问生产环境
https://go.moneyfly.top

# 4. 创建测试订单并支付
# 回调会正常到达生产服务器
```

**优点：**
- 无需额外配置
- 真实环境测试
- 回调地址已正确配置

**缺点：**
- 需要访问生产服务器
- 可能影响生产数据
- 调试不如本地方便

### 方案3：手动模拟回调（用于已支付订单）

如果订单已经支付，但回调失败，可以手动完成：

```bash
# 批量完成所有待处理订单
./complete_pending_orders.sh

# 或者完成单个订单
sqlite3 /Users/apple/v2/cboard.db <<EOF
UPDATE orders SET status='paid', payment_time=datetime('now'), payment_method_name='支付宝' WHERE order_no='YOUR_ORDER_NO';
UPDATE payment_transactions SET status='paid', external_transaction_id='MANUAL_$(date +%Y%m%d%H%M%S)' WHERE order_id=YOUR_ORDER_ID;
EOF
```

**优点：**
- 快速解决已支付订单
- 无需等待回调
- 适合紧急情况

**缺点：**
- 需要手动操作
- 无法测试回调逻辑
- 不是长期解决方案

## 推荐的开发流程

### 本地开发时

1. **使用 ngrok 进行支付测试**
   ```bash
   # 终端1：启动服务器
   ./cboard-server

   # 终端2：启动 ngrok
   ngrok http 8000

   # 终端3：更新配置
   sqlite3 cboard.db "UPDATE system_configs SET value='https://YOUR_NGROK_URL/api/v1/payment/notify/alipay' WHERE key='pay_alipay_notify_url';"
   ```

2. **或者手动完成测试订单**
   ```bash
   # 支付后手动完成
   ./complete_pending_orders.sh
   ```

### 生产部署时

1. **确保回调地址正确**
   ```sql
   SELECT value FROM system_configs WHERE key='pay_alipay_notify_url';
   -- 应该是：https://go.moneyfly.top/api/v1/payment/notify/alipay
   ```

2. **验证回调端点可访问**
   ```bash
   curl -X POST https://go.moneyfly.top/api/v1/payment/notify/alipay -d "test=1"
   # 应该返回 "verify fail"
   ```

3. **检查支付宝开放平台配置**
   - 登录 https://open.alipay.com/
   - 检查应用的"授权回调地址"
   - 确保与数据库配置一致

## 验证修复是否成功

### 1. 检查 transaction_id 是否正确

```bash
sqlite3 /Users/apple/v2/cboard.db "SELECT id, order_id, transaction_id, status FROM payment_transactions ORDER BY id DESC LIMIT 5;"
```

**期望结果：**
- `transaction_id` 应该是订单号（如 `ORD1772506591XMaGGc`）
- 不应该是 `PAY` 开头的随机字符串

### 2. 测试支付流程

1. 创建测试订单
2. 选择支付宝支付
3. 扫码支付（使用小额如 0.01 元）
4. 观察服务器日志

**期望看到：**
```
[alipay] ========== 收到支付宝回调 ==========
[alipay] Method: POST
[alipay] ✓ 回调验证成功
[alipay]   - out_trade_no: ORD1772506591XMaGGc
[alipay]   - trade_status: TRADE_SUCCESS
[alipay] ✓ 找到订单
[alipay] ✓ 订单状态已更新为 paid
[alipay] ✓ 订阅激活成功
```

### 3. 检查订单状态

```bash
sqlite3 /Users/apple/v2/cboard.db "SELECT order_no, status, payment_time FROM orders WHERE order_no='YOUR_ORDER_NO';"
```

**期望结果：**
- `status` = `paid`
- `payment_time` 有值

## 常见问题

### Q1: 为什么本地测试回调失败？
**A:** 支付宝无法访问 localhost，需要使用 ngrok 或在生产环境测试。

### Q2: ngrok URL 每次都变化怎么办？
**A:**
- 使用 ngrok 付费版获得固定域名
- 或者每次启动时更新数据库配置
- 或者直接在生产环境测试

### Q3: 如何确认回调地址配置正确？
**A:**
```bash
# 1. 检查数据库配置
sqlite3 cboard.db "SELECT value FROM system_configs WHERE key='pay_alipay_notify_url';"

# 2. 测试端点可访问性
curl -X POST YOUR_CALLBACK_URL -d "test=1"
# 应该返回 "verify fail" 而不是 404
```

### Q4: 已支付的订单如何手动完成？
**A:**
```bash
./complete_pending_orders.sh
```

## 总结

- ✅ 代码修复已完成（transaction_id 正确设置为订单号）
- ✅ 待处理订单已手动完成
- ⚠️ 本地测试需要使用 ngrok 或在生产环境测试
- ✅ 生产环境回调配置正确，可以正常工作

**建议：**
- 本地开发时使用 ngrok 进行支付测试
- 或者在生产环境进行支付功能测试
- 保留手动完成订单的脚本用于紧急情况
