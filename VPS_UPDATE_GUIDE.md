# VPS 更新指南

## 方法一：快速更新（推荐）

如果您的 VPS 已经安装过，使用以下命令快速更新：

```bash
cd /root/v2
git pull origin main
sudo systemctl stop cboard
go build -o cboard ./cmd/server/main.go
sudo systemctl start cboard
sudo systemctl status cboard
```

## 方法二：完整重新安装

如果遇到问题，可以完整重新安装：

```bash
cd /root
rm -rf v2
git clone https://github.com/moneyfly1/newweb.git v2
cd v2
sudo bash install.sh
```

**注意**：重新安装会保留数据库文件 `cboard.db`，不会丢失数据。

## 本次更新内容

### 1. 修复支付回调问题
- 添加 `payment_nonces` 表到自动迁移
- 新部署的 VPS 会自动创建此表
- 修复支付回调防重放功能

### 2. 优化界面显示
- 修正支付宝设置页面文字（"应用私钥" → "商户私钥"）
- 移除硬编码域名显示
- 回调地址提示改为通用格式

### 3. 改进构建过程
- 修复 Go 1.24 兼容性问题
- 添加构建进度显示
- 优化依赖下载过程

## 验证更新成功

```bash
# 检查服务状态
sudo systemctl status cboard

# 查看日志
sudo journalctl -u cboard -f

# 检查数据库表
sqlite3 cboard.db "SELECT name FROM sqlite_master WHERE type='table' AND name='payment_nonces';"
```

如果看到 `payment_nonces` 表存在，说明更新成功。

## 常见问题

### Q: 更新后支付回调还是失败？
A: 检查以下几点：
1. 确认 `payment_nonces` 表已创建
2. 检查支付宝配置中的回调地址是否正确
3. 查看日志：`sudo journalctl -u cboard -f`

### Q: 构建很慢怎么办？
A: 设置国内代理加速：
```bash
export GOPROXY=https://goproxy.cn,direct
go build -o cboard ./cmd/server/main.go
```

### Q: 如何备份数据？
A:
```bash
cp cboard.db cboard.db.backup.$(date +%Y%m%d)
```
