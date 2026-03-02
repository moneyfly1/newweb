# Modal 改 Drawer 完整示例

## 套餐管理页面修改示例

### 修改步骤

#### 1. 引入 CommonDrawer 组件

在 `<script setup>` 中添加：
```typescript
import CommonDrawer from '@/components/CommonDrawer.vue'
```

#### 2. 修改变量名

```typescript
// 修改前
const showEditModal = ref(false)

// 修改后
const showEditDrawer = ref(false)
```

#### 3. 替换 Modal 为 Drawer

```vue
<!-- 修改前 -->
<n-modal
  v-model:show="showEditModal"
  preset="dialog"
  :title="isCreating ? '新建套餐' : '编辑套餐'"
  :positive-text="'保存'"
  :negative-text="'取消'"
  :style="appStore.isMobile ? 'width: 95%; max-width: 600px' : 'width: 600px'"
  @positive-click="handleSavePackage"
>
  <n-form>...</n-form>
</n-modal>

<!-- 修改后 -->
<common-drawer
  v-model:show="showEditDrawer"
  :title="isCreating ? '新建套餐' : '编辑套餐'"
  :width="600"
  show-footer
  :loading="saving"
  @confirm="handleSavePackage"
  @cancel="showEditDrawer = false"
>
  <n-form>...</n-form>
</common-drawer>
```

#### 4. 更新所有引用

使用查找替换：
- `showEditModal` → `showEditDrawer`

---

## 批量修改脚本

由于需要修改多个页面，建议使用以下方法：

### 方法 1：手动修改（推荐）
按照上述步骤逐个页面修改，确保准确性。

### 方法 2：使用 sed 批量替换（谨慎使用）

```bash
# 备份文件
cp file.vue file.vue.bak

# 替换变量名
sed -i '' 's/showEditModal/showEditDrawer/g' file.vue
sed -i '' 's/showAddModal/showAddDrawer/g' file.vue
sed -i '' 's/showDetailModal/showDetailDrawer/g' file.vue
sed -i '' 's/showImportModal/showImportDrawer/g' file.vue

# 注意：Modal 标签的替换需要手动处理，因为涉及属性变化
```

---

## 通用替换模式

### 添加/编辑表单类 Modal

**特征：**
- 有表单输入
- 有保存/取消按钮
- 需要加载状态

**替换为：**
```vue
<common-drawer
  v-model:show="showDrawer"
  :title="title"
  :width="600"
  show-footer
  :loading="saving"
  @confirm="handleSave"
  @cancel="showDrawer = false"
>
  <n-form>...</n-form>
</common-drawer>
```

### 详情查看类 Modal

**特征：**
- 只读内容
- 只有关闭按钮

**替换为：**
```vue
<common-drawer
  v-model:show="showDetailDrawer"
  title="详情"
  :width="700"
  show-footer
  :show-confirm="false"
  cancel-text="关闭"
  @cancel="showDetailDrawer = false"
>
  <n-descriptions>...</n-descriptions>
</common-drawer>
```

### 确认对话框类 Modal

**特征：**
- 简单确认
- 危险操作

**保持使用 useDialog：**
```typescript
// 这类简单确认对话框保持使用 useDialog
dialog.warning({
  title: '确认删除',
  content: '确定要删除吗？',
  positiveText: '确定',
  negativeText: '取消',
  onPositiveClick: () => handleDelete()
})
```

---

## 需要注意的点

### 1. 宽度设置
- 小表单：400-500px
- 中等表单：600px
- 大表单/详情：700-800px
- 移动端：自动全屏（CommonDrawer 已处理）

### 2. 加载状态
确保添加 `saving` 或 `loading` 状态：
```typescript
const saving = ref(false)

const handleSave = async () => {
  saving.value = true
  try {
    await saveData()
    showDrawer.value = false
  } finally {
    saving.value = false
  }
}
```

### 3. 表单验证
Drawer 中的表单验证与 Modal 相同：
```typescript
const formRef = ref()

const handleSave = async () => {
  await formRef.value?.validate()
  // 保存逻辑
}
```

---

## 测试清单

修改完成后，测试以下内容：

- [ ] Drawer 能正常打开/关闭
- [ ] 表单数据正确显示
- [ ] 保存功能正常
- [ ] 取消功能正常
- [ ] 移动端全屏显示
- [ ] 桌面端宽度合适
- [ ] 加载状态正确显示
- [ ] 表单验证正常工作

---

**创建时间**: 2026-03-02
