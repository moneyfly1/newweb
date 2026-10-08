# 手机端「像 App」改造契约（所有页面统一遵守）

> 这份文档是**唯一**的改造标准。改页面前先读它，改完用文末的体检脚本自检。
> 目标：手机端不再是「桌面页面缩一缩」，而是接近原生 App 的列表/卡片/操作栏体验。

## 0. 铁律

1. **不要改公共文件**（`src/styles/mobile-app-ui.css`、`src/styles/mobile-cards.css`、
   `src/composables/useBatchSelection.ts`、`src/components/BatchSelectBar.vue`、
   `scripts/mobile-audit.mjs`）。这些由基础设施负责人统一维护。
   页面里需要新样式就写在页面自己的 `<style scoped>` 里。
2. **不要每页各写一套全选/批量逻辑**。必须用公共组件（见第 2 节）。
3. 改完必须跑体检脚本，把「你负责的页面」跑到没有 `small-tap-target` /
   `element-overflow-x` / `doc-overflow-x` / `missing-select-all` / `input-focus-outline` /
   `desktop-table-on-mobile` 问题。`tiny-text` 也要清（正文不小于 12px）。

## 1. 手机端布局基线（已由全局 CSS 提供，页面别再自己加边距）

| 事项 | 基线 |
| --- | --- |
| 页面左右留白 | 全局 10px，**页面根容器不要再写左右 padding/margin**（原来三层叠加成 36px，手机上是白边浪费） |
| 卡片 | 16px 圆角、轻阴影、纯色底（不要 nth-child 渐变、不要重阴影） |
| 卡片内边距 | 10px 12px（全局已设），页面不要覆盖 |
| 触控目标 | 按钮 ≥ 40px 高；图标按钮 ≥ 40×40；复选框/开关/关闭按钮已由全局兜底 |
| 字体 | 正文 ≥ 13px，说明文字 ≥ 12px；标题 15–17px |
| 列表 | 优先用 `.mobile-card-list` + `.mobile-card`（已重写为 App 风格）；也可用 `.app-list` + `.app-list-item`（自带按压反馈、右侧箭头 `.app-list-item__chevron`） |
| 工具栏 | 手机端筛选/搜索用 `.app-sticky-toolbar` 吸顶。**注意**：`position:sticky` 只能在直接父元素的盒子里移动，所以这个元素必须是「卡片内容区 / 页面根容器」的**直接子元素**；不要包在高度≈自身的短 wrapper（空 div、短 n-space）里，否则没有滚动行程、等于没吸顶（全局样式已对「短空 div 包裹」的情况做 `display:contents` 兜底，但别依赖它） |
| 底部 | 内容区底部已由布局预留 tabbar + 批量栏空间，不要自己再加 `padding-bottom` |

**判定「够不够宽」**：`手机视口宽 - 内容实际宽度` 应 ≤ 约 24px（含卡片自身内边距）。
体检脚本的 `element-overflow-x` 与截图对比可验证；把页面根容器上的 `padding: 16px/24px`、
`max-width` + `margin: 0 auto` 去掉。

## 2. 全选 / 批量选择（公共组件，必用）

```vue
<script setup lang="ts">
import { useBatchSelection } from '@/composables/useBatchSelection'
import BatchSelectBar from '@/components/BatchSelectBar.vue'

const tableData = ref<any[]>([])
// 唯一的选择状态：桌面表格与手机卡片共用
const selection = useBatchSelection(() => tableData.value)
</script>

<template>
  <!-- 列表上方（手机端会自动变成固定在底部的操作栏） -->
  <BatchSelectBar
    :total="selection.total.value"
    :selected-count="selection.count.value"
    :all-selected="selection.allSelected.value"
    :indeterminate="selection.indeterminate.value"
    label="条订单"
    @toggle-all="selection.toggleAll"
    @clear="selection.clear"
  >
    <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">
      批量删除
    </n-button>
  </BatchSelectBar>

  <!-- 桌面：naive 表格仍需数组，用 computed 桥接，保证两边状态一致 -->
  <n-data-table v-model:checked-row-keys="checkedRowKeys" ... />
</template>
```

```ts
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
})
```

手机端卡片列表（**关键：手机端必须也能多选**）：

```vue
<div class="mobile-card-list">
  <div
    v-for="row in tableData"
    :key="row.id"
    class="mobile-card is-selectable"
    :class="{ 'is-selected': selection.isSelected(row) }"
    @click="selection.toggle(row)"
  >
    <div class="card-check" @click.stop>
      <n-checkbox :checked="selection.isSelected(row)" @update:checked="() => selection.toggle(row)" />
    </div>
    <div class="card-header"> ... </div>
    <div class="card-body"> ... </div>
    <!-- 卡片里的按钮必须 @click.stop，否则点按钮会连带选中 -->
    <div class="card-actions" @click.stop> ... </div>
  </div>
</div>
```

要点：
- 批量操作读 `selection.selectedRows.value`（对象数组）或 `selection.selectedKeys.value`，**不要**再读 `checkedRowKeys`。
- 操作完成后调用 `selection.clear()`。

### 2.1 硬性要求：**每个列表页手机端都必须有「全选」**（用户明确要求）

不允许出现「有列表、有行内操作，但没有全选」的页面。批量动作按下面优先级选：

1. **有后端批量接口** → 直接用（如 `batchUserAction` / `batchOrderAction` / `batchNodeAction`、
   `/nodes/batch-test`、`/admin/custom-nodes/batch-*`）。
2. **没有批量接口，但有行内单条操作** → 用 `Promise.allSettled` 逐条调单条接口，
   操作前弹确认框写明「将影响 N 项」，结束后汇总「成功 X / 失败 Y」。
   （这是本项目既有做法：invites / redeem / mystery-box 的批量删除就是这么实现的。）
3. **只读列表（没有任何行内操作）** → 提供「复制所选」作为批量动作
   （用 `@/utils/clipboard` 的 `copyToClipboard`，把所选行按一行一条拼成文本）。
   不允许空有勾选框却没有动作。

禁止：只有全选、没有任何动作；或者按钮点了不生效（假按钮）。

## 3. 页面级常见问题与做法

1. **桌面表格硬塞手机**：手机端改为 `.mobile-card-list` 卡片（`appStore.isMobile` 判断，
   现有页面大多已经有这个分支，只是缺选择与样式统一）。
2. **横向溢出**：按钮/工具行 `flex-wrap: wrap`；宽按钮 `max-width: 100%`；
   表格容器 `overflow-x: auto`。
3. **字号过小**：把 11px 提到 12px、12px 提到 13px（页面 scoped 样式里改）。
4. **筛选区太长**：手机端收成「搜索框 + 筛选按钮（打开抽屉/下拉）」，
   或放进 `.app-sticky-toolbar`。
5. **弹窗/抽屉**：手机端用整屏宽度（全局已把 n-modal/n-dialog 限制到 95vw），
   底部操作类抽屉用 `placement="bottom"`。

## 4. 自检（必须做）

```bash
cd frontend
# 只体检你负责的页面
node scripts/mobile-audit.mjs /admin/users /admin/orders
```

- 登录态缓存在 `/tmp/mobile-audit-state.json`（脚本自动创建），登录接口有 10 次/分钟限流，
  连续跑不会重复登录。
- 详细样本在 `/tmp/mobile-audit/report.json`，含「哪个元素、宽多少、在哪个页面」。
- 目标：负责的页面输出全是 `✓`。

## 5. 报告格式

改完回报：
1. 改了哪些文件、每个页面做了什么（一句话/页）
2. 体检脚本输出（你负责的页面的那一行）
3. 没能解决的问题 + 原因（例如后端缺批量接口）
