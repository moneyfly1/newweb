# 批量操作功能待添加页面

## 需要添加的页面（按优先级）

### 高优先级
1. ✅ invites (邀请码) - 进行中
2. ❌ redeem (卡密)
3. ❌ levels (等级)
4. ❌ tickets (工单)

### 中优先级
5. ❌ email-queue (邮件队列)
6. ❌ mystery-box (盲盒)
7. ❌ payment-gateways (支付网关)
8. ❌ abnormal-users (异常用户)

### 低优先级（统计/配置页面）
- logs (日志) - 只读，不需要批量操作
- stats (统计) - 只读，不需要批量操作
- settings (设置) - 配置页面，不需要批量操作
- config-update (配置更新) - 配置页面，不需要批量操作

## 添加步骤
1. 添加 checkedRowKeys 状态
2. 在列定义中添加 { type: 'selection' }
3. 添加批量操作按钮区域
4. 为数据表添加选择功能
5. 实现批量操作函数
