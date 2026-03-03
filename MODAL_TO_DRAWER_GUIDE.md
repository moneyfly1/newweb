# 用户端 Modal 改为 Drawer 修改指南

## 已完成
- ✅ Shop.vue - 套餐购买页面
- ✅ Order/Index.vue - 订单列表（已使用 Drawer）

## 待修改页面

### 1. Subscription/Index.vue - 订阅管理
需要修改的 Modal:
- QR Code Modal (二维码)
- Reset Modal (重置订阅)
- Convert Modal (转换余额)
- Upgrade Pay Modal (升级支付)
- Pay QR Modal (支付二维码)
- Crypto Modal (加密货币)
- Upgrade Modal (升级订阅)

### 2. Recharge/Index.vue - 充值页面
### 3. Invite/Index.vue - 邀请码
### 4. Ticket/Index.vue - 工单
### 5. Device/Index.vue - 设备管理

## 修改模式

### Modal 转 Drawer 的标准模式

**原 Modal:**
```vue
<n-modal
  v-model:show="showModal"
  preset="card"
  title="标题"
  style="width: 500px;"
>
  内容
  <template #footer>
    <n-space justify="end">
      <n-button @click="showModal = false">取消</n-button>
      <n-button type="primary" @click="handleConfirm">确定</n-button>
    </n-space>
  </template>
</n-modal>
```

**改为 Drawer:**
```vue
<common-drawer
  v-model:show="showModal"
  title="标题"
  :width="500"
  show-footer
  @confirm="handleConfirm"
  @cancel="showModal = false"
>
  内容
</common-drawer>
```

### Dialog 类型 Modal 转 Drawer

**原 Dialog Modal:**
```vue
<n-modal
  v-model:show="showModal"
  preset="dialog"
  title="确认"
  content="确定要执行此操作吗？"
  positive-text="确定"
  negative-text="取消"
  @positive-click="handleConfirm"
/>
```

**改为 Drawer:**
```vue
<common-drawer
  v-model:show="showModal"
  title="确认"
  :width="400"
  show-footer
  @confirm="handleConfirm"
  @cancel="showModal = false"
>
  <p>确定要执行此操作吗？</p>
</common-drawer>
```

## 注意事项

1. **导入组件**: 需要在 script 中导入 `CommonDrawer`
2. **宽度设置**: Drawer 使用 `:width="500"` 而不是 `style="width: 500px;"`
3. **Footer 处理**: 使用 `show-footer` 和 `@confirm/@cancel` 事件
4. **Loading 状态**: 使用 `:loading="loading"` 传递加载状态
5. **特殊按钮**: 使用 `:show-confirm="false"` 隐藏确认按钮，`cancel-text` 自定义取消按钮文本

## 测试清单

每个页面修改后需要测试：
- [ ] 抽屉能正常打开/关闭
- [ ] 表单提交功能正常
- [ ] 取消按钮功能正常
- [ ] 移动端显示正常
- [ ] 没有控制台错误
