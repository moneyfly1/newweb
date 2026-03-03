# 支付回调问题诊断报告

## 问题现状

**订单信息：**
- 订单号：ORD17725062420taFVX
- 订单状态：pending（未支付）
- 支付金额：0.05元
- 创建时间：2026-03-03 10:50:42

**支付交易信息：**
- 交易ID：23
- transaction_id：PAY1772506247IBxk3Dgg（❌ 错误，应该是订单号）
- 状态：pending
- 外部交易号：空（说明回调未到达）

**回调记录：**
- 无任何回调记录（payment_callbacks 表为空）

## 问题分析

### 1. 服务器未重启（次要问题）
- 当前运行的是旧代码（进程启动于 18:19）
- 修复代码已提交但未生效
- **需要重启服务器**

### 2. 支付回调未到达（主要问题）
- 测试显示服务器可以从外网访问
- 路由配置正确（POST /api/v1/payment/notify/alipay 返回 "verify fail"）
- 但实际支付后没有收到回调

**可能原因：**
1. 支付宝开放平台应用配置中的"授权回调地址"未设置或设置错误
2. 支付时传递的 notify_url 参数被支付宝忽略
3. 支付宝沙箱环境的回调有延迟或不稳定

## 解决方案

### 方案1：检查支付宝开放平台配置（推荐）

1. 登录支付宝开放平台：https://open.alipay.com/
2. 进入你的应用（AppID: 2021005145632109）
3. 找到"接口加签方式（密钥/证书）"或"应用信息"
4. 查看"授权回调地址"或"网关地址"配置
5. 确保设置为：`https://go.moneyfly.top/api/v1/payment/notify/alipay`

### 方案2：手动触发订单完成（临时解决）

如果你已经支付成功，可以手动更新数据库：

```bash
# 1. 更新订单状态
sqlite3 /Users/apple/v2/cboard.db "UPDATE orders SET status='paid', payment_time=datetime('now'), payment_method_name='支付宝' WHERE order_no='ORD17725062420taFVX';"

# 2. 更新支付交易
sqlite3 /Users/apple/v2/cboard.db "UPDATE payment_transactions SET status='paid', external_transaction_id='MANUAL_2026030310' WHERE id=23;"

# 3. 手动激活订阅（需要重启服务器后执行）
# 重启服务器后，访问管理后台手动激活订阅
```

### 方案3：重启服务器并重新测试

```bash
# 1. 停止旧服务器
pkill -f cboard

# 2. 启动新服务器（在终端中运行，可以看到日志）
cd /Users/apple/v2
./cboard-server

# 3. 创建新订单并支付
# 4. 观察服务器终端是否有回调日志
```

## 验证步骤

### 1. 验证服务器可以接收回调

```bash
chmod +x /Users/apple/v2/test_callback.sh
/Users/apple/v2/test_callback.sh
```

预期结果：返回 "verify fail"（说明服务器收到了请求）

### 2. 验证支付宝配置

在支付宝开放平台查看：
- 应用网关地址
- 授权回调地址
- 是否启用了异步通知

### 3. 查看支付宝交易详情

登录支付宝商家中心，查看交易详情：
- 是否显示"通知商户"
- 通知状态是什么
- 通知地址是什么

## 关键配置检查清单

- [ ] 服务器已重启（使用新代码）
- [ ] 支付宝开放平台配置了正确的回调地址
- [ ] 回调地址可以从外网访问（已验证 ✓）
- [ ] 路由配置正确（已验证 ✓）
- [ ] 支付宝应用已上线（不是沙箱环境）

## 下一步行动

**立即执行：**
1. 重启服务器
2. 检查支付宝开放平台配置
3. 如果已支付，使用方案2手动完成订单

**长期解决：**
1. 在支付宝开放平台正确配置回调地址
2. 测试新订单的支付流程
3. 确认回调能正常到达

## 调试命令

```bash
# 查看最近的订单
sqlite3 /Users/apple/v2/cboard.db "SELECT id, order_no, status, datetime(created_at, 'localtime') FROM orders ORDER BY id DESC LIMIT 5;"

# 查看支付交易
sqlite3 /Users/apple/v2/cboard.db "SELECT id, order_id, transaction_id, status, external_transaction_id FROM payment_transactions ORDER BY id DESC LIMIT 5;"

# 查看回调记录
sqlite3 /Users/apple/v2/cboard.db "SELECT id, callback_type, processed, datetime(created_at, 'localtime') FROM payment_callbacks ORDER BY id DESC LIMIT 5;"

# 测试回调端点
curl -X POST https://go.moneyfly.top/api/v1/payment/notify/alipay -d "test=1"
```
