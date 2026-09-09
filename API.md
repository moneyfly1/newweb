# CBoard API 文档

> 速查文档：完整路由以代码为准（`internal/api/router/router.go`）。
> 所有 `data` 字段均可按需增删，本文仅列核心。

## 认证

登录后所有请求需携带 Token（`/api/v1/auth/*` 与公开接口除外）：

```
Authorization: Bearer {access_token}
```

CSRF：所有 POST/PUT/PATCH/DELETE（除 `/auth/*`）还需在 Header 携带 `X-CSRF-Token`。
获取方式：`GET /api/v1/csrf-token`（需登录）。Token 单次有效，后端每次校验成功后轮换。

## 统一响应结构

```json
{ "code": 0, "message": "success", "data": { ... } }
```

分页接口返回：`data: { "items": [...], "total": 100, "page": 1, "page_size": 20 }`

## 认证接口

```
POST /api/v1/auth/login
Body: { "email": "...", "password": "..." }
Resp data: { "access_token": "...", "refresh_token": "...", "user": { ... } }  # user 为当前用户信息

POST /api/v1/auth/refresh
Body: { "refresh_token": "..." }
Resp data: { "access_token": "...", "refresh_token": "..." }

POST /api/v1/auth/logout            # 需登录
POST /api/v1/auth/register          # 注册（可能要求邮箱验证码，视站点配置）
```

## 用户接口（需登录）

```
GET  /api/v1/users/me               # 当前用户信息
PUT  /api/v1/users/me               # 修改资料
POST /api/v1/users/change-password  # 修改密码
GET  /api/v1/users/login-history    # 登录历史
GET  /api/v1/users/dashboard-info   # 仪表盘数据
POST /api/v1/users/bind-telegram    # 绑定 Telegram
POST /api/v1/users/unbind-telegram
```

## 工单（含附件，需登录）

```
GET    /api/v1/tickets?page=&page_size=       # 我的工单列表
POST   /api/v1/tickets                        # 创建工单
       Body: { "title": "...", "content": "...", "type": "other",
               "attachment_ids": [1,2] }      # attachment_ids 可选
GET    /api/v1/tickets/:id                    # 工单详情（含 replies、attachments）
POST   /api/v1/tickets/:id/reply              # 回复
       Body: { "content": "...", "attachment_ids": [1] }
PUT    /api/v1/tickets/:id                    # 关闭工单 { "status": "closed" }

# 附件（图片/视频/文档）
POST   /api/v1/tickets/attachments            # 上传（multipart，字段 files，可多文件，≤5 个，
                                               # 单文件 ≤20MB；先落为待绑定）
DELETE /api/v1/tickets/attachments/:attId     # 删除未绑定附件（仅上传者本人）
GET    /api/v1/tickets/:id/attachments/:attId # 下载/预览（工单所有者或管理员；图片/视频/pdf 内联）
```

上传成功响应 `data.files` 数组含 `{ id, file_name, file_type, file_size }`；
创建工单/回复时把这些 `id` 放入 `attachment_ids` 即完成绑定（附件随后出现在详情与管理员后台）。

## 订阅 / 订单

```
GET  /api/v1/subscriptions/user-subscription   # 我的订阅
GET  /api/v1/subscriptions/devices             # 设备列表
POST /api/v1/subscriptions/reset-subscription  # 重置订阅
POST /api/v1/orders                            # 创建订单
     Body: { "package_id": 1, "coupon_code": "可选" }
     Resp data: 完整订单对象（含 order_no、amount、status 等）
GET  /api/v1/orders?page=&page_size=           # 订单列表（分页结构）
POST /api/v1/orders/:orderNo/pay               # 发起支付
POST /api/v1/orders/:orderNo/cancel
GET  /api/v1/orders/:orderNo/status
```

## 管理接口（需管理员）

```
GET    /api/v1/admin/dashboard            # 仪表盘
GET    /api/v1/admin/tickets              # 工单列表
GET    /api/v1/admin/tickets/:id          # 工单详情（含 replies、attachments，可看用户上传的附件）
PUT    /api/v1/admin/tickets/:id          # 更新状态/优先级/分配
POST   /api/v1/admin/tickets/:id/reply    # 回复（Body 同上，可带 attachment_ids）
POST   /api/v1/admin/tickets/attachments  # 管理员上传附件（与用户端同端点逻辑）
GET    /api/v1/admin/users                # 用户列表（分页）
POST   /api/v1/admin/users                # 创建用户
PUT    /api/v1/admin/users/:id            # 更新用户
DELETE /api/v1/admin/users/:id            # 删除用户
DELETE /api/v1/admin/users/:id/full       # 完全删除（连数据）
GET    /api/v1/admin/orders               # 订单管理
GET    /api/v1/admin/stats/*              # 各类统计
GET    /api/v1/admin/logs/*               # 各类日志
```

> 管理端附件下载与预览复用用户端端点
> `GET /api/v1/tickets/:id/attachments/:attId`（管理员对任意工单有权限）。

## 错误码

| code | 含义 |
|------|------|
| 0     | 成功 |
| 40000 | 请求参数错误 |
| 40100 | 未登录 / Token 失效 |
| 40300 | 无权限 / CSRF 无效 |
| 40400 | 资源不存在 |
| 40900 | 资源冲突（如重复操作） |
| 42900 | 请求过于频繁 |
| 50000 | 服务器错误 |
