<template>
  <div class="notification-page" @touchstart.passive="pullTouchStart" @touchmove.passive="pullTouchMove" @touchend.passive="pullTouchEnd">
    <!-- 下拉刷新指示器 -->
    <transition name="fade">
      <div v-if="pullDistance > 0 || pullRefreshing" class="pull-indicator" :style="{ transform: `translate(-50%, ${Math.min(pullDistance, 70) - 40}px)` }">
        <n-spin v-if="pullRefreshing" size="small" />
        <span v-else>{{ pullDistance >= 55 ? '释放刷新' : '下拉刷新' }}</span>
      </div>
    </transition>

    <!-- 筛选条：手机端由 .app-sticky-toolbar 吸顶（该类只在 ≤767px 生效，桌面无副作用） -->
    <div class="notif-header app-sticky-toolbar">
      <div class="notif-tabs">
        <div class="notif-tab" :class="{ active: filter === 'all' }" @click="switchFilter('all')">全部</div>
        <div class="notif-tab" :class="{ active: filter === 'unread' }" @click="switchFilter('unread')">
          未读<template v-if="unreadCount > 0"><span class="unread-badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span></template>
        </div>
      </div>
      <n-button v-if="unreadCount > 0" text size="small" type="primary" @click="handleMarkAllRead">全部已读</n-button>
    </div>

    <n-spin :show="loading">
      <div v-if="notifications.length === 0" class="mobile-empty">暂无通知</div>
      <template v-else>
        <!-- 全选 / 批量操作：公共组件（桌面在列表上方，手机端自动变成固定在底部的操作栏） -->
        <BatchSelectBar
          :total="selection.total.value"
          :selected-count="selection.count.value"
          :all-selected="selection.allSelected.value"
          :indeterminate="selection.indeterminate.value"
          label="条通知"
          @toggle-all="selection.toggleAll"
          @clear="selection.clear"
        >
          <n-button size="small" :disabled="!unreadSelectedCount" @click="handleBatchMarkRead">
            批量已读
          </n-button>
          <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">
            批量删除
          </n-button>
        </BatchSelectBar>

        <div class="mobile-card-list">
          <div
            v-for="n in notifications"
            :key="n.id"
            class="mobile-card is-selectable"
            :class="{ 'is-unread': !n.is_read, 'is-selected': selection.isSelected(n) }"
            @click="selection.toggle(n)"
          >
            <div class="card-check" @click.stop>
              <n-checkbox :checked="selection.isSelected(n)" @update:checked="() => selection.toggle(n)" />
            </div>
            <div class="card-header">
              <span class="card-title">{{ n.title }}</span>
              <n-tag v-if="!n.is_read" size="small" type="error" :bordered="false">未读</n-tag>
            </div>
            <p class="notif-content">{{ n.content }}</p>
            <div class="notif-foot" @click.stop>
              <span class="notif-time">{{ formatRelativeTime(n.created_at, '') }}</span>
              <div class="notif-foot__actions">
                <n-button v-if="!n.is_read" size="tiny" quaternary type="primary" @click="handleClick(n)">标记已读</n-button>
                <n-button size="tiny" quaternary type="error" @click="handleDelete(n.id)">删除</n-button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- 分页 -->
      <n-pagination
        v-if="pagination.itemCount > pagination.pageSize"
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :item-count="pagination.itemCount"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        style="margin-top: 12px; justify-content: center"
        @update:page="loadData"
        @update:page-size="handlePageSize"
      />
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { listNotifications, getUnreadCount, markNotificationRead, markAllRead, deleteNotification } from '@/api/common'
import { useTable } from '@/composables/useTable'
import { usePullRefresh } from '@/composables/usePullRefresh'
import { useBatchSelection } from '@/composables/useBatchSelection'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { formatRelativeTime } from '@/utils/format'

const message = useMessage()
const dialog = useDialog()
const filter = ref('all')
const unreadCount = ref(0)

const { loading, tableData: notifications, pagination, loadData, reload } = useTable(listNotifications, {
  getParams: () => ({ is_read: filter.value === 'unread' ? 'false' : undefined }),
})

// 全选 / 多选：唯一的选择状态（公共组件 useBatchSelection + BatchSelectBar）
const selection = useBatchSelection<any>(() => notifications.value)
// 已选里还能标记已读的数量（都是已读时按钮置灰，避免点了没反应）
const unreadSelectedCount = computed(() => selection.selectedRows.value.filter(n => !n.is_read).length)
const { distance: pullDistance, refreshing: pullRefreshing, onTouchStart: pullTouchStart, onTouchMove: pullTouchMove, onTouchEnd: pullTouchEnd } =
  usePullRefresh(async () => { await loadData(); await fetchUnread() })

const fetchUnread = async () => {
  try { const res: any = await getUnreadCount(); unreadCount.value = res.data?.unread_count || 0 } catch {}
}

function switchFilter(f: string) {
  filter.value = f
  reload()
}

function handlePageSize(size: number) {
  pagination.pageSize = size
  pagination.page = 1
  loadData()
}

const handleClick = async (n: any) => {
  if (!n.is_read) {
    n.is_read = true
    unreadCount.value = Math.max(0, unreadCount.value - 1)
    try { await markNotificationRead(n.id) } catch { /* 忽略 */ }
  }
}

const handleMarkAllRead = async () => {
  try {
    await markAllRead()
    notifications.value.forEach(n => { n.is_read = true })
    unreadCount.value = 0
    message.success('已全部标为已读')
  } catch (e: any) {
    message.error(e.message || '操作失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteNotification(id)
    notifications.value = notifications.value.filter(n => n.id !== id)
    message.success('已删除')
    await fetchUnread()
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

// ===== 批量操作 =====
// 后端没有批量接口：按契约 §2.1 第 2 条逐条调单条接口，
// 用 Promise.allSettled 保证一条失败不影响其余，完成后汇总「成功 X / 失败 Y」。
const handleBatchMarkRead = () => {
  const rows = selection.selectedRows.value.filter(n => !n.is_read)
  if (!rows.length) {
    message.warning('所选通知都已是已读')
    return
  }
  dialog.warning({
    title: '批量标记已读',
    content: `将影响 ${rows.length} 条未读通知。`,
    positiveText: '标记已读',
    negativeText: '取消',
    onPositiveClick: async () => {
      const results = await Promise.allSettled(rows.map(n => markNotificationRead(n.id)))
      const okIds = rows.filter((_, i) => results[i].status === 'fulfilled').map(n => n.id)
      const fail = results.length - okIds.length
      if (okIds.length > 0) {
        notifications.value.forEach(n => { if (okIds.includes(n.id)) n.is_read = true })
        message.success(`批量标记完成：成功 ${okIds.length} 条`)
      }
      if (fail > 0) message.error(`批量标记部分失败：失败 ${fail} 条`)
      selection.clear()
      await fetchUnread()
    },
  })
}

const handleBatchDelete = () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  dialog.warning({
    title: '批量删除通知',
    content: `将影响 ${rows.length} 条通知，删除后不可恢复。`,
    positiveText: '确定删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      const results = await Promise.allSettled(rows.map(n => deleteNotification(n.id)))
      const okIds = rows.filter((_, i) => results[i].status === 'fulfilled').map(n => n.id)
      const fail = results.length - okIds.length
      if (okIds.length > 0) {
        notifications.value = notifications.value.filter(n => !okIds.includes(n.id))
        message.success(`批量删除完成：成功 ${okIds.length} 条`)
      }
      if (fail > 0) message.error(`批量删除部分失败：失败 ${fail} 条`)
      selection.clear()
      await fetchUnread()
    },
  })
}

onMounted(() => {
  loadData()
  fetchUnread()
})
</script>

<style scoped>
/* 手机端横向留白由全局统一（10px）：页面根容器不再叠加左右 padding */
.notification-page { padding: 8px 0 12px; position: relative; }

/* 下拉刷新指示器 */
.pull-indicator {
  position: fixed;
  top: 0;
  left: 50%;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 96px;
  height: 34px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--bg-color, #fff);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.12);
  font-size: 12px;
  color: var(--text-color-secondary, #666);
  transition: transform 0.15s ease;
}
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 筛选条：桌面是普通一行，手机端被全局 .app-sticky-toolbar 变成吸顶工具栏 */
.notif-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.notif-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
.notif-tab {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
  padding: 6px 16px;
  border-radius: 999px;
  font-size: 14px;
  color: var(--text-color-secondary, #666);
  background: var(--primary-color-soft, rgba(79, 70, 229, 0.06));
  cursor: pointer;
  position: relative;
}
.notif-tab.active {
  color: #fff;
  background: var(--primary-color, #4f46e5);
  font-weight: 600;
}
.unread-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  margin-left: 4px;
  border-radius: 999px;
  background: var(--danger-color, #dc2626);
  color: #fff;
  font-size: 12px;
  line-height: 1;
}

/* 通知内容：卡片外观走全局 .mobile-card（16px 圆角/轻阴影/纯色底），
   这里只补页面自己的文本排版（用页面专属类名，避免覆盖全局卡片样式） */
.notif-content {
  margin: 0;
  padding: 0 12px;
  font-size: 13px;
  color: var(--text-color-secondary, #666);
  line-height: 1.6;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.notif-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 12px 8px;
  margin-top: 6px;
}
.notif-foot__actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
.notif-time { font-size: 12px; color: var(--text-color-secondary, #999); }
.notif-header.app-sticky-toolbar { padding: 8px 12px; }

@media (max-width: 767px) {
  /* 卡片自身 padding 为 0（全局列表行自带内边距），只给标题行补内边距避免标题贴边 */
  .notification-page :deep(.mobile-card .card-header) { padding: 12px 12px 0; margin-bottom: 8px; }
  /* 可选中卡片：左侧给复选框让出 44px（页面卡片被全局改成 padding:0 !important，
     这里必须同权重压过，否则复选框会压在通知标题上） */
  .notification-page :deep(.mobile-card.is-selectable) { padding-left: 44px !important; }
  /* 吸顶工具栏的默认负边距是给「卡片内边距」留位的，这里列表已贴齐，改成 0 防溢出 */
  .notif-header.app-sticky-toolbar { margin: 0 0 10px; padding: 8px 12px; }
  /* 分页按钮默认 28×28，手指点不准：手机端撑到 40px（父级已 flex-wrap，不会横向撑破） */
  .notification-page :deep(.n-pagination .n-pagination-item) { min-width: 40px; height: 40px; }
  /* 未读通知用主色描边 + 淡底，比「一个红点」更容易扫到 */
  .notification-page :deep(.mobile-card.is-unread) {
    border-color: color-mix(in srgb, var(--primary-color, #4f46e5) 45%, transparent) !important;
    background: color-mix(in srgb, var(--primary-color-soft, rgba(79, 70, 229, 0.08)) 70%, var(--bg-color, #fff)) !important;
  }
}
</style>
