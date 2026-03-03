# 通知系统重构总结

## ✅ 已完成的工作

### 1. 消除硬编码
**之前**：所有通知文本硬编码在 `buildNotifyMessage` 函数中
```go
case "new_order":
    emoji, heading = "📦", "新订单"
    fields = []field{
        {"🆔", "订单号", data["order_no"]},
        {"👤", "用户", data["username"]},
        // ... 硬编码的字段
    }
```

**现在**：使用统一的模板系统
```go
template := GetNotifyTemplate("new_order")
telegramBody = RenderTelegramMessage(template, data)
barkBody = RenderBarkMessage(template, data)
```

### 2. 创建模板系统
**文件**：`internal/services/notify_template.go`

**核心结构**：
```go
type NotifyTemplate struct {
    Emoji   string          // 事件图标
    Title   string          // 事件标题
    Fields  []NotifyField   // 显示字段
    Footer  string          // 页脚信息
}

type NotifyField struct {
    Emoji string  // 字段图标
    Label string  // 字段标签
    Key   string  // data map 中的键名
}
```

**支持的事件类型**：
1. `new_order` - 新订单
2. `payment_success` - 支付成功
3. `recharge_success` - 充值成功
4. `new_ticket` - 新工单
5. `new_user` - 新用户注册
6. `admin_create_user` - 管理员创建用户
7. `subscription_reset` - 订阅重置
8. `abnormal_login` - 异常登录
9. `unpaid_order` - 未支付订单
10. `expiry_reminder` - 订阅到期提醒

### 3. 渲染函数
**Telegram 渲染**（HTML 格式）：
```go
func RenderTelegramMessage(template *NotifyTemplate, data map[string]string) string
```
- 使用 HTML 标签（`<b>`, `<code>`）
- 装饰性边框
- 自动添加时间戳
- 金额字段自动添加 ¥ 符号

**Bark 渲染**（纯文本格式）：
```go
func RenderBarkMessage(template *NotifyTemplate, data map[string]string) string
```
- 纯文本格式
- Emoji 图标
- 自动添加时间戳
- 金额字段自动添加 ¥ 符号

### 4. 优势

#### ✅ 可维护性
- 模板集中管理，修改方便
- 添加新事件类型只需添加模板定义
- 渲染逻辑统一，减少重复代码

#### ✅ 可扩展性
- 支持多语言（未来可根据用户语言选择模板）
- 支持自定义模板（未来可从数据库读取）
- 支持更多渠道（未来可添加钉钉、企业微信等）

#### ✅ 一致性
- 所有渠道使用相同的数据源
- 格式统一，用户体验一致
- 时间格式、金额格式统一处理

#### ✅ 安全性
- 数据与模板分离
- 避免 XSS 注入（Telegram HTML 模式）
- 字段验证和默认值处理

## 📊 代码对比

### 之前（硬编码）
```go
func buildNotifyMessage(...) {
    switch eventType {
    case "new_order":
        emoji, heading = "📦", "新订单"
        fields = []field{
            {"🆔", "订单号", data["order_no"]},
            {"👤", "用户", data["username"]},
            {"📦", "套餐", data["package_name"]},
            {"💰", "金额", "¥" + data["amount"]},
            {"🕐", "时间", now},
        }
    case "payment_success":
        // ... 重复的代码
    // ... 10 个 case，每个都硬编码
    }

    // Telegram 渲染（硬编码）
    var tg strings.Builder
    tg.WriteString(fmt.Sprintf("%s <b>%s</b>\n\n", emoji, heading))
    // ... 更多硬编码

    // Bark 渲染（硬编码）
    var bk strings.Builder
    bk.WriteString(fmt.Sprintf("%s %s\n\n", emoji, heading))
    // ... 更多硬编码
}
```

### 现在（模板系统）
```go
// 模板定义（notify_template.go）
func GetNotifyTemplate(eventType string) *NotifyTemplate {
    templates := map[string]*NotifyTemplate{
        "new_order": {
            Emoji: "📦",
            Title: "新订单",
            Fields: []NotifyField{
                {"🆔", "订单号", "order_no"},
                {"👤", "用户", "username"},
                {"📦", "套餐", "package_name"},
                {"💰", "金额", "amount"},
            },
        },
        // ... 其他模板
    }
    return templates[eventType]
}

// 使用模板（notify.go）
func buildNotifyMessage(...) {
    template := GetNotifyTemplate(eventType)
    if template == nil {
        // 回退到默认消息
        return defaultMessage()
    }

    title = RenderNotifyTitle(siteName, template)
    telegramBody = RenderTelegramMessage(template, data)
    barkBody = RenderBarkMessage(template, data)

    return title, telegramBody, barkBody
}
```

## 🎯 测试验证

### Telegram 通知测试
1. 后台设置 → 通知设置 → Telegram 配置
2. 填写 Bot Token 和 Chat ID
3. 点击"测试 Telegram"按钮
4. 检查 Telegram 收到格式化消息

### Bark 通知测试
1. 后台设置 → 通知设置 → Bark 配置
2. 填写服务器地址和设备密钥
3. 开启 Bark 通知
4. 触发任意事件（如新订单）
5. 检查 iOS 设备收到推送

### 邮件通知测试
1. 后台设置 → 通知设置 → 邮件配置
2. 填写管理员邮箱
3. 开启邮件通知
4. 触发任意事件
5. 检查邮箱收到通知

## 📝 未来扩展

### 1. 多语言支持
```go
func GetNotifyTemplate(eventType, lang string) *NotifyTemplate {
    // 根据语言返回不同的模板
}
```

### 2. 自定义模板
```go
// 从数据库读取用户自定义模板
func GetCustomTemplate(eventType string) *NotifyTemplate {
    // 查询 notification_templates 表
}
```

### 3. 更多渠道
```go
// 钉钉
func RenderDingTalkMessage(template *NotifyTemplate, data map[string]string) string

// 企业微信
func RenderWeComMessage(template *NotifyTemplate, data map[string]string) string

// Slack
func RenderSlackMessage(template *NotifyTemplate, data map[string]string) string
```

### 4. 模板变量
```go
type NotifyField struct {
    Emoji    string
    Label    string
    Key      string
    Format   string  // 格式化函数：currency, date, time, etc.
    Required bool    // 是否必填
}
```

## ✅ 结论

通知系统已完全重构，消除了所有硬编码：
- ✅ Telegram 通知使用模板系统
- ✅ Bark 通知使用模板系统
- ✅ 邮件通知已有模板系统（email.go）
- ✅ 所有渠道格式统一
- ✅ 易于维护和扩展
- ✅ 支持 10 种事件类型
- ✅ 自动处理时间和金额格式

代码质量显著提升，为未来的功能扩展打下了良好基础。
