# 前端优化指南 - Modal 改 Drawer

## 优化目标

1. 统一所有列表风格（管理员端和用户端）
2. 优化手机端显示
3. 将所有弹窗（Modal）改为抽屉（Drawer）形式

## 已完成的工作

### 1. 创建统一组件

- ✅ `CommonDrawer.vue` - 统一的抽屉组件
- ✅ `UnifiedTable.vue` - 统一的表格组件（自动适配桌面/移动端）
- ✅ `UnifiedCardList.vue` - 统一的卡片列表组件（移动端）
- ✅ `unified.css` - 统一的样式文件

### 2. 组件特性

#### CommonDrawer 特性
- 响应式宽度（移动端全屏，桌面端固定宽度）
- 支持自定义头部、内容、底部
- 支持确认/取消按钮
- 支持加载状态

#### UnifiedTable 特性
- 自动检测设备类型
- 桌面端显示表格
- 移动端显示卡片列表
- 统一的分页组件
- 支持自定义移动端字段和操作

#### UnifiedCardList 特性
- 统一的卡片样式
- 支持自定义头部、内容、操作
- 支持空状态和加载状态
- 响应式设计

## 使用示例

### 1. 将 Modal 改为 Drawer

**修改前（Modal）：**
```vue
<n-modal
  v-model:show="showDetailModal"
  preset="card"
  title="订单详情"
  style="width: 560px; max-width: 92vw;"
>
  <n-descriptions :column="1" bordered>
    <!-- 内容 -->
  </n-descriptions>
</n-modal>
```

**修改后（Drawer）：**
```vue
<common-drawer
  v-model:show="showDetailDrawer"
  title="订单详情"
  :width="560"
>
  <n-descriptions :column="1" bordered>
    <!-- 内容 -->
  </n-descriptions>
</common-drawer>
```

### 2. 使用 UnifiedTable 组件

```vue
<template>
  <unified-table
    :columns="columns"
    :data="data"
    :loading="loading"
    :pagination="pagination"
    :mobile-fields="mobileFields"
    :mobile-actions="mobileActions"
    @update:page="handlePageChange"
  />
</template>

<script setup lang="ts">
import UnifiedTable from '@/components/UnifiedTable.vue'

const columns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '名称', key: 'name' },
  // ...
]

const mobileFields = [
  { key: 'id', label: 'ID' },
  { key: 'name', label: '名称' },
  // ...
]

const mobileActions = [
  {
    key: 'edit',
    label: '编辑',
    type: 'primary',
    onClick: (item) => handleEdit(item)
  },
  // ...
]
</script>
```

### 3. 使用 UnifiedCardList 组件

```vue
<template>
  <unified-card-list
    :data="data"
    :fields="fields"
    :actions="actions"
    :loading="loading"
  >
    <template #header="{ item }">
      <span class="card-title">订单 #{{ item.order_no }}</span>
      <n-tag :type="getStatusType(item.status)">
        {{ getStatusText(item.status) }}
      </n-tag>
    </template>
  </unified-card-list>
</template>

<script setup lang="ts">
import UnifiedCardList from '@/components/UnifiedCardList.vue'

const fields = [
  { key: 'order_no', label: '订单号' },
  { key: 'amount', label: '金额', format: (v) => `¥${v}` },
  // ...
]

const actions = [
  {
    key: 'detail',
    label: '详情',
    type: 'info',
    onClick: (item) => showDetail(item)
  },
  // ...
]
</script>
```

## 需要修改的页面清单

### 用户端页面（优先级高）

1. ✅ `/order/Index.vue` - 订单列表（3个Modal → Drawer）
2. ✅ `/order/Shop.vue` - 购买套餐（2个Modal → Drawer）
3. `/subscription/Index.vue` - 订阅管理
4. `/device/Index.vue` - 设备管理
5. `/settings/Index.vue` - 设置页面
6. `/dashboard/Index.vue` - 仪表盘
7. `/mystery-box/Index.vue` - 盲盒
8. `/ticket/Index.vue` - 工单

### 管理端页面（优先级中）

1. `/admin/users/Index.vue` - 用户管理（多个Modal）
2. `/admin/orders/Index.vue` - 订单管理
3. `/admin/packages/Index.vue` - 套餐管理
4. `/admin/nodes/Index.vue` - 节点管理
5. `/admin/subscriptions/Index.vue` - 订阅管理
6. `/admin/coupons/Index.vue` - 优惠券管理
7. `/admin/tickets/Index.vue` - 工单管理
8. `/admin/redeem/Index.vue` - 卡密管理
9. `/admin/mystery-box/Index.vue` - 盲盒管理
10. `/admin/announcements/Index.vue` - 公告管理
11. `/admin/email-queue/Index.vue` - 邮件队列
12. `/admin/settings/Index.vue` - 系统设置

## 修改步骤

### 步骤 1：引入组件

```vue
<script setup lang="ts">
import CommonDrawer from '@/components/CommonDrawer.vue'
import UnifiedTable from '@/components/UnifiedTable.vue'
import UnifiedCardList from '@/components/UnifiedCardList.vue'
</script>
```

### 步骤 2：替换 Modal 为 Drawer

1. 将 `n-modal` 改为 `common-drawer`
2. 将 `v-model:show` 保持不变
3. 移除 `preset="card"` 和 `style` 属性
4. 添加 `:width` 属性（可选）
5. 如果有 footer，使用 `show-footer` 和 `@confirm`/`@cancel` 事件

### 步骤 3：统一列表样式

1. 桌面端使用 `n-data-table`
2. 移动端使用 `unified-card-list` 或自定义卡片
3. 使用统一的 CSS 类名（`.mobile-card-list`, `.mobile-card`, `.card-row` 等）

### 步骤 4：测试

1. 测试桌面端显示
2. 测试移动端显示
3. 测试 Drawer 打开/关闭
4. 测试分页功能

## 样式规范

### 卡片结构

```vue
<div class="mobile-card">
  <div class="card-header">
    <span class="card-title">标题</span>
    <n-tag>标签</n-tag>
  </div>
  <div class="card-body">
    <div class="card-row">
      <span class="card-label">标签:</span>
      <span class="card-value">值</span>
    </div>
  </div>
  <div class="card-actions">
    <n-button size="small">操作</n-button>
  </div>
</div>
```

### 统计卡片

```vue
<n-card class="stat-card stat-card-blue">
  <div class="stat-content">
    <div class="stat-icon">
      <n-icon :size="28" :component="Icon" />
    </div>
    <div class="stat-info">
      <div class="stat-label">标签</div>
      <div class="stat-value">123</div>
    </div>
  </div>
</n-card>
```

## 注意事项

1. **响应式设计**：所有组件都应该在移动端和桌面端都能正常工作
2. **性能优化**：大列表使用虚拟滚动或分页
3. **用户体验**：Drawer 从右侧滑出，移动端全屏显示
4. **一致性**：所有页面使用相同的组件和样式
5. **可访问性**：确保键盘导航和屏幕阅读器支持

## 优化效果

### 修改前
- ❌ Modal 在移动端体验不佳
- ❌ 列表样式不统一
- ❌ 代码重复度高

### 修改后
- ✅ Drawer 在移动端全屏显示，体验更好
- ✅ 所有列表使用统一样式
- ✅ 代码复用率高，易于维护
- ✅ 响应式设计，自动适配设备

## 下一步

1. 逐个页面进行优化
2. 测试所有功能
3. 收集用户反馈
4. 持续改进

---

**创建时间**: 2026-03-02
**文档版本**: 1.0
