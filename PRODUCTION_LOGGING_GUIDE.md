# 生产环境日志系统使用指南

## 已添加的日志功能

### 1. 日志文件位置
```
logs/app-2026-03-03.log  # 按日期自动创建
```

### 2. 日志类型

- **[INFO]** - 一般信息
- **[ERROR]** - 错误信息
- **[WARN]** - 警告信息
- **[DEBUG]** - 调试信息
- **[PAYMENT]** - 支付相关操作
- **[CALLBACK]** - 支付回调
- **[ORDER]** - 订单操作

### 3. 记录的关键信息

#### 支付创建
```
[PAYMENT] [CreatePayment] 开始创建支付 - user_id=1, order_id=123, payment_method_id=1
[PAYMENT] [CreatePayment] 找到订单 - order_no=ORD123, package_id=1, amount=9.99
[PAYMENT] [CreatePayment] ✅ 支付宝支付创建成功
  - order_no: ORD123
  - transaction_id: ORD123
  - payment_url: https://...
  - amount: 9.99
  - notify_url: https://go.moneyfly.top/api/v1/payment/notify/alipay
  - return_url: https://go.moneyfly.top/payment/return
```

#### 支付回调
```
[CALLBACK] ==========================================
[CALLBACK] 收到支付回调请求
[CALLBACK]   Method: POST
[CALLBACK]   Path: /api/v1/payment/notify/alipay
[CALLBACK]   Client IP: 110.75.xxx.xxx
[CALLBACK]   User-Agent: Mozilla/5.0...
[CALLBACK]   Content-Type: application/x-www-form-urlencoded
[CALLBACK]   Body: out_trade_no=ORD123&trade_no=2026...&trade_status=TRADE_SUCCESS...
[CALLBACK] ==========================================
[CALLBACK] [Alipay] ✅ 回调验证成功
[CALLBACK]   - out_trade_no: ORD123
[CALLBACK]   - trade_no: 2026030322001234567890
[CALLBACK]   - trade_status: TRADE_SUCCESS
[CALLBACK]   - total_amount: 9.99
[CALLBACK] 回调处理完成
[CALLBACK]   Status: 200
[CALLBACK]   Latency: 123ms
[CALLBACK] ==========================================
```

## 部署到生产环境

### 1. 上传新代码到服务器

```bash
# 在本地
git add -A
git commit -m "feat: 添加完整的日志系统"
git push origin main

# 在服务器上
cd /path/to/your/project
git pull origin main
```

### 2. 编译新版本

```bash
go build -o cboard-server cmd/server/main.go
```

### 3. 停止旧服务

```bash
# 查找进程
ps aux | grep cboard

# 停止进程
pkill -f cboard
# 或者
kill -9 <PID>
```

### 4. 启动新服务

```bash
# 方式1：直接启动（会在终端显示日志）
./cboard-server

# 方式2：后台启动
nohup ./cboard-server > /dev/null 2>&1 &

# 方式3：使用 systemd（推荐）
sudo systemctl restart cboard
```

## 查看日志

### 实时查看日志

```bash
# 实时查看所有日志
tail -f logs/app-$(date +%Y-%m-%d).log

# 只看支付相关
tail -f logs/app-$(date +%Y-%m-%d).log | grep PAYMENT

# 只看回调相关
tail -f logs/app-$(date +%Y-%m-%d).log | grep CALLBACK

# 只看错误
tail -f logs/app-$(date +%Y-%m-%d).log | grep ERROR
```

### 查看历史日志

```bash
# 查看最近100行
tail -100 logs/app-$(date +%Y-%m-%d).log

# 查看最近的支付日志
tail -500 logs/app-$(date +%Y-%m-%d).log | grep PAYMENT

# 查看特定订单的日志
grep "ORD1772507117" logs/app-$(date +%Y-%m-%d).log

# 查看最近的回调
grep "收到支付回调" logs/app-$(date +%Y-%m-%d).log -A 20
```

### 搜索特定信息

```bash
# 搜索特定订单号
grep "ORD1772507117" logs/app-*.log

# 搜索支付失败的记录
grep "支付.*失败" logs/app-*.log

# 搜索回调验证失败
grep "回调验证失败" logs/app-*.log
```

## 诊断支付问题

### 步骤1：查看订单创建日志

```bash
# 查找订单创建记录
grep "CreatePayment.*开始创建支付" logs/app-$(date +%Y-%m-%d).log | tail -5

# 查看完整的支付创建过程
grep "ORD1772507117" logs/app-$(date +%Y-%m-%d).log
```

### 步骤2：检查是否收到回调

```bash
# 查看今天是否有回调
grep "收到支付回调请求" logs/app-$(date +%Y-%m-%d).log

# 查看特定订单的回调
grep "out_trade_no.*ORD1772507117" logs/app-$(date +%Y-%m-%d).log
```

### 步骤3：检查回调处理结果

```bash
# 查看回调处理状态
grep "回调处理完成" logs/app-$(date +%Y-%m-%d).log | tail -10

# 查看回调错误
grep "CALLBACK.*ERROR" logs/app-$(date +%Y-%m-%d).log
```

## 常见问题诊断

### 问题1：支付后订单状态不更新

**检查步骤：**

```bash
# 1. 确认支付创建成功
grep "支付宝支付创建成功" logs/app-$(date +%Y-%m-%d).log | tail -1

# 2. 检查 notify_url 配置
grep "notify_url:" logs/app-$(date +%Y-%m-%d).log | tail -1

# 3. 检查是否收到回调
grep "收到支付回调请求" logs/app-$(date +%Y-%m-%d).log | tail -5

# 4. 如果没有回调，检查回调地址是否可访问
curl -X POST https://go.moneyfly.top/api/v1/payment/notify/alipay -d "test=1"
```

### 问题2：回调验证失败

```bash
# 查看验证失败的原因
grep "回调验证失败" logs/app-$(date +%Y-%m-%d).log -A 5
```

### 问题3：订单找不到

```bash
# 查看订单查找日志
grep "找到订单\|订单不存在" logs/app-$(date +%Y-%m-%d).log | tail -10
```

## 发送日志给开发者

### 方式1：复制最近的日志

```bash
# 复制最近500行日志
tail -500 logs/app-$(date +%Y-%m-%d).log > /tmp/debug.log

# 下载到本地
scp user@server:/tmp/debug.log ./
```

### 方式2：只发送相关日志

```bash
# 提取特定订单的所有日志
grep "ORD1772507117" logs/app-$(date +%Y-%m-%d).log > /tmp/order_debug.log

# 提取最近的支付和回调日志
tail -1000 logs/app-$(date +%Y-%m-%d).log | grep -E "PAYMENT|CALLBACK" > /tmp/payment_debug.log
```

### 方式3：在线查看

```bash
# 创建一个临时的 web 服务器来查看日志
python3 -m http.server 8888 --directory logs/

# 然后在浏览器访问
# http://your-server-ip:8888/app-2026-03-03.log
```

## 日志维护

### 清理旧日志

```bash
# 删除7天前的日志
find logs/ -name "app-*.log" -mtime +7 -delete

# 压缩旧日志
find logs/ -name "app-*.log" -mtime +1 -exec gzip {} \;
```

### 日志轮转（可选）

创建 `/etc/logrotate.d/cboard`:

```
/path/to/your/project/logs/app-*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

## 测试日志系统

### 1. 测试支付创建日志

```bash
# 创建一个测试订单并支付
# 然后查看日志
tail -50 logs/app-$(date +%Y-%m-%d).log | grep PAYMENT
```

### 2. 测试回调日志

```bash
# 发送测试回调
curl -X POST https://go.moneyfly.top/api/v1/payment/notify/alipay \
  -d "test=1" \
  -H "Content-Type: application/x-www-form-urlencoded"

# 查看日志
tail -20 logs/app-$(date +%Y-%m-%d).log | grep CALLBACK
```

## 快速诊断脚本

创建 `check_payment.sh`:

```bash
#!/bin/bash

ORDER_NO=$1

if [ -z "$ORDER_NO" ]; then
    echo "用法: ./check_payment.sh ORDER_NO"
    exit 1
fi

echo "=========================================="
echo "诊断订单: $ORDER_NO"
echo "=========================================="

echo ""
echo "1. 支付创建记录:"
grep "$ORDER_NO" logs/app-*.log | grep "CreatePayment"

echo ""
echo "2. 回调记录:"
grep "$ORDER_NO" logs/app-*.log | grep "CALLBACK"

echo ""
echo "3. 错误记录:"
grep "$ORDER_NO" logs/app-*.log | grep "ERROR"

echo ""
echo "4. 数据库状态:"
sqlite3 cboard.db "SELECT id, order_no, status, payment_time FROM orders WHERE order_no='$ORDER_NO';"
```

使用方法：

```bash
chmod +x check_payment.sh
./check_payment.sh ORD1772507117
```
