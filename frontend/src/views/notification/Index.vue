<template>
  <div class="notification-page" @touchstart.passive="pullTouchStart" @touchmove.passive="pullTouchMove" @touchend.passive="pullTouchEnd">
    <!-- 下拉刷新指示器 -->
    <transition name="fade">
      <div v-if="pullDistance > 0 || pullRefreshing" class="pull-indicator" :style="{ transform: `translate(-50%, ${Math.min(pullDistance, 70) - 40}px)` }">
        <n-spin v-if="pullRefreshing" size="small" />
        <span v-else>{{ pullDistance >= 55 ? '释放刷新' : '下拉刷新' }}</span>
      </div>
    </transition>

    <div class="notif-header">
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
      <div v-else class="notif-list">
        <div
          v-for="n in notifications"
          :key="n.id"
          class="notif-item"
          :class="{ unread: !n.is_read }"
          @click="handleClick(n)"
        >
          <div class="notif-dot" v-if="!n.is_read" />
          <div class="notif-main">
            <div class="notif-title">{{ n.title }}</div>
            <div class="notif-content">{{ n.content }}</div>
            <div class="notif-time">{{ formatTime(n.created_at) }}</div>
          </div>
          <n-button text size="tiny" type="error" @click.stop="handleDelete(n.id)">删除</n-button>
        </div>
      </div>

      <!-- 分页 -->
      <n-pagination
        v-if="pagination.itemCount > pagination.pageSize"
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :item-count="pagination.itemCount"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        style="margin-top: 16px; justify-content: center"
        @update:page="loadData"
        @update:page-size="handlePageSize"
      />
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { listNotifications, getUnreadCount, markNotificationRead, markAllRead, deleteNotification } from '@/api/common'
import { useTable } from '@/composables/useTable'
import { usePullRefresh } from '@/composables/usePullRefresh'

const message = useMessage()
const filter = ref('all')
const unreadCount = ref(0)

const { loading, tableData: notifications, pagination, loadData, reload } = useTable(listNotifications, {
  getParams: () => ({ is_read: filter.value === 'unread' ? 'false' : undefined }),
})
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

const formatTime = (t: string) => {
  if (!t) return ''
  const d = new Date(t)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60 * 1000) return '刚刚'
  if (diff < 60 * 60 * 1000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 24 * 60 * 60 * 1000) return `${Math.floor(diff / 3600000)} 小时前`
  if (diff < 7 * 24 * 60 * 60 * 1000) return `${Math.floor(diff / 86400000)} 天前`
  return d.toLocaleDateString('zh-CN')
}

onMounted(() => {
  loadData()
  fetchUnread()
})
</script>

<style scoped>
.notification-page { padding: 12px; position: relative; }

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

.notif-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.notif-tabs { display: flex; gap: 8px; }
.notif-tab {
  padding: 6px 16px;
  border-radius: 999px;
  font-size: 13px;
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
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  margin-left: 4px;
  border-radius: 999px;
  background: var(--danger-color, #dc2626);
  color: #fff;
  font-size: 10px;
}

.notif-list { display: flex; flex-direction: column; gap: 10px; }
.notif-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  border-radius: 14px;
  background: var(--bg-color, #fff);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  position: relative;
  transition: transform 0.12s ease;
}

.notif-item.unread { background: var(--primary-color-soft, rgba(79, 70, 229, 0.04)); }
.notif-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--danger-color, #dc2626);
  flex-shrink: 0;
  margin-top: 6px;
}
.notif-main { flex: 1; min-width: 0; }
.notif-title { font-size: 14px; font-weight: 600; color: var(--text-color); margin-bottom: 4px; }
.notif-content {
  font-size: 13px;
  color: var(--text-color-secondary, #666);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.notif-time { font-size: 11px; color: var(--text-color-secondary, #999); margin-top: 6px; }
</style>
