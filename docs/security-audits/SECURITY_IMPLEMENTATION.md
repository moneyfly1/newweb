# 安全加固实施指南

## 立即实施（高优先级）

### 1. 应用数据库迁移

```bash
cd /Users/apple/v2
sqlite3 cboard.db < migrations/add_payment_nonces.sql
```

### 2. 重新编译并重启服务

```bash
# 停止当前服务
kill $(cat backend.pid)

# 重新编译
go build -o cboard cmd/server/main.go

# 重启服务
./start.sh
```

### 3. 在路由中添加 CSRF 保护

编辑 `/internal/api/router/router.go`，在敏感路由添加 CSRF 中间件：

```go
// 需要 CSRF 保护的路由组
protected := v1.Group("")
protected.Use(middleware.AuthRequired())
protected.Use(middleware.CSRFProtection())
{
    // 订单相关
    protected.POST("/orders", handlers.CreateOrder)
    protected.POST("/orders/custom", handlers.CreateCustomOrder)
    protected.POST("/orders/upgrade", handlers.CreateUpgradeOrder)
    protected.POST("/orders/:orderNo/pay", handlers.PayOrder)

    // 充值相关
    protected.POST("/recharge", handlers.CreateRecharge)
    protected.POST("/recharge/:id/payment", handlers.CreateRechargePayment)

    // 订阅相关
    protected.POST("/subscription/reset", handlers.ResetSubscription)
    protected.POST("/subscription/convert", handlers.ConvertToBalance)
    protected.DELETE("/subscription/devices/:id", handlers.DeleteSubscriptionDevice)
}

// CSRF token 获取接口
v1.GET("/csrf-token", middleware.AuthRequired(), middleware.GetCSRFToken)
```

### 4. 前端添加 CSRF Token 支持

编辑 `/frontend/src/utils/request.ts`：

```typescript
import axios from 'axios'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

// 存储 CSRF token
let csrfToken: string | null = null

// 获取 CSRF token
export async function fetchCSRFToken() {
  try {
    const response = await request.get('/csrf-token')
    csrfToken = response.data.data.csrf_token
    return csrfToken
  } catch (error) {
    console.error('Failed to fetch CSRF token:', error)
    return null
  }
}

// 请求拦截器
request.interceptors.request.use(
  async (config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // 为 POST/PUT/DELETE/PATCH 请求添加 CSRF token
    if (['post', 'put', 'delete', 'patch'].includes(config.method?.toLowerCase() || '')) {
      if (!csrfToken) {
        await fetchCSRFToken()
      }
      if (csrfToken) {
        config.headers['X-CSRF-Token'] = csrfToken
      }
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response) => response,
  async (error) => {
    // CSRF token 失效时重新获取
    if (error.response?.status === 403 && error.response?.data?.message?.includes('CSRF')) {
      await fetchCSRFToken()
      // 重试原请求
      return request(error.config)
    }
    return Promise.reject(error)
  }
)

export default request
```

在 App.vue 中初始化 CSRF token：

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { fetchCSRFToken } from '@/utils/request'

onMounted(async () => {
  // 用户登录后获取 CSRF token
  const token = localStorage.getItem('access_token')
  if (token) {
    await fetchCSRFToken()
  }
})
</script>
```

---

## 推荐实施（中优先级）

### 5. 添加订阅访问频率限制

编辑 `/internal/api/router/router.go`：

```go
// 订阅路由添加严格的频率限制
subGroup := router.Group("/api/v1/sub")
subGroup.Use(middleware.RateLimit(60, 1*time.Minute)) // 每分钟最多 60 次
{
    subGroup.GET("/:url", handlers.GetSubscription)
    subGroup.GET("/clash/:url", handlers.GetSubscription)
    subGroup.GET("/:format/:url", handlers.GetSubscriptionByFormat)
}
```

### 6. 配置支付回调 IP 白名单

在系统设置中添加配置项：

```sql
-- 易支付回调 IP 白名单（多个 IP 用逗号分隔）
INSERT INTO system_configs (key, value, description)
VALUES ('pay_epay_callback_ips', '127.0.0.1,支付网关IP', '易支付回调 IP 白名单');

-- 支付宝回调 IP 白名单
INSERT INTO system_configs (key, value, description)
VALUES ('pay_alipay_callback_ips', '', '支付宝回调 IP 白名单（留空则不限制）');

-- Stripe 回调 IP 白名单
INSERT INTO system_configs (key, value, description)
VALUES ('pay_stripe_callback_ips', '', 'Stripe 回调 IP 白名单（留空则不限制）');
```

在 `payment.go` 中添加 IP 验证：

```go
func verifyCallbackIP(c *gin.Context, payType string) bool {
    clientIP := utils.GetRealClientIP(c)
    whitelist := utils.GetSetting(fmt.Sprintf("pay_%s_callback_ips", payType))

    if whitelist == "" {
        return true // 未配置白名单则允许所有 IP
    }

    allowedIPs := strings.Split(whitelist, ",")
    for _, ip := range allowedIPs {
        if strings.TrimSpace(ip) == clientIP {
            return true
        }
    }

    utils.SysError("payment", fmt.Sprintf("%s 回调 IP 不在白名单: %s", payType, clientIP))
    return false
}

// 在各个回调处理函数开头添加
func handleEpayNotify(c *gin.Context, db *gorm.DB) {
    if !verifyCallbackIP(c, "epay") {
        c.String(403, "IP not allowed")
        return
    }
    // ... 原有逻辑
}
```

### 7. 添加订单自动过期任务

编辑 `/internal/services/scheduler.go`：

```go
// CancelExpiredOrders 取消过期订单
func CancelExpiredOrders() {
    db := database.GetDB()

    result := db.Model(&models.Order{}).
        Where("status = ? AND expire_time < ?", "pending", time.Now()).
        Update("status", "expired")

    if result.RowsAffected > 0 {
        utils.SysInfo("scheduler", fmt.Sprintf("已取消 %d 个过期订单", result.RowsAffected))
    }
}

// 在 StartScheduler 中添加定时任务
func StartScheduler() {
    // ... 现有任务

    // 每小时检查一次过期订单
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        for range ticker.C {
            CancelExpiredOrders()
        }
    }()
}
```

---

## 可选实施（低优先级）

### 8. 前端敏感信息脱敏

编辑 `/frontend/src/views/subscription/Index.vue`：

```vue
<template>
  <div class="url-row">
    <span class="url-label">通用订阅</span>
    <div class="url-input-wrapper">
      <n-input
        :value="showFullUrl ? subscriptionUrl : maskedUrl"
        readonly
        size="small"
        class="url-input"
      />
      <n-button size="small" @click="showFullUrl = !showFullUrl">
        <template #icon>
          <n-icon :component="showFullUrl ? EyeOffOutline : EyeOutline" />
        </template>
      </n-button>
      <n-button size="small" type="primary" @click="copyToClipboard(subscriptionUrl, '通用订阅地址')">
        <template #icon><n-icon :component="CopyOutline" /></template>
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { EyeOutline, EyeOffOutline } from '@vicons/ionicons5'

const showFullUrl = ref(false)

const maskedUrl = computed(() => {
  if (!subscriptionUrl.value) return ''
  const url = subscriptionUrl.value
  const parts = url.split('/')
  const token = parts[parts.length - 1]
  if (token.length > 8) {
    return url.replace(token, token.substring(0, 4) + '****' + token.substring(token.length - 4))
  }
  return url
})
</script>
```

### 9. 配置 HTTPS 和安全头

Nginx 配置示例：

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # 安全头
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    location / {
        proxy_pass http://localhost:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

### 10. 配置日志监控告警

创建监控脚本 `/Users/apple/v2/scripts/security_monitor.sh`：

```bash
#!/bin/bash

LOG_FILE="/Users/apple/v2/backend.log"
ALERT_EMAIL="admin@yourdomain.com"

# 检查支付回调异常
PAYMENT_ERRORS=$(grep -c "金额不匹配\|重放攻击" "$LOG_FILE")
if [ "$PAYMENT_ERRORS" -gt 0 ]; then
    echo "检测到 $PAYMENT_ERRORS 个支付回调异常" | mail -s "支付安全告警" "$ALERT_EMAIL"
fi

# 检查订阅枚举尝试
SUB_ENUM=$(grep -c "订阅地址不存在访问尝试" "$LOG_FILE")
if [ "$SUB_ENUM" -gt 100 ]; then
    echo "检测到 $SUB_ENUM 次订阅枚举尝试" | mail -s "订阅安全告警" "$ALERT_EMAIL"
fi

# 检查登录失败次数
LOGIN_FAILS=$(grep -c "登录失败" "$LOG_FILE")
if [ "$LOGIN_FAILS" -gt 50 ]; then
    echo "检测到 $LOGIN_FAILS 次登录失败" | mail -s "登录安全告警" "$ALERT_EMAIL"
fi
```

添加到 crontab：

```bash
# 每小时执行一次安全监控
0 * * * * /Users/apple/v2/scripts/security_monitor.sh
```

---

## 验证清单

完成实施后，请验证以下功能：

- [ ] 支付回调重放攻击被成功拦截
- [ ] 金额不匹配的支付回调被拒绝
- [ ] CSRF token 验证正常工作
- [ ] 订阅访问频率限制生效
- [ ] Token 刷新后旧 token 失效
- [ ] 过期订单自动取消
- [ ] 日志正常记录安全事件
- [ ] HTTPS 正常工作
- [ ] 数据库定期备份

---

## 回滚方案

如果出现问题，可以快速回滚：

```bash
# 1. 停止服务
kill $(cat backend.pid)

# 2. 恢复旧版本
cp server cboard

# 3. 回滚数据库（如果需要）
sqlite3 cboard.db "DROP TABLE IF EXISTS payment_nonces;"

# 4. 重启服务
./start.sh
```

---

## 技术支持

如遇到问题，请检查：
1. 日志文件：`tail -f backend.log`
2. 数据库状态：`sqlite3 cboard.db ".tables"`
3. 服务状态：`ps aux | grep cboard`

**实施完成后请更新 SECURITY_FIXES.md 中的部署清单。**
