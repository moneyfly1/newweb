<template>
  <div class="config-update-container">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">节点自动更新</h2>
        <p class="page-subtitle">配置节点爬虫与采集任务，支持定时采集、规则过滤及实时日志监控</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-button type="primary" :loading="starting" @click="handleStart">
            <template #icon><n-icon><PlayOutline /></n-icon></template>
            立即执行采集
          </n-button>
          <n-button type="warning" secondary :disabled="!status.running" @click="handleStop">
            <template #icon><n-icon><StopOutline /></n-icon></template>
            停止
          </n-button>
          <n-button @click="fetchStatus" secondary>
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新状态
          </n-button>
        </n-space>
      </div>
    </div>

    <n-card title="运行状态" :bordered="false" class="admin-main-card">
      <n-space>
        <n-tag :type="status.running ? 'success' : 'default'">
          {{ status.running ? '运行中' : '已停止' }}
        </n-tag>
        <n-tag :type="status.scheduled ? 'info' : 'default'">
          {{ status.scheduled ? '定时任务已启用' : '定时任务未启用' }}
        </n-tag>
      </n-space>
    </n-card>

    <n-card title="更新配置" style="margin-top: 16px">
      <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '120'">
        <n-form-item label="订阅URL列表">
          <div style="width: 100%">
            <div ref="urlListRef" class="url-list">
              <div v-for="(url, index) in config.urls" :key="index" class="url-item">
                <div class="drag-handle">
                  <n-icon size="20"><ReorderThreeOutline /></n-icon>
                </div>
                <div class="url-index">{{ index + 1 }}</div>
                <template v-if="url === '__MANUAL_NODES__'">
                  <div class="url-input manual-placeholder">
                    <n-tag type="warning" size="small" round>手动节点</n-tag>
                    <n-text depth="3" style="margin-left: 8px; font-size: 12px">拖动此条目可调整手动节点在订阅中的显示位置</n-text>
                  </div>
                </template>
                <template v-else>
                  <n-input
                    v-model:value="config.urls[index]"
                    placeholder="请输入订阅URL"
                    class="url-input"
                  />
                </template>
                <n-button
                  v-if="url !== '__MANUAL_NODES__'"
                  text
                  type="error"
                  @click="config.urls.splice(index, 1)"
                  class="delete-btn"
                >
                  <template #icon>
                    <n-icon size="18">
                      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
                        <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M368 368L144 144M368 144L144 368"/>
                      </svg>
                    </n-icon>
                  </template>
                </n-button>
                <div v-else style="width: 18px"></div>
              </div>
            </div>
            <n-space style="margin-top: 8px">
              <n-button dashed @click="config.urls.push('')" style="flex: 1">+ 添加订阅URL</n-button>
              <n-button dashed type="warning" @click="addManualPlaceholder" :disabled="config.urls.includes('__MANUAL_NODES__')">+ 插入手动节点位置</n-button>
            </n-space>
          </div>
        </n-form-item>
        <n-form-item label="关键词过滤">
          <div style="width: 100%">
            <n-dynamic-input v-model:value="config.keywords" placeholder="输入关键词（匹配节点名称）" />
            <n-text depth="3" style="font-size: 12px; margin-top: 4px; display: block">
              留空表示不过滤，导入所有节点。填写关键词后，名称中包含关键词的节点将被排除。支持地区缩写如 hk、us、jp 等。
            </n-text>
          </div>
        </n-form-item>
        <n-form-item label="定时更新">
          <n-switch v-model:value="config.enabled" />
          <n-text depth="3" style="margin-left: 12px; font-size: 12px">
            启用后将按设定间隔自动更新节点
          </n-text>
        </n-form-item>
        <n-form-item label="更新间隔（分钟）">
          <n-input-number v-model:value="config.interval" :min="1" :max="1440" :style="{ width: appStore.isMobile ? '100%' : '200px' }" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end">
          <n-button type="primary" :loading="saving" @click="handleSaveConfig">保存配置</n-button>
        </div>
      </template>
    </n-card>

    <n-card title="更新日志" style="margin-top: 16px">
      <template #header-extra>
        <n-space>
          <n-button size="small" @click="fetchLogs">
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新日志
          </n-button>
          <n-button size="small" @click="handleClearLogs">清空日志</n-button>
        </n-space>
      </template>
      <div class="log-viewer" ref="logViewerRef">
        <div v-if="logs.length === 0" class="log-empty">暂无日志</div>
        <div v-for="(entry, index) in logs" :key="index" class="log-entry" :class="'log-' + entry.level">
          <span class="log-time">{{ entry.time }}</span>
          <span class="log-level">[{{ entry.level.toUpperCase() }}]</span>
          <span class="log-message">{{ entry.message }}</span>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { useMessage } from 'naive-ui'
import { RefreshOutline, PlayOutline, StopOutline, ReorderThreeOutline } from '@vicons/ionicons5'
import Sortable from 'sortablejs'
import {
  getConfigUpdateStatus,
  getConfigUpdateConfig,
  saveConfigUpdateConfig,
  startConfigUpdate,
  stopConfigUpdate,
  getConfigUpdateLogs,
  clearConfigUpdateLogs
} from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const message = useMessage()
const logViewerRef = ref(null)
const urlListRef = ref(null)
const starting = ref(false)
const saving = ref(false)
let pollTimer = null

const status = reactive({ running: false, scheduled: false })
const config = reactive({ urls: [], keywords: [], enabled: false, interval: 60 })
const logs = ref([])

const fetchStatus = async () => {
  try {
    const res = await getConfigUpdateStatus()
    Object.assign(status, res.data)
  } catch {}
}

const fetchConfig = async () => {
  try {
    const res = await getConfigUpdateConfig()
    Object.assign(config, res.data)
  } catch {}
}

const fetchLogs = async () => {
  try {
    const res = await getConfigUpdateLogs()
    logs.value = res.data || []
    await nextTick()
    if (logViewerRef.value) {
      logViewerRef.value.scrollTop = logViewerRef.value.scrollHeight
    }
  } catch {}
}

const handleSaveConfig = async () => {
  saving.value = true
  try {
    await saveConfigUpdateConfig(config)
    message.success('配置已保存')
    fetchStatus()
  } catch (error) {
    message.error(error.message || '保存配置失败')
  } finally {
    saving.value = false
  }
}

const addManualPlaceholder = () => {
  if (!config.urls.includes('__MANUAL_NODES__')) {
    config.urls.push('__MANUAL_NODES__')
  }
}

const handleStart = async () => {
  starting.value = true
  try {
    const res = await startConfigUpdate()
    message.success('更新任务已启动，正在采集中...')

    // 立即获取一次日志
    await fetchLogs()

    // 启动快速轮询（每秒一次，持续10秒）
    let fastPollCount = 0
    const fastPollInterval = setInterval(async () => {
      await fetchLogs()
      await fetchStatus()
      fastPollCount++
      if (fastPollCount >= 10) {
        clearInterval(fastPollInterval)
      }
    }, 1000)

  } catch (error) {
    message.error(error.message || '启动失败')
  } finally {
    starting.value = false
  }
}

const handleStop = async () => {
  try {
    await stopConfigUpdate()
    message.success('更新任务已停止')
    fetchStatus()
  } catch (error) {
    message.error(error.message || '停止失败')
  }
}

const handleClearLogs = async () => {
  try {
    await clearConfigUpdateLogs()
    logs.value = []
    message.success('日志已清空')
  } catch (error) {
    message.error(error.message || '清空日志失败')
  }
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(() => {
    fetchLogs()
    fetchStatus()
  }, 3000)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

onMounted(() => {
  fetchStatus()
  fetchConfig()
  fetchLogs()
  startPolling()

  nextTick(() => {
    if (urlListRef.value) {
      Sortable.create(urlListRef.value, {
        animation: 150,
        handle: '.drag-handle',
        ghostClass: 'sortable-ghost',
        onEnd: (evt) => {
          const { oldIndex, newIndex } = evt
          if (oldIndex == null || newIndex == null || oldIndex === newIndex) return

          // Revert the DOM move that Sortable did — let Vue handle rendering
          const parent = evt.from
          const children = parent.children
          if (evt.oldIndex < evt.newIndex) {
            parent.insertBefore(evt.item, children[evt.oldIndex] || null)
          } else {
            parent.insertBefore(evt.item, children[evt.oldIndex + 1] || null)
          }

          // Now update the reactive array — Vue will re-render correctly
          const item = config.urls.splice(oldIndex, 1)[0]
          config.urls.splice(newIndex, 0, item)
        }
      })
    }
  })
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.config-update-container {
  padding: 20px;
}
.log-viewer {
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  padding: 16px;
  border-radius: 6px;
  max-height: 400px;
  overflow-y: auto;
  min-height: 200px;
}
.log-empty {
  color: #666;
  text-align: center;
  padding: 40px 0;
}
.log-entry {
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-all;
}
.log-time {
  color: #858585;
  margin-right: 8px;
}
.log-level {
  margin-right: 8px;
  font-weight: bold;
}
.log-info .log-level { color: #569cd6; }
.log-error .log-level { color: #f44747; }
.log-error .log-message { color: #f44747; }
.log-success .log-level { color: #6a9955; }
.log-success .log-message { color: #6a9955; }

.url-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.url-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background: var(--n-color);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  transition: all 0.3s;
}

.url-item:hover {
  border-color: var(--n-border-color-hover);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.drag-handle {
  cursor: move;
  color: var(--n-text-color-disabled);
  display: flex;
  align-items: center;
  transition: color 0.3s;
  flex-shrink: 0;
}

.drag-handle:hover {
  color: var(--n-primary-color);
}

.url-index {
  font-size: 14px;
  font-weight: 500;
  color: var(--n-text-color-disabled);
  min-width: 24px;
  text-align: center;
  flex-shrink: 0;
}

.url-input {
  flex: 1;
}

.manual-placeholder {
  display: flex;
  align-items: center;
  padding: 0 12px;
  height: 34px;
  border: 1px dashed #e8a838;
  border-radius: 4px;
  background: rgba(232, 168, 56, 0.06);
}

.delete-btn {
  flex-shrink: 0;
}

.sortable-ghost {
  opacity: 0.5;
  background: var(--n-primary-color-hover);
}

@media (max-width: 767px) {
  .config-update-container { padding: 8px; }
  .log-viewer { font-size: 12px; padding: 12px; max-height: 300px; }

  .url-item {
    padding: 6px;
    gap: 6px;
  }

  .drag-handle {
    font-size: 16px;
  }

  .url-index {
    font-size: 12px;
    min-width: 20px;
  }
}
.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
.mobile-toolbar-row { display: flex; gap: 8px; align-items: center; }
</style>
