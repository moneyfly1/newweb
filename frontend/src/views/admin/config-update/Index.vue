<template>
  <div class="config-update-container">
    <n-card title="节点自动更新">
      <template #header-extra>
        <n-space>
          <n-button @click="fetchStatus">
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新状态
          </n-button>
          <n-button type="primary" :loading="starting" :disabled="status.running" @click="handleStart">
            <template #icon><n-icon><PlayOutline /></n-icon></template>
            立即更新
          </n-button>
          <n-button type="warning" :disabled="!status.running" @click="handleStop">
            <template #icon><n-icon><StopOutline /></n-icon></template>
            停止
          </n-button>
        </n-space>
      </template>
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
            <n-dynamic-input v-model:value="config.urls" placeholder="请输入订阅URL" />
          </div>
        </n-form-item>
        <n-form-item label="关键词过滤">
          <div style="width: 100%">
            <n-dynamic-input v-model:value="config.keywords" placeholder="输入关键词（匹配节点名称）" />
            <n-text depth="3" style="font-size: 12px; margin-top: 4px; display: block">
              留空表示不过滤，导入所有节点。填写后只导入名称包含关键词的节点。
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
        <n-button size="small" @click="handleClearLogs">清空日志</n-button>
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
import { RefreshOutline, PlayOutline, StopOutline } from '@vicons/ionicons5'
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

const handleStart = async () => {
  starting.value = true
  try {
    await startConfigUpdate()
    message.success('更新任务已启动')
    fetchStatus()
    startPolling()
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

@media (max-width: 767px) {
  .config-update-container { padding: 8px; }
  .log-viewer { font-size: 12px; padding: 12px; max-height: 300px; }
}
</style>
