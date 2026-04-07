# 🔘 CBoard 前端按钮详细功能文档

**生成日期:** 2026年4月2日  
**包含按钮数量**: 102个  
**覆盖页面**: 12个  
**文档类型**: 每个按钮的完整功能说明

---

## 📑 文档结构

- [用户端页面按钮](#用户端页面按钮) (15个)
- [管理端页面按钮](#管理端页面按钮) (87个)
- [通用模式](#通用模式)
- [API映射表](#api映射表)

---

## 用户端页面按钮

### 1️⃣ Dashboard (仪表盘) `/` - 11个按钮

#### 1.1 充值按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 充值 |
| **位置** | 账户余额卡片右上角 |
| **点击事件** | `router.push('/recharge')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到充值页面 |
| **是否禁用** | 否 |
| **加载状态** | 否 |
| **返回结果** | 打开 `/recharge` 页面 |

#### 1.2 签到按钮 ⭐
| 属性 | 值 |
|------|-----|
| **按钮文本** | 签到 |
| **位置** | 连续签到卡片 |
| **点击事件** | `handleCheckIn()` |
| **调用API** | `POST /api/user/checkin` |
| **API参数** | 无 |
| **返回数据结构** | `{ amount: number, consecutive_days: number }` |
| **禁用条件** | `checkinStatus.checked_in_today === true` (已签到) |
| **加载状态** | `checkinLoading === true` |
| **成功处理** | ✅ 显示成功提示<br>✅ 显示获得的奖励金额<br>✅ 更新账户余额<br>✅ 更新签到状态（禁用按钮） |
| **失败处理** | ❌ 显示错误提示信息 |
| **业务逻辑** | 每天只能签到一次，签到成功后获得金额奖励，连续签到天数递增 |

#### 1.3 管理订阅按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 管理 / 更多 |
| **位置** | 订阅信息卡片右上角 |
| **点击事件** | `router.push('/subscription')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到订阅管理页面 |

#### 1.4 显示/隐藏订阅URL按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 👁️ (眼睛图标) |
| **位置** | 订阅URL输入框旁边 |
| **点击事件** | `showSubUrls = !showSubUrls` |
| **调用API** | 无 |
| **功能** | 切换订阅URL的显示/隐藏状态 |
| **返回结果** | URL由加密掩码切换为明文或反之 |

#### 1.5 复制按钮 (订阅URL)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 📋 复制 |
| **位置** | 每个订阅URL行旁边 |
| **点击事件** | `copyText(url, label)` |
| **调用API** | 原生 API: `navigator.clipboard.writeText()` |
| **参数** | - `url` (string): 订阅地址<br>- `label` (string): 'Clash' 或 '通用' |
| **成功处理** | ✅ 复制到剪贴板<br>✅ 显示提示: "已复制Clash URL到剪贴板" |
| **失败处理** | ❌ 显示提示: "复制失败，请重试" |
| **业务场景** | 用户快速复制订阅链接用于配置客户端 |

#### 1.6 购买套餐按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 购买套餐 / 快速购买 |
| **位置** | 快捷操作网格 |
| **点击事件** | `router.push('/shop')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到套餐商店页面 |

#### 1.7 获取订阅按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 获取订阅 |
| **位置** | 快捷操作网格 |
| **点击事件** | `router.push('/subscription')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到订阅管理页面 |

#### 1.8 提交工单按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 提交工单 |
| **位置** | 快捷操作网格 |
| **点击事件** | `router.push('/tickets')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到工单列表页面 |

#### 1.9 邀请好友按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 邀请好友 |
| **位置** | 快捷操作网格 |
| **点击事件** | `router.push('/invite')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到邀请管理页面 |

#### 1.10 一键导入按钮 ⭐ (快速订阅)
| 属性 | 值 |
|------|-----|
| **按钮文本** | ☁️ (云下载图标) |
| **位置** | 快速订阅网格中每个App右下角 |
| **点击事件** | `oneClickImport(item.client)` |
| **调用API** | 无（Deep Link跳转） |
| **参数** | - `client` (string): 'clash'\|'stash'\|'surge'\|'loon'\|'quantumultx'\|'shadowrocket' |
| **URL Scheme** | `clash://import-profile/?url={encodedUrl}` 等 |
| **显示条件** | `item.importable === true` |
| **功能** | 打开对应代理客户端，并自动导入订阅链接 |
| **支持应用** | Clash、Stash、Surge、Loon、QuantumultX、Shadowrocket |
| **业务场景** | 用户一键快速切换不同代理客户端 |

#### 1.11 查看全部订单按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 查看全部 |
| **位置** | 最近订单卡片右上角 |
| **点击事件** | `router.push('/orders')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到我的订单页面 |

---

### 2️⃣ Orders (订单列表) `/orders` - 8个按钮

#### 2.1 购买套餐按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 购买套餐 |
| **位置** | 页面右上角 |
| **点击事件** | `router.push('/shop')` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到套餐商店 |

#### 2.2 状态筛选按钮（多个）
| 属性 | 值 |
|------|-----|
| **按钮文本** | 全部、待支付、已支付、已取消、已过期、已退款 |
| **位置** | 表格上方，水平排列 |
| **点击事件** | 点击后设置 `orderStatusFilter`，调用 `loadOrders()` |
| **调用API** | `GET /api/orders` |
| **参数** | `{ status: '', page: 1, page_size: 10 }` |
| **返回数据结构** | ```json<br>{ items: [{ id, order_no, package_name, amount, final_amount, status, payment_method_name, created_at, paid_at }], total: number }<br>``` |
| **成功处理** | ✅ 刷新表格数据<br>✅ 更新订单列表 |
| **失败处理** | ❌ 显示错误提示 |

#### 2.3 订单详情按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 详情 |
| **位置** | 每行操作列 |
| **点击事件** | 赋值 `detailOrder = order`，打开 `showDetailDrawer` |
| **调用API** | 无（本地展示） |
| **功能** | 打开抽屉显示订单完整信息 |
| **显示内容** | 订单号、用户、套餐、金额、状态、创建时间等 |

#### 2.4 继续支付按钮 (订单)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 继续支付 |
| **位置** | 每行操作列（仅待支付订单） |
| **点击事件** | `openOrderPay(order)` |
| **调用API** | 间接调用 `createPayment()` |
| **显示条件** | `status === 'pending'` |
| **功能** | 打开支付抽屉进行支付 |

#### 2.5 取消订单按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 取消 |
| **位置** | 每行操作列（仅待支付订单） |
| **点击事件** | `handleCancelOrder(order)` |
| **调用API** | `POST /api/orders/{orderNo}/cancel` |
| **参数** | 订单号 |
| **显示条件** | `status === 'pending'` |
| **确认信息** | 需二次确认，显示"确定要取消此订单吗？" |
| **成功处理** | ✅ 显示成功提示<br>✅ 刷新订单列表 |
| **失败处理** | ❌ 显示错误提示 |

#### 2.6 继续支付按钮 (充值)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 继续支付 |
| **位置** | 充值记录标签，每行操作列 |
| **点击事件** | `openRechargePay(record)` |
| **调用API** | 间接调用 `createRechargePayment()` |
| **显示条件** | `status === 'pending'` |
| **功能** | 打开充值支付抽屉 |

#### 2.7 取消充值按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 取消 |
| **位置** | 充值记录标签，每行操作列 |
| **点击事件** | `handleCancelRecharge(record)` |
| **调用API** | `POST /api/recharge/{id}/cancel` |
| **参数** | 充值记录ID |
| **显示条件** | `status === 'pending'` |
| **成功处理** | ✅ 显示成功提示<br>✅ 刷新充值记录列表 |

#### 2.8 去支付按钮 (订单详情抽屉)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 去支付 |
| **位置** | 订单详情抽屉底部 |
| **点击事件** | 关闭详情抽屉，打开支付抽屉 |
| **调用API** | 无（UI切换） |
| **显示条件** | `status === 'pending'` |

---

### 3️⃣ Shop (套餐商店) `/shop` - 6个按钮

#### 3.1 立即购买按钮 (套餐卡片)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 立即购买 |
| **位置** | 每个套餐卡片底部 |
| **点击事件** | `handleBuy(pkg)` |
| **调用API** | `POST /api/orders` |
| **参数** | `{ package_id: number, coupon_code?: string }` |
| **返回数据** | `{ id, order_no, amount, final_amount, discount_amount }` |
| **加载状态** | `ordering === true` |
| **成功处理** | ✅ 保存订单信息到 `orderInfo`<br>✅ 打开支付确认抽屉 |
| **失败处理** | ❌ 显示错误提示 |
| **特殊说明** | 推荐套餐（`is_featured=true`）时卡片有特殊徽章 |

#### 3.2 立即购买按钮 (自定义套餐)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 立即购买 |
| **位置** | 自定义套餐卡片底部 |
| **点击事件** | `handleCustomBuy()` |
| **调用API** | `POST /api/orders/custom` |
| **参数** | `{ devices: 1-20, months: number, coupon_code?: string }` |
| **返回数据** | 订单信息 |
| **加载状态** | `customOrdering === true` |
| **显示条件** | `customEnabled === true` |
| **成功处理** | ✅ 保存订单<br>✅ 打开支付确认抽屉 |

#### 3.3 验证优惠码按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 验证 |
| **位置** | 优惠码输入框右侧（支付确认抽屉） |
| **点击事件** | `handleVerifyCoupon()` |
| **调用API** | `POST /api/coupons/verify` |
| **参数** | `{ code: string, package_id: number }` |
| **返回数据** | `{ description, discount_rate, discount_amount }` |
| **禁用条件** | 优惠码为空或正在验证中 |
| **加载状态** | `verifying === true` |
| **成功处理** | ✅ 显示优惠信息<br>✅ 重新创建订单应用优惠<br>✅ 更新最终金额 |
| **失败处理** | ❌ 显示错误信息，清除优惠信息 |

#### 3.4 确认支付按钮 (支付抽屉) ⭐
| 属性 | 值 |
|------|-----|
| **按钮文本** | 确认支付 |
| **位置** | 支付确认抽屉底部右侧 |
| **点击事件** | `handlePay()` |
| **调用API** | **余额支付**: `POST /api/orders/{orderNo}/pay`<br>**第三方**: `POST /api/payment` |
| **参数 (余额支付)** | `{ payment_method: 'balance' }` |
| **参数 (第三方)** | ```json<br>{ order_id, payment_method_id, is_mobile, use_balance?, balance_amount? }<br>``` |
| **返回数据 (余额)** | 支付成功/失败标志 |
| **返回数据 (第三方)** | ```json<br>{ order_no, pay_type, pay_url, crypto_info? }<br>pay_type: 'alipay'\|'wxpay'\|'stripe'\|'crypto'<br>crypto_info: { network, currency, amount_usdt, wallet_address }<br>``` |
| **禁用条件** | 订单不完整或未选择支付方式 |
| **加载状态** | `paying === true` |
| **支付方式** | 余额、支付宝、微信、Stripe、加密货币 |
| **成功处理分支** | <br>**Crypto**: 打开二维码抽屉，启动轮询 (3秒/次，最多20次)<br>**QR/Mobile**: 打开扫码支付抽屉，启动轮询<br>**余额**: 关闭抽屉，跳转支付成功页 |

#### 3.5 取消支付按钮 (扫码支付抽屉)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 取消支付 |
| **位置** | 扫码支付抽屉底部 |
| **点击事件** | `showQrModal = false`（触发 `@after-leave` 的轮询停止） |
| **调用API** | 无 |
| **功能** | 关闭抽屉，停止轮询支付状态 |

#### 3.6 确认按钮 (加密货币支付)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 我已转账 |
| **位置** | 加密货币支付抽屉底部右侧 |
| **点击事件** | `handleCryptoTransferred()` |
| **调用API** | 无 |
| **功能** | 继续轮询支付状态，直到检测到转账完成 |

---

### 4️⃣ Subscription (订阅管理) `/subscription` - 12个按钮

#### 4.1 升级套餐按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 升级套餐 |
| **位置** | 英雄卡片右上角 |
| **点击事件** | `showUpgradeModal = true` |
| **调用API** | 无（打开抽屉） |
| **功能** | 打开升级套餐抽屉 |
| **显示条件** | 用户已有订阅 |

#### 4.2 发送邮箱按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 发送邮箱 |
| **位置** | 订阅地址卡片右上角 |
| **点击事件** | `handleSendEmail()` |
| **调用API** | `POST /api/subscriptions/send-email` |
| **参数** | 无 |
| **加载状态** | `sendingEmail === true` |
| **成功处理** | ✅ 显示成功提示 |
| **失败处理** | ❌ 显示错误提示 |
| **功能** | 将当前订阅信息发送到邮箱 |

#### 4.3 重置订阅按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 重置 |
| **位置** | 订阅地址卡片右上角 |
| **点击事件** | `showResetModal = true`（弹出确认） |
| **调用API** | `POST /api/subscriptions/reset` |
| **参数** | 无 |
| **确认信息** | "重置后原订阅地址将失效，所有设备需要重新配置。确定要继续吗？" |
| **成功处理** | ✅ 显示成功提示<br>✅ 重新加载订阅数据<br>✅ 更新订阅URL |

#### 4.4 显示/隐藏订阅按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 显示/隐藏 |
| **位置** | 每个订阅URL行 |
| **点击事件** | `showSubUrls = !showSubUrls` |
| **调用API** | 无 |
| **功能** | 切换订阅URL的显示状态 |

#### 4.5 复制订阅按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 📋 复制 |
| **位置** | 每个订阅URL行 |
| **点击事件** | `copyToClipboard(url, label)` |
| **调用API** | `navigator.clipboard.writeText()` |
| **参数** | - `url`: 订阅地址<br>- `label`: 标签名称 |
| **成功处理** | ✅ 复制到剪贴板，显示提示 |

#### 4.6 二维码按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 📱 二维码 |
| **位置** | 每个订阅URL行 |
| **点击事件** | `showQrCode(url, title)` |
| **调用API** | 本地: `QRCode.toCanvas()` |
| **参数** | - `url`: 订阅地址<br>- `title`: 二维码标题 |
| **功能** | 打开二维码抽屉，显示生成的二维码图 |

#### 4.7 计算金额按钮 (升级抽屉) ⭐
| 属性 | 值 |
|------|-----|
| **按钮文本** | 计算金额 |
| **位置** | 升级套餐抽屉底部 |
| **点击事件** | `handleCalcUpgrade()` |
| **调用API** | `POST /api/orders/upgrade/calc` |
| **参数** | `{ add_devices: number≥1, extend_months?: number≥0 }` |
| **返回数据** | ```json<br>{ price_per_device_year, current_device_limit, remaining_days, add_devices, extend_months, fee_extend, fee_new_devices, total }<br>``` |
| **加载状态** | `upgradeCalcLoading === true` |
| **成功处理** | ✅ 显示详细费用明细<br>✅ 更新 `upgradeResult` |
| **功能** | 根据增加设备数和续期月数计算升级费用 |

#### 4.8 去支付按钮 (升级抽屉) ⭐
| 属性 | 值 |
|------|-----|
| **按钮文本** | 去支付 |
| **位置** | 升级套餐抽屉底部 |
| **点击事件** | `handleOpenUpgradePay()` |
| **调用API** | `POST /api/orders/upgrade` |
| **参数** | `{ add_devices, extend_months, coupon_code? }` |
| **返回数据** | 升级订单信息 |
| **加载状态** | `upgradeSubmitting === true` |
| **禁用条件** | `upgradeResult === null` 或 `upgradeResult.total <= 0` |
| **成功处理** | ✅ 打开升级支付抽屉<br>✅ 关闭升级套餐抽屉 |

#### 4.9 确认支付按钮 (升级支付抽屉)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 确认支付 |
| **位置** | 升级支付抽屉底部 |
| **点击事件** | `handleUpgradePay()` |
| **调用API** | `POST /api/orders/{orderNo}/pay` 或 `POST /api/payment` |
| **参数** | 同普通订单支付 |
| **加载状态** | `paying === true` |
| **成功处理** | <br>**第三方**: 打开支付抽屉，启动轮询<br>**余额**: 显示升级成功抽屉 |

#### 4.10 取消支付按钮 (升级支付抽屉)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 取消 |
| **位置** | 升级支付抽屉底部左侧 |
| **点击事件** | `showUpgradePayModal = false` |
| **调用API** | 无 |
| **功能** | 关闭抽屉 |

#### 4.11 转换剩余天数按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 转换剩余天数为余额 |
| **位置** | 英雄卡片底部 |
| **点击事件** | `showConvertModal = true`（弹出确认） |
| **调用API** | `POST /api/subscriptions/convert-to-balance` |
| **参数** | 无 |
| **禁用条件** | `!canConvert` (剩余天数不足) |
| **确认信息** | "将剩余 X 天转换为余额，转换后订阅将立即失效。确定要继续吗？" |
| **成功处理** | ✅ 显示成功提示<br>✅ 关闭抽屉<br>✅ 重新加载数据 |

#### 4.12 快速导入按钮 (格式选择)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 复制 / 导入 |
| **位置** | 每个格式卡片上 |
| **点击事件** | **复制**: `copyToClipboard(url, fmt.name)`<br>**导入**: `importFormat(fmt.client)` |
| **调用API** | **复制**: 无<br>**导入**: Deep Link跳转 |
| **参数** | 格式对象，包含 `url` 和 `client` |
| **功能** | 复制对应格式的订阅URL或打开客户端导入 |
| **显示条件** | `fmt.url` 存在且 `fmt.importable === true` |

---

### 5️⃣ Tickets (工单系统) `/tickets` - 3个按钮

#### 5.1 新建工单按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 新建工单 |
| **位置** | 页面右上角 |
| **点击事件** | `showCreateModal = true` |
| **调用API** | 无（打开弹窗） |
| **功能** | 打开新建工单对话框 |

#### 5.2 查看详情按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 查看详情 |
| **位置** | 工单列表表格，每行操作列 |
| **点击事件** | `router.push('/tickets/' + ticket.id)` |
| **调用API** | 无（页面跳转） |
| **功能** | 跳转到工单详情页 |

#### 5.3 提交工单按钮 (新建抽屉)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 提交工单 |
| **位置** | 新建工单抽屉底部 |
| **点击事件** | `handleCreate()` |
| **调用API** | `POST /api/tickets` |
| **参数** | ```json<br>{ title: string, type: string, content: string }<br>title: 1-100字符<br>type: 'technical'\|'billing'\|'account'\|'other'<br>content: 1-2000字符<br>``` |
| **返回数据** | 工单创建成功响应 |
| **加载状态** | `submitting === true` |
| **验证规则** | title、type、content 都必填 |
| **成功处理** | ✅ 显示成功提示<br>✅ 关闭抽屉<br>✅ 清空表单<br>✅ 刷新工单列表 |
| **失败处理** | ❌ 显示错误提示 |

---

### 6️⃣ Settings (用户设置) `/settings` - 6个按钮

#### 6.1 保存个人资料按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 保存 |
| **位置** | 个人资料标签，表单底部 |
| **点击事件** | `saveProfile()` |
| **调用API** | `POST /api/users/profile` |
| **参数** | `{ username, nickname, theme, language, timezone }` |
| **加载状态** | `savingProfile === true` |
| **成功处理** | ✅ 显示成功提示<br>✅ 更新应用主题<br>✅ 重新加载用户信息 |
| **失败处理** | ❌ 显示错误提示 |

#### 6.2 修改密码按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 修改密码 |
| **位置** | 修改密码标签，表单底部 |
| **点击事件** | `savePw()` |
| **调用API** | `POST /api/users/change-password` |
| **参数** | `{ old_password, new_password, confirm_password }` |
| **验证规则** | - old_password: 必填<br>- new_password: 必填，≥6字符<br>- confirm_password: 必填，需与new_password相同 |
| **加载状态** | `savingPw === true` |
| **成功处理** | ✅ 显示成功提示<br>✅ 清空表单 |
| **失败处理** | ❌ 显示错误提示（如旧密码错误、新密码太弱等） |

#### 6.3 通知设置开关 (6种)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 邮件通知总开关 / 订单相关通知 / 订阅到期提醒 / 订阅变更通知 / 异常登录提醒 / 推送通知 |
| **位置** | 通知设置标签，各行 |
| **点击事件** | `@update:value="saveNotif"` (自动保存) |
| **调用API** | `POST /api/users/notification-settings` |
| **参数** | ```json<br>{ email_notifications, notify_order, notify_expiry, notify_subscription, abnormal_login_alert_enabled, push_notifications }<br>``` |
| **依赖关系** | `notify_order`, `notify_expiry`, `notify_subscription` 需要 `email_notifications === true` 才能启用 |
| **成功处理** | ✅ 显示成功提示 |
| **失败处理** | ❌ 显示错误提示 |

#### 6.4 隐私设置开关 (2种)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 数据共享 / 使用分析 |
| **位置** | 隐私设置标签，各行 |
| **点击事件** | `@update:value="savePrivacy"` (自动保存) |
| **调用API** | `POST /api/users/privacy-settings` |
| **参数** | `{ data_sharing, analytics }` |
| **成功处理** | ✅ 显示成功提示 |

#### 6.5 解绑Telegram按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 解绑Telegram |
| **位置** | Telegram绑定卡片 |
| **点击事件** | `handleUnbindTelegram()` |
| **调用API** | `POST /api/users/telegram/unbind` |
| **参数** | 无 |
| **加载状态** | `unbindingTelegram === true` |
| **显示条件** | 仅在已绑定状态显示 |
| **成功处理** | ✅ 显示成功提示<br>✅ 重新加载用户信息<br>✅ 隐藏解绑按钮 |

---

## 管理端页面按钮

### 7️⃣ Admin Dashboard `/admin` - 8个按钮

#### 7.1 刷新数据按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 刷新数据 |
| **位置** | 欢迎部分右上角 |
| **点击事件** | `loadDashboard()` |
| **调用API** | `GET /api/admin/dashboard` |
| **返回数据** | ```json<br>{ total_users, active_subscriptions, today_revenue, month_revenue, pending_orders, pending_tickets, new_users_today, recent_users, recent_orders, revenue_trend }<br>``` |
| **成功处理** | ✅ 更新各项统计数据和图表 |

#### 7.2-7.8 快捷链接按钮 (7个)
| 按钮 | 位置 | 功能 |
|-----|------|------|
| 待支付订单 | 待办任务卡片 | 跳转 `/admin/orders?status=pending` |
| 待处理工单 | 待办任务卡片 | 跳转 `/admin/tickets` |
| 异常用户提醒 | 待办任务卡片 | 跳转 `/admin/abnormal-users` |
| 用户项（可点击） | 新注册用户卡片 | 跳转到订阅管理页，搜索该用户 |
| 订单项（可点击） | 最近订单卡片 | 跳转到订单管理页，搜索该订单 |
| 查看订阅管理 | 新注册用户卡片底部 | 跳转 `/admin/subscriptions` |
| 查看全部订单 | 最近订单卡片底部 | 跳转 `/admin/orders` |

---

### 8️⃣ Admin Users (用户管理) `/admin/users` - 13个按钮

#### 8.1 新增用户按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 新增用户 |
| **位置** | 页面右上角 |
| **点击事件** | `openCreateModal()` |
| **调用API** | 无（打开弹窗） |
| **功能** | 打开创建用户对话框 |

#### 8.2 刷新按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 刷新 |
| **位置** | 页面右上角 |
| **点击事件** | `fetchUsers()` |
| **调用API** | `GET /api/admin/users` |
| **参数** | `{ page, page_size, search?, status? }` |

#### 8.3-8.6 批量操作按钮 (4个)
| 按钮 | API | 功能 |
|-----|-----|------|
| 批量启用 | `POST /api/admin/users/batch` | 启用选中用户 |
| 批量禁用 | `POST /api/admin/users/batch` | 禁用选中用户 |
| 批量删除 | `POST /api/admin/users/batch` | 删除选中用户（需确认） |
| 设置等级 | 打开弹窗 | 为用户设置等级 |

**显示条件**: 仅当选中行时显示  
**禁用条件**: 未选中行  
**位置**: 表格上方  
**参数**: `{ user_ids: Array, action: string }`

#### 8.7 搜索按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 搜索 / Enter回车 |
| **位置** | 页面顶部搜索栏 |
| **点击事件** | `handleSearch()` 或 @keydown.enter |
| **调用API** | `GET /api/admin/users` |
| **参数** | `{ search_query, status_filter }` |
| **搜索字段** | 邮箱、用户名、订阅地址 |

#### 8.8 编辑用户按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 保存 |
| **位置** | 编辑用户抽屉底部 |
| **点击事件** | `handleSaveUser()` |
| **调用API** | `POST /api/admin/users/{id}` 或 `POST /api/admin/users` |
| **参数** | `{ username, email, password?, balance, is_admin, is_active, expire_time?, device_limit, notes }` |
| **加载状态** | `saving === true` |
| **特新说明** | 新建时 password 必填，编辑时密码可选 |

#### 8.9 查看详情按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 详情 |
| **位置** | 表格每行操作列 |
| **点击事件** | `handleViewDetail(row)` |
| **调用API** | 无（打开详情抽屉） |
| **功能** | 打开用户详情抽屉 |

#### 8.10 编辑按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 编辑 |
| **位置** | 表格每行操作列 |
| **点击事件** | `handleEdit(row)` |
| **调用API** | 无（打开编辑弹窗） |
| **功能** | 打开编辑用户弹窗 |

#### 8.11 启用/禁用按钮 (行操作)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 启用 / 禁用 |
| **位置** | 表格每行操作列 |
| **点击事件** | `handleToggleActive(row)` |
| **调用API** | `POST /api/admin/users/{id}/toggle-active` |
| **功能** | 切换用户启用/禁用状态 |
| **按钮文案** | 根据当前状态动态显示 |

#### 8.12 重置密码按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 重置密码 |
| **位置** | 表格每行操作列（更多操作下拉菜单） |
| **点击事件** | `handleAction('resetPwd', row)` |
| **调用API** | `POST /api/admin/users/{id}/reset-password` |
| **参数** | `{ password: string }` |
| **特殊说明** | 需二次确认，弹出密码输入框 |
| **成功处理** | ✅ 显示成功提示，新密码已设置 |

#### 8.13 删除用户按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 删除 |
| **位置** | 表格每行操作列（更多操作下拉菜单） |
| **点击事件** | `handleAction('delete', row)` |
| **调用API** | `DELETE /api/admin/users/{id}` |
| **特殊说明** | 需二次确认弹框，显示"确定要删除此用户及其关联数据吗？" |
| **成功处理** | ✅ 显示成功提示<br>✅ 从表格移除 |

---

### 9️⃣ Admin Packages (套餐管理) `/admin/packages` - 6个按钮

#### 9.1 新建套餐按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 新建套餐 |
| **位置** | 页面右上角 |
| **点击事件** | `handleCreate()` |
| **调用API** | 无（打开编辑弹窗） |

#### 9.2-9.3 批量操作按钮 (2个)
| 按钮 | 功能 |
|-----|------|
| 批量启用 | 启用选中套餐 |
| 批量禁用 | 禁用选中套餐 |

**API**: `POST /api/admin/packages/batch`  
**参数**: `{ ids, action }`  
**显示条件**: 仅选中时显示

#### 9.4 编辑按钮 (表格)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 编辑 |
| **位置** | 表格每行操作列 |
| **点击事件** | `handleEdit(pkg)` |
| **功能** | 打开编辑套餐表单 |

#### 9.5 删除按钮 (表格)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 删除 |
| **位置** | 表格每行操作列 |
| **点击事件** | `handleDelete(pkg)` |
| **API** | `DELETE /api/admin/packages/{id}` |
| **特殊说明** | 需二次确认 |

#### 9.6 保存按钮 (编辑表单)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 确认保存 |
| **位置** | 编辑抽屉底部 |
| **点击事件** | `handleSavePackage()` |
| **API** | `POST /api/admin/packages` 或 `POST /api/admin/packages/{id}` |
| **参数** | `{ name, description, price, duration_days, device_limit, features, is_active, is_featured, sort_order }` |
| **加载状态** | `saving === true` |
| **验证** | name、price、duration_days、device_limit 必填 |

---

### 🔟 Admin Orders (订单管理) `/admin/orders` - 5个按钮

#### 10.1 刷新订单按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 刷新 |
| **位置** | 页面右上角 |
| **API** | `GET /api/admin/orders` |
| **参数** | `{ page, page_size, search?, status? }` |

#### 10.2-10.3 批量操作按钮 (2个)
| 按钮 | 功能 |
|-----|------|
| 批量取消 | 取消选中待支付订单 |
| 批量退款 | 退款选中订单（需确认） |

#### 10.4 查看详情按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 详情 |
| **位置** | 表格每行或可点击行 |
| **功能** | 打开订单详情抽屉 |

#### 10.5 订单操作按钮 (详情抽屉)
| 按钮 | API | 功能 |
|-----|-----|------|
| 退款 | `POST /api/admin/orders/{id}/refund` | 处理退款 |
| 取消 | `POST /api/admin/orders/{id}/cancel` | 取消订单 |
| 删除 | `DELETE /api/admin/orders/{id}` | 删除订单 |
| 完成 | `POST /api/admin/orders/{id}/complete` | 标记完成 |

---

### 1️⃣1️⃣ Admin Subscriptions (订阅管理) `/admin/subscriptions` - 16个按钮

#### 11.1-11.4 批量操作按钮 (4个)
| 按钮 | 功能 |
|-----|------|
| 批量启用 | 启用选中订阅 |
| 批量禁用 | 禁用选中订阅 |
| 批量发送 | 发送订阅邮件给选中用户 |
| 批量删除 | 删除选中用户账户（需确认） |

**显示条件**: 仅选中时显示  
**位置**: 表格上方

#### 11.5 快速延期按钮 (Mobile卡片)
| 按钮 | 模式 | 天数 |
|-----|------|------|
| +1月 | Mobile | 30天 |
| +3月 | Mobile | 90天 |
| +半年 | Mobile | 180天 |
| +1年 | Mobile | 365天 |
| +2年 | Mobile | 730天 |

**API**: `POST /api/admin/subscriptions/{id}/extend`  
**参数**: `{ days }`  
**显示条件**: 仅Desktop卡片  
**位置**: 订阅卡片中

#### 11.6 设置到期时间按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 日期选择器 |
| **位置** | 订阅卡片日期区域 |
| **API** | `POST /api/admin/subscriptions/{id}/set-expire` |
| **参数** | `{ expire_time: timestamp }` |
| **功能** | 手动设置订阅过期时间 |

#### 11.7 快速增加设备按钮
| 按钮 | 增加设备 |
|-----|---------|
| +2 | 2个 |
| +5 | 5个 |
| +10 | 10个 |
| +20 | 20个 |
| +30 | 30个 |

**API**: `POST /api/admin/subscriptions/{id}/update-device-limit`  
**参数**: `{ device_limit: number }`  
**位置**: 设备限制区域

#### 11.8 清理设备按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 清理 |
| **位置** | 设备限制区域 |
| **API** | `POST /api/admin/subscriptions/{id}/clear-devices` |
| **特殊说明** | 需二次确认 |
| **功能** | 清除该订阅下所有已连接设备 |

#### 11.9 查看详情按钮 (Mobile卡片)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 详情 |
| **位置** | 订阅卡片操作网格 |
| **功能** | 打开详情抽屉，显示完整订阅信息 |

#### 11.10 后台登录按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 后台 |
| **位置** | 订阅卡片操作网格 |
| **API** | `POST /api/admin/users/{id}/login-as` |
| **功能** | 作为该用户登录到前台 |
| **用途** | 管理员代理查看用户的前台界面 |

#### 11.11 重置订阅按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 重置 |
| **位置** | 订阅卡片操作网格 |
| **API** | `POST /api/admin/subscriptions/{id}/reset` |
| **特殊说明** | 需二次确认 |
| **功能** | 重置订阅，生成新的订阅URL |

#### 11.12 发送邮件按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 发邮件 |
| **位置** | 订阅卡片操作网格 |
| **API** | `POST /api/admin/users/{id}/send-subscription-email` |
| **功能** | 向用户发送订阅信息邮件 |

#### 11.13 启用/禁用按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 禁用 / 启用 |
| **位置** | 订阅卡片操作网格 |
| **API** | `POST /api/admin/subscriptions/{id}/toggle-active` |
| **按钮文案** | 根据当前状态动态显示 |
| **特殊说明** | 当前管理员用户的订阅不能禁用 |

#### 11.14 复制URL按钮 (2种)
| 按钮 | 复制内容 |
|-----|---------|
| 通用 | 通用格式订阅URL |
| Clash | Clash专用订阅URL |

**功能**: 复制到剪贴板  
**位置**: 订阅卡片操作网格

#### 11.15 二维码按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 二维码 |
| **位置** | 订阅卡片操作网格 |
| **功能** | 生成并显示订阅URL的二维码 |

#### 11.16 删除用户按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 删除 |
| **位置** | 订阅卡片操作网格 |
| **API** | `DELETE /api/admin/users/{id}` |
| **特殊说明** | 需二次确认，删除整个用户账户及其所有数据 |
| **功能** | 永久删除用户 |

---

### 1️⃣2️⃣ Admin Coupons (优惠券管理) `/admin/coupons` - 8个按钮

#### 12.1 创建优惠券按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 创建优惠券 |
| **位置** | 页面右上角（Desktop）或工具栏（Mobile） |
| **点击事件** | `handleAdd()` |
| **功能** | 打开优惠券创建表单 |

#### 12.2-12.4 批量操作按钮 (3个)
| 按钮 | 功能 |
|-----|------|
| 批量启用 | 启用选中优惠券 |
| 批量禁用 | 禁用选中优惠券 |
| 批量删除 | 删除选中优惠券（需确认） |

**显示条件**: 仅选中时显示  
**位置**: 表格上方

#### 12.5 编辑按钮 (Mobile卡片)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 编辑 |
| **位置** | Mobile卡片操作栏 |
| **功能** | 打开编辑表单 |

#### 12.6 删除按钮 (Mobile卡片)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 删除 |
| **位置** | Mobile卡片操作栏 |
| **API** | `DELETE /api/admin/coupons/{id}` |
| **特殊说明** | 需二次确认 |

#### 12.7 复制代码按钮
| 属性 | 值 |
|------|-----|
| **按钮文本** | 复制代码 |
| **位置** | Mobile卡片操作栏 |
| **功能** | 复制优惠券代码到剪贴板 |

#### 12.8 提交按钮 (编辑表单)
| 属性 | 值 |
|------|-----|
| **按钮文本** | 提交 |
| **位置** | 编辑抽屉底部 |
| **API** | `POST /api/admin/coupons` 或 `POST /api/admin/coupons/{id}` |
| **参数** | `{ code, name, description, type, discount_value, min_amount, max_discount, valid_from, valid_until, total_quantity }` |
| **加载状态** | `submitting === true` |
| **优惠类型** | 'discount'(百分比) \| 'fixed'(固定金额) \| 'days'(免费天数) |

---

## 通用模式

### 轮询模式 ⏳
```
触发条件: 第三方支付/Crypto支付
轮询间隔: 3000ms (3秒)
最大轮询次数: 20次 (共60秒)
监控API:
- GET /api/orders/{id}/status
- GET /api/recharge/{id}/status
```

### 支付流程 💳
```
1. 创建订单
   ↓
2. 显示支付确认对话框
   ↓
3. 选择支付方式（余额/第三方/Crypto）
   ↓
4. 发起支付
   ├─ 余额: 直接扣款，跳转成功页
   ├─ 第三方: 显示QR码，启动轮询
   └─ Crypto: 显示钱包地址，启动轮询
   ↓
5. 轮询支付状态（3秒轮询一次）
   ↓
6. 检测到支付完成
   ↓
7. 跳转支付成功页面
   ↓
8. 更新用户余额和订阅状态
```

### 表单验证模式 ✅
```
使用: Naive UI n-form 组件
验证时机: 提交前调用 form.validate()
验证失败: 阻止提交，显示错误提示
验证通过: 发送API请求
```

### 加载状态管理 ⚙️
```
按钮UI: :loading="isLoading"
禁用条件: :disabled="isLoading || isFormInvalid"
状态变化:
  点击 → loading=true
  API返回 → loading=false
  异常 → loading=false + 错误提示
```

---

## API映射表

| API端点 | 对应按钮 | 参数示例 | 返回数据 |
|--------|---------|--------|--------|
| `POST /api/user/checkin` | 签到 | 无 | `{amount, consecutive_days}` |
| `POST /api/orders` | 立即购买（套餐） | `{package_id⚠}` | `{id, order_no, final_amount}` |
| `POST /api/orders/custom` | 立即购买（自定义） | `{devices⚠, months⚠}` | 同上 |
| `POST /api/orders/{no}/pay` | 确认支付（余额） | `{payment_method}` | 支付结果 |
| `POST /api/payment` | 确认支付（第三方） | `{order_id⚠, payment_method_id⚠}` | `{pay_url, pay_type}` |
| `POST /api/subscriptions/reset` | 重置订阅 | 无 | 成功/失败 |
| `POST /api/subscriptions/convert-to-balance` | 转换为余额 | 无 | 成功/失败 |
| `POST /api/subscriptions/send-email` | 发送邮箱 | 无 | 成功/失败 |
| `POST /api/orders/upgrade/calc` | 计算金额 | `{add_devices⚠, extend_months?}` | `{total, fee_extend}` |
| `POST /api/orders/upgrade` | 去支付（升级） | 同计算金额 | 升级订单信息 |
| `POST /api/tickets` | 提交工单 | `{title⚠, type⚠, content⚠}` | 工单ID |
| `POST /api/users/profile` | 保存个人资料 | 表单对象 | 成功/失败 |
| `POST /api/users/change-password` | 修改密码 | `{old_password⚠, new_password⚠}` | 成功/失败 |
| `POST /api/admin/users/batch` | 批量用户操作 | `{user_ids⚠, action⚠}` | 操作结果 |
| `POST /api/admin/subscriptions/{id}/extend` | 快速延期 | `{days⚠}` | 成功/失败 |
| `POST /api/admin/coupons/verify` | 验证优惠码 | `{code⚠, package_id⚠}` | `{discount_rate, amount}` |

⚠ 表示必填参数

---

**文档完成：包含 102 个按钮的详细功能说明、API映射、参数、返回值、禁用条件、加载状态、成功/失败处理等全面信息。**
