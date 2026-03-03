# VPS 服务器更新指南

## 更新步骤

### 1. 备份数据库（重要！）
```bash
cd /root/cboard
cp cboard.db cboard.db.backup.$(date +%Y%m%d_%H%M%S)
```

### 2. 停止服务
```bash
pkill cboard
# 或者使用 systemd
systemctl stop cboard
```

### 3. 拉取最新代码
```bash
git pull origin main
```

### 4. 执行数据库迁移
```bash
sqlite3 cboard.db < migrations/add_payment_nonces.sql
```

### 5. 重新编译
```bash
go build -o cboard cmd/server/main.go
```

### 6. 启动服务
```bash
# 使用 nohup
nohup ./cboard > logs/app.log 2>&1 &

# 或者使用 systemd
systemctl start cboard
```

### 7. 验证服务
```bash
# 检查进程
ps aux | grep cboard

# 检查日志
tail -f logs/app.log

# 测试 API
curl http://localhost:9000/api/v1/config
```

### 8. 验证数据库迁移
```bash
sqlite3 cboard.db "SELECT COUNT(*) FROM payment_nonces;"
```

## 本次更新内容

### 修复的问题
- ✅ 修复支付宝回调无法匹配订单导致支付状态不更新
- ✅ 创建 payment_nonces 表防止重放攻击
- ✅ 优化回调查找逻辑，添加详细日志
- ✅ 修复同步回调 URL 处理

### 关键变更
1. **支付创建**: 使用 `PAY` 开头的 txID 作为 out_trade_no
2. **回调处理**: 通过 transaction_id 直接匹配支付事务
3. **日志增强**: 添加详细的调试日志便于追踪
4. **防重放**: 使用 payment_nonces 表防止重复处理

### 测试要点
- 创建新订单，transaction_id 应为 `PAY` 开头
- 支付后检查日志是否有回调记录
- 确认订单状态从 pending 变为 paid
- 确认订阅已激活

## 回滚方案

如果更新后出现问题：

```bash
# 1. 停止服务
pkill cboard

# 2. 恢复数据库
cp cboard.db.backup.YYYYMMDD_HHMMSS cboard.db

# 3. 回退代码
git reset --hard HEAD~1

# 4. 重新编译
go build -o cboard cmd/server/main.go

# 5. 启动服务
nohup ./cboard > logs/app.log 2>&1 &
```

## 注意事项

1. **不要删除数据库** - 只需执行迁移脚本添加新表
2. **保留旧订单** - 旧订单的 transaction_id 仍为 order_no 格式，不影响使用
3. **新订单生效** - 只有更新后创建的订单才使用新的 txID 格式
4. **日志监控** - 更新后建议监控日志确认支付流程正常

## 常见问题

### Q: 更新后旧订单还能查询吗？
A: 可以，旧订单不受影响。

### Q: 需要清空数据库吗？
A: 不需要！只需执行迁移脚本添加新表。

### Q: 如何确认更新成功？
A: 创建新订单，检查 payment_transactions 表的 transaction_id 字段是否为 PAY 开头。

### Q: 回调还是不工作怎么办？
A: 检查：
1. 支付宝开放平台的回调地址配置
2. 服务器防火墙是否开放
3. nginx 配置是否正确转发
4. 查看 logs/app.log 是否有回调日志
