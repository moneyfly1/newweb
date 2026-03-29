# CBoard API 文档

## 认证

所有 API 请求需要在 Header 中携带 Token：
```
Authorization: Bearer {token}
```

## 用户相关

### 登录
```
POST /api/v1/auth/login
Body: { "email": "user@example.com", "password": "password" }
Response: { "code": 0, "data": { "access_token": "...", "refresh_token": "..." } }
```

### 获取用户信息
```
GET /api/v1/user/info
Response: { "code": 0, "data": { "id": 1, "username": "...", ... } }
```

## 订单相关

### 创建订单
```
POST /api/v1/orders
Body: { "package_id": 1, "coupon_code": "..." }
Response: { "code": 0, "data": { "order_no": "...", "amount": 100 } }
```

### 订单列表
```
GET /api/v1/orders?page=1&page_size=20
Response: { "code": 0, "data": [...], "total": 100 }
```

## 管理员相关

### 用户管理
```
GET /api/v1/admin/users?page=1&page_size=20
POST /api/v1/admin/users
PUT /api/v1/admin/users/:id
DELETE /api/v1/admin/users/:id
```

## 错误码

- 0: 成功
- 40000: 请求参数错误
- 40100: 未登录
- 40300: 无权限
- 42900: 请求过于频繁
- 50000: 服务器错误
