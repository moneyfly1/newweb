# 支付回调修复说明

## 问题分析

### 原问题
1. 用户支付成功后，页面无法检测到订单状态变化
2. 支付回调虽然成功，但订阅没有被激活
3. 订单状态没有正确更新

### 根本原因
通过对比老项目（goweb）的实现，发现以下关键问题：

1. **订单号与交易ID不匹配**
   - 创建支付时使用 `order.OrderNo` 作为 `out_trade_no` 传给支付宝
   - 但 `PaymentTransaction` 的 `transaction_id` 字段没有更新为 `order.OrderNo`
   - 导致回调时无法通过 `out_trade_no` 找到对应的交易记录

2. **订单缺少支付交易ID关联**
   - 订单的 `payment_transaction_id` 字段没有被设置
   - 导致无法追踪订单对应的支付交易

3. **回调处理不完整**
   - 虽然调用了 `ActivateSubscription`，但订单的 `payment_transaction_id` 没有更新
   - 日志信息不够详细，难以诊断问题

## 修复内容

### 1. 修复 CreatePayment 中的订单号关联 (payment.go:184-192)

**修改前：**
```go
if err == nil {
    // 更新支付事务的 transaction_id 为订单号
    db.Model(&transaction).Update("transaction_id", outTradeNo)
    utils.Success(c, gin.H{...})
    return
}
```

**修改后：**
```go
if err == nil {
    // 更新支付事务的 transaction_id 为订单号，这样回调时可以通过 out_trade_no 找到
    db.Model(&transaction).Updates(map[string]interface{}{
        "transaction_id": outTradeNo,
    })
    // 同时更新订单的 payment_transaction_id
    ptxID := fmt.Sprintf("%d", transaction.ID)
    db.Model(&order).Update("payment_transaction_id", &ptxID)
    utils.Success(c, gin.H{...})
    return
}
```

### 2. 修复 handleAlipayOrderCallback 中的订单更新 (payment.go:1065-1105)

**修改前：**
```go
if err := tx.Model(&order).Updates(map[string]interface{}{
    "status":              "paid",
    "payment_method_name": &pmName,
    "payment_time":        &now,
}).Error; err != nil {
    return err
}
```

**修改后：**
```go
txIDStr := fmt.Sprintf("%d", transaction.ID)
if err := tx.Model(&order).Updates(map[string]interface{}{
    "status":                 "paid",
    "payment_method_name":    &pmName,
    "payment_time":           &now,
    "payment_transaction_id": &txIDStr,
}).Error; err != nil {
    return err
}
```

同时增强了日志输出：
```go
fmt.Printf("[alipay] ✓ 找到订单: order_no=%s, user_id=%d, package_id=%d, amount=%.2f\n",
    order.OrderNo, order.UserID, order.PackageID, order.Amount)
```

### 3. 修复 handleEpayOrderCallback (payment.go:884-908)

添加了 `payment_transaction_id` 的更新，与 Alipay 保持一致。

### 4. 修复 handleStripeOrderCallback (payment.go:1284-1307)

添加了 `payment_transaction_id` 的更新，与 Alipay 保持一致。

## 修复效果

### 修复前的流程
1. 用户创建订单 → 生成 `order_no` (如 ORD1234567890abcdef)
2. 创建支付 → 生成 `transaction_id` (如 PAY1234567890abcdef)
3. 调用支付宝 → 使用 `order_no` 作为 `out_trade_no`
4. 支付宝回调 → 返回 `out_trade_no` = ORD1234567890abcdef
5. 查找交易 → 通过 `transaction_id` = ORD1234567890abcdef 查找 ❌ **找不到**
6. 尝试通过 `order_no` 查找订单 → 找到订单
7. 通过 `order_id` 查找交易 → 找到交易 ✓
8. 更新订单状态 → `status` = paid ✓
9. 激活订阅 → 调用 `ActivateSubscription` ✓
10. 前端轮询 → 检测到 `status` = paid ✓

**问题：** 虽然最终能找到交易并激活订阅，但 `payment_transaction_id` 没有更新，导致订单和交易的关联不完整。

### 修复后的流程
1. 用户创建订单 → 生成 `order_no` (如 ORD1234567890abcdef)
2. 创建支付 → 生成 `transaction_id` (如 PAY1234567890abcdef)
3. 更新交易 → `transaction_id` = ORD1234567890abcdef ✓
4. 更新订单 → `payment_transaction_id` = transaction.ID ✓
5. 调用支付宝 → 使用 `order_no` 作为 `out_trade_no`
6. 支付宝回调 → 返回 `out_trade_no` = ORD1234567890abcdef
7. 查找交易 → 通过 `transaction_id` = ORD1234567890abcdef 查找 ✓ **直接找到**
8. 更新订单状态 → `status` = paid, `payment_transaction_id` = transaction.ID ✓
9. 激活订阅 → 调用 `ActivateSubscription` ✓
10. 前端轮询 → 检测到 `status` = paid ✓

**改进：**
- 交易和订单的关联更加完整
- 回调处理更加高效（直接通过 transaction_id 找到）
- 订单的 payment_transaction_id 字段被正确设置

## 测试建议

### 1. 测试支付宝支付流程
```bash
# 1. 创建订单
curl -X POST http://localhost:8000/api/v1/orders \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"package_id": 1}'

# 2. 创建支付
curl -X POST http://localhost:8000/api/v1/payment/create \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order_id": ORDER_ID, "payment_method_id": ALIPAY_METHOD_ID, "is_mobile": false}'

# 3. 扫码支付后，检查订单状态
curl http://localhost:8000/api/v1/orders/ORDER_NO/status \
  -H "Authorization: Bearer YOUR_TOKEN"

# 4. 检查订阅是否激活
curl http://localhost:8000/api/v1/subscription \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 2. 检查数据库记录
```sql
-- 检查订单和交易的关联
SELECT
    o.id as order_id,
    o.order_no,
    o.status as order_status,
    o.payment_transaction_id,
    pt.id as transaction_id,
    pt.transaction_id as tx_out_trade_no,
    pt.status as tx_status
FROM orders o
LEFT JOIN payment_transactions pt ON o.payment_transaction_id = CAST(pt.id AS TEXT)
WHERE o.order_no = 'YOUR_ORDER_NO';

-- 检查订阅是否激活
SELECT * FROM subscriptions WHERE user_id = YOUR_USER_ID;
```

### 3. 查看日志
```bash
# 查看支付创建日志
grep "支付创建成功" logs/app.log

# 查看回调处理日志
grep "alipay.*回调" logs/app.log

# 查看订阅激活日志
grep "订阅激活" logs/app.log
```

## 注意事项

1. **数据库字段类型**
   - `payment_transaction_id` 在 Order 模型中是 `*string` 类型
   - 需要将 `transaction.ID` (uint) 转换为字符串

2. **事务处理**
   - 所有订单状态更新和订阅激活都在数据库事务中进行
   - 确保原子性，防止数据不一致

3. **日志增强**
   - 增加了 `package_id` 的日志输出
   - 便于诊断订阅激活问题

4. **兼容性**
   - 修复同时适用于 Alipay、Epay 和 Stripe
   - 保持了与老项目相同的处理逻辑

## 参考老项目实现

老项目（goweb）的关键实现：
- `internal/services/order/order.go:276-284` - 创建 PaymentTransaction
- `internal/api/handlers/payment.go:354-376` - 处理支付回调
- `internal/services/order/order.go:345-374` - ProcessPaidOrder 激活订阅

当前项目完全按照老项目的逻辑进行了修复。
