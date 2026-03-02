# 管理端 Modal 改 Drawer 任务清单

## 任务概述
将所有管理端的编辑、添加、修改功能从 Modal 改为 Drawer。

---

## 📋 需要修改的页面

### 1. 节点管理 - `/admin/nodes/Index.vue`
**Modal 类型：**
- 添加节点
- 编辑节点
- 批量导入节点
- 节点详情

### 2. 专线节点 - `/admin/custom-nodes/Index.vue`
**Modal 类型：**
- 添加专线节点
- 编辑专线节点
- 批量导入
- 分配用户

### 3. 用户管理 - `/admin/users/Index.vue`
**Modal 类型：**
- 添加用户
- 编辑用户
- 重置密码
- 用户详情（登录历史、余额日志等）
- CSV 导入

### 4. 套餐管理 - `/admin/packages/Index.vue`
**Modal 类型：**
- 添加套餐
- 编辑套餐

### 5. 优惠券管理 - `/admin/coupons/Index.vue`
**Modal 类型：**
- 添加优惠券
- 编辑优惠券

### 6. 订阅管理 - `/admin/subscriptions/Index.vue`
**Modal 类型：**
- 重置订阅
- 延期订阅
- 修改设备限制

### 7. 订单管理 - `/admin/orders/Index.vue`
**Modal 类型：**
- 订单详情
- 退款确认

### 8. 工单管理 - `/admin/tickets/Index.vue`
**Modal 类型：**
- 工单详情
- 回复工单

### 9. 卡密管理 - `/admin/redeem/Index.vue`
**Modal 类型：**
- 批量生成卡密

### 10. 公告管理 - `/admin/announcements/Index.vue`
**Modal 类型：**
- 添加公告
- 编辑公告

### 11. 用户等级 - `/admin/levels/Index.vue`
**Modal 类型：**
- 添加等级
- 编辑等级

### 12. 盲盒管理 - `/admin/mystery-box/Index.vue`
**Modal 类型：**
- 添加奖池
- 编辑奖池
- 添加奖品
- 编辑奖品

---

## 🔧 修改步骤

### 步骤 1：准备工作
1. 确保 CommonDrawer 组件已创建 ✅
2. 确保统一样式已引入 ✅

### 步骤 2：逐个页面修改

#### 通用修改模式

**1. 引入组件**
```vue
<script setup>
import CommonDrawer from '@/components/CommonDrawer.vue'
</script>
```

**2. 替换 Modal 为 Drawer**
```vue
<!-- 修改前 -->
<n-modal
  v-model:show="showEditModal"
  preset="card"
  title="编辑"
  style="width: 600px"
>
  <n-form>...</n-form>
  <template #footer>
    <n-space justify="end">
      <n-button @click="showEditModal = false">取消</n-button>
      <n-button type="primary" @click="handleSave">保存</n-button>
    </n-space>
  </template>
</n-modal>

<!-- 修改后 -->
<common-drawer
  v-model:show="showEditDrawer"
  title="编辑"
  :width="600"
  show-footer
  :loading="saving"
  @confirm="handleSave"
  @cancel="showEditDrawer = false"
>
  <n-form>...</n-form>
</common-drawer>
```

**3. 更新变量名**
```typescript
// 修改前
const showEditModal = ref(false)

// 修改后
const showEditDrawer = ref(false)
```

**4. 更新所有引用**
```typescript
// 修改前
showEditModal.value = true

// 修改后
showEditDrawer.value = true
```

---

## 📝 详细修改计划

### 优先级 P0 - 高频使用页面

#### 1. 节点管理 (nodes/Index.vue)
**预计 Modal 数量：** 4个
**修改内容：**
- showAddModal → showAddDrawer
- showEditModal → showEditDrawer
- showImportModal → showImportDrawer
- showDetailModal → showDetailDrawer

#### 2. 用户管理 (users/Index.vue)
**预计 Modal 数量：** 5个
**修改内容：**
- showCreateModal → showCreateDrawer
- showEditModal → showEditDrawer
- showResetPwdModal → showResetPwdDrawer
- showDetailModal → showDetailDrawer
- showImportModal → showImportDrawer

#### 3. 套餐管理 (packages/Index.vue)
**预计 Modal 数量：** 2个
**修改内容：**
- showAddModal → showAddDrawer
- showEditModal → showEditDrawer

### 优先级 P1 - 中频使用页面

#### 4. 优惠券管理 (coupons/Index.vue)
**预计 Modal 数量：** 2个

#### 5. 专线节点 (custom-nodes/Index.vue)
**预计 Modal 数量：** 4个

#### 6. 订阅管理 (subscriptions/Index.vue)
**预计 Modal 数量：** 3个

### 优先级 P2 - 低频使用页面

#### 7-12. 其他页面
- 订单管理
- 工单管理
- 卡密管理
- 公告管理
- 用户等级
- 盲盒管理

---

## 🎯 预期效果

### 修改前
- ❌ Modal 在移动端显示不佳
- ❌ 表单内容多时需要滚动
- ❌ 用户体验不一致

### 修改后
- ✅ Drawer 从右侧滑出，移动端全屏
- ✅ 表单内容自然展示
- ✅ 所有页面体验一致
- ✅ 更符合现代 UI 设计规范

---

## 📊 工作量估算

| 页面 | Modal 数量 | 预计时间 | 优先级 |
|------|-----------|---------|--------|
| 节点管理 | 4 | 30分钟 | P0 |
| 用户管理 | 5 | 40分钟 | P0 |
| 套餐管理 | 2 | 15分钟 | P0 |
| 优惠券管理 | 2 | 15分钟 | P1 |
| 专线节点 | 4 | 30分钟 | P1 |
| 订阅管理 | 3 | 20分钟 | P1 |
| 其他6个页面 | ~15 | 90分钟 | P2 |
| **总计** | **~35** | **~4小时** | - |

---

## 🚀 开始执行

现在开始修改高优先级页面...

---

**创建时间**: 2026-03-02
**任务状态**: 🔄 进行中
