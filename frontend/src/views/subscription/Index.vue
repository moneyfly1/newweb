<template>
  <div class="subscription-page">
    <n-spin :show="loading">
      <n-space vertical :size="24">
        <!-- No Subscription State -->
        <div v-if="!subscription" class="empty-state">
          <n-empty description="您还没有订阅">
            <template #extra>
              <n-button type="primary" size="large" @click="$router.push('/shop')">购买套餐</n-button>
            </template>
          </n-empty>
        </div>

        <template v-else>
          <!-- Hero Card -->
          <n-card class="hero-card" :bordered="false">
            <div class="hero-content">
              <div class="hero-top">
                <div class="status-section">
                  <div class="status-badge" :class="statusClass">
                    <n-icon :size="20" :component="statusIcon" />
                    <span>{{ statusText }}</span>
                  </div>
                  <h2 class="package-name">{{ subscription.package_name || '当前套餐' }}</h2>
                </div>
                <div class="hero-stats">
                  <div class="hero-stat">
                    <span class="hero-stat-val">{{ remainingDays }}</span>
                    <span class="hero-stat-label">剩余天数</span>
                  </div>
                  <div class="hero-stat-divider"></div>
                  <div class="hero-stat">
                    <span class="hero-stat-val">{{ devices.length }}/{{ subscription.device_limit || 0 }}</span>
                    <span class="hero-stat-label">设备使用</span>
                  </div>
                </div>
              </div>
              <div class="hero-meta">
                <span><n-icon :component="TimeOutline" :size="16" /> 到期：{{ formatDate(subscription.expire_time) }}</span>
              </div>
            </div>
          </n-card>

          <!-- Subscription URL Section -->
          <n-card :bordered="false" class="section-card">
            <template #header><span class="section-title">订阅地址</span></template>
            <n-space vertical :size="16">
              <!-- Universal URL -->
              <div class="url-row">
                <span class="url-label">通用订阅</span>
                <div class="url-input-wrapper">
                  <n-input :value="subscriptionUrl" readonly size="small" class="url-input" />
                  <n-button size="small" type="primary" @click="copyToClipboard(subscriptionUrl, '通用订阅地址')">
                    <template #icon><n-icon :component="CopyOutline" /></template>
                    复制
                  </n-button>
                  <n-button size="small" @click="showQrCode(subscriptionUrl, '通用订阅')">
                    <template #icon><n-icon :component="QrCodeOutline" /></template>
                  </n-button>
                </div>
              </div>
              <!-- Clash URL -->
              <div class="url-row">
                <span class="url-label">Clash 订阅</span>
                <div class="url-input-wrapper">
                  <n-input :value="clashUrl" readonly size="small" class="url-input" />
                  <n-button size="small" type="primary" @click="copyToClipboard(clashUrl, 'Clash 订阅地址')">
                    <template #icon><n-icon :component="CopyOutline" /></template>
                    复制
                  </n-button>
                  <n-button size="small" @click="showQrCode(clashUrl, 'Clash 订阅')">
                    <template #icon><n-icon :component="QrCodeOutline" /></template>
                  </n-button>
                </div>
              </div>
            </n-space>
          </n-card>

          <!-- Format Selector -->
          <n-card :bordered="false" class="section-card">
            <template #header><span class="section-title">选择订阅格式</span></template>
            <div class="format-grid">
              <div
                v-for="fmt in formats"
                :key="fmt.type"
                class="format-card"
                :class="{ active: selectedFormat === fmt.type }"
                @click="selectedFormat = fmt.type"
              >
                <div class="format-icon">{{ fmt.icon }}</div>
                <div class="format-name">{{ fmt.name }}</div>
                <div class="format-desc">{{ fmt.desc }}</div>
                <n-space :size="8" style="margin-top: 10px;">
                  <n-button size="tiny" type="primary" @click.stop="copyToClipboard(getFormatUrl(fmt.type), fmt.name)">复制</n-button>
                  <n-button size="tiny" @click.stop="importFormat(fmt)">一键导入</n-button>
                </n-space>
              </div>
            </div>
          </n-card>

          <!-- Connected Devices -->
          <n-card :bordered="false" class="section-card">
            <template #header>
              <div class="section-header-row">
                <span class="section-title">已连接设备</span>
                <n-tag :bordered="false" type="info">{{ devices.length }} / {{ subscription?.device_limit || 0 }} 台</n-tag>
              </div>
            </template>
            <n-space vertical :size="12">
              <n-card v-for="device in devices" :key="device.id" class="device-card" :bordered="false">
                <div class="device-content">
                  <div class="device-info">
                    <n-icon :size="24" :component="PhonePortraitOutline" class="device-icon" />
                    <div class="device-details">
                      <div class="device-name">{{ device.device_name || '未知设备' }}</div>
                      <div class="device-meta">
                        <span>{{ device.ip || 'N/A' }}</span>
                        <span class="sep">|</span>
                        <span>{{ formatDate(device.last_used) }}</span>
                      </div>
                    </div>
                  </div>
                  <n-button text type="error" @click="handleDeleteDevice(device.id)">
                    <template #icon><n-icon :component="TrashOutline" /></template>
                    删除
                  </n-button>
                </div>
              </n-card>
              <n-empty v-if="devices.length === 0" description="暂无已连接设备" />
            </n-space>
          </n-card>

          <!-- Action Buttons -->
          <div class="action-section">
            <n-space :size="16" :wrap="true">
              <n-button type="warning" size="large" @click="showResetModal = true" :disabled="!subscription">
                <template #icon><n-icon :component="RefreshOutline" /></template>
                重置订阅
              </n-button>
              <n-button type="info" size="large" @click="showConvertModal = true" :disabled="!subscription || !canConvert">
                <template #icon><n-icon :component="SwapHorizontalOutline" /></template>
                转换为余额
              </n-button>
              <n-button size="large" @click="handleSendEmail" :loading="sendingEmail">
                <template #icon><n-icon :component="MailOutline" /></template>
                发送到邮箱
              </n-button>
            </n-space>
          </div>
        </template>
      </n-space>
    </n-spin>

    <!-- QR Code Modal -->
    <n-modal v-model:show="showQrModal" preset="card" :title="qrTitle + ' 二维码'" style="width: 340px; max-width: 92vw;" :bordered="false">
      <div style="text-align: center;">
        <canvas ref="qrCanvas" style="margin: 0 auto;"></canvas>
        <p style="margin-top: 12px; color: #999; font-size: 13px;">使用客户端扫描二维码导入订阅</p>
      </div>
    </n-modal>

    <!-- Reset Modal -->
    <n-modal v-model:show="showResetModal" preset="dialog" title="重置订阅地址"
      content="重置后原订阅地址将失效，所有设备需要重新配置。确定要继续吗？"
      positive-text="确定" negative-text="取消" @positive-click="handleResetSubscription" />

    <!-- Convert Modal -->
    <n-modal v-model:show="showConvertModal" preset="dialog" title="转换剩余天数"
      :content="`将剩余 ${remainingDays} 天转换为余额，转换后订阅将立即失效。确定要继续吗？`"
      positive-text="确定" negative-text="取消" @positive-click="handleConvertToBalance" />

    <!-- Delete Device Modal -->
    <n-modal v-model:show="showDeleteModal" preset="dialog" title="删除设备"
      content="确定要删除此设备吗？" positive-text="确定" negative-text="取消"
      @positive-click="confirmDeleteDevice" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import {
  CopyOutline, TimeOutline, PhonePortraitOutline, TrashOutline,
  RefreshOutline, SwapHorizontalOutline, MailOutline,
  CheckmarkCircle, CloseCircle, AlertCircle, QrCodeOutline
} from '@vicons/ionicons5'
import {
  getSubscription, getSubscriptionDevices, deleteDevice,
  resetSubscription, convertToBalance, sendSubscriptionEmail
} from '@/api/subscription'

const message = useMessage()

const subscription = ref<any>(null)
const devices = ref<any[]>([])
const loading = ref(false)
const showResetModal = ref(false)
const showConvertModal = ref(false)
const showDeleteModal = ref(false)
const deviceToDelete = ref<number | null>(null)
const sendingEmail = ref(false)
const selectedFormat = ref('clash')

const showQrModal = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrTitle = ref('')

const formats = [
  { type: 'clash', name: 'Clash', icon: '\u2694\uFE0F', desc: 'Clash 系列客户端' },
  { type: 'v2ray', name: 'V2Ray Base64', icon: '\uD83D\uDE80', desc: '通用 V2Ray 格式' },
  { type: 'surge', name: 'Surge', icon: '\uD83C\uDF0A', desc: 'Surge 客户端' },
  { type: 'quantumult', name: 'Quantumult X', icon: '\u2716', desc: 'Quantumult X' },
  { type: 'shadowrocket', name: 'Shadowrocket', icon: '\uD83D\uDD25', desc: '小火箭客户端' },
  { type: 'stash', name: 'Stash', icon: '\uD83D\uDC8E', desc: 'Stash 客户端' },
]

const subscriptionUrl = computed(() => subscription.value?.universal_url || '')
const clashUrl = computed(() => subscription.value?.clash_url || '')

const getFormatUrl = (type: string) => {
  if (!subscription.value) return ''
  // Clash-based formats use the clash URL, others use universal
  if (type === 'clash' || type === 'stash') {
    return clashUrl.value
  }
  return subscriptionUrl.value
}

const statusClass = computed(() => {
  if (!subscription.value) return 'status-none'
  const now = new Date()
  const exp = new Date(subscription.value.expire_time)
  return exp < now ? 'status-expired' : 'status-active'
})

const statusText = computed(() => {
  if (!subscription.value) return '暂无订阅'
  const now = new Date()
  const exp = new Date(subscription.value.expire_time)
  return exp < now ? '已过期' : '使用中'
})

const statusIcon = computed(() => {
  if (!subscription.value) return AlertCircle
  const now = new Date()
  const exp = new Date(subscription.value.expire_time)
  return exp < now ? CloseCircle : CheckmarkCircle
})

const remainingDays = computed(() => {
  if (!subscription.value) return 0
  const diff = new Date(subscription.value.expire_time).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
})

const canConvert = computed(() => remainingDays.value > 0)

const formatDate = (dateStr: string) => {
  if (!dateStr) return 'N/A'
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  })
}

const copyToClipboard = async (text: string, label: string) => {
  if (!text) { message.warning('暂无可用订阅'); return }
  try {
    await navigator.clipboard.writeText(text)
    message.success(`${label}已复制到剪贴板`)
  } catch { message.error('复制失败，请手动复制') }
}

const importFormat = (fmt: any) => {
  const url = getFormatUrl(fmt.type)
  if (!url) { message.warning('暂无可用订阅'); return }
  const schemeMap: Record<string, string> = {
    clash: 'clash://install-config?url=',
    stash: 'stash://install-config?url=',
    shadowrocket: 'sub://',
    surge: 'surge:///install-config?url=',
    quantumult: 'quantumult-x:///update-configuration?remote-resource=',
  }
  const scheme = schemeMap[fmt.type]
  if (scheme) {
    if (fmt.type === 'shadowrocket') {
      // Shadowrocket uses sub://BASE64(url)
      window.location.href = scheme + btoa(url)
    } else {
      window.location.href = scheme + encodeURIComponent(url)
    }
  } else {
    copyToClipboard(url, fmt.name)
  }
}

const showQrCode = async (url: string, label: string) => {
  if (!url) { message.warning('暂无可用订阅'); return }
  qrTitle.value = label
  showQrModal.value = true
  await nextTick()
  if (qrCanvas.value) {
    QRCode.toCanvas(qrCanvas.value, url, { width: 240, margin: 2 })
  }
}

const loadData = async () => {
  loading.value = true
  try {
    const [subRes, devRes] = await Promise.all([getSubscription(), getSubscriptionDevices()])
    subscription.value = subRes.data
    devices.value = devRes.data || []
  } catch (e: any) {
    if (e?.response?.status !== 404) message.error(e.message || '加载数据失败')
  } finally { loading.value = false }
}

const handleDeleteDevice = (id: number) => { deviceToDelete.value = id; showDeleteModal.value = true }

const confirmDeleteDevice = async () => {
  if (!deviceToDelete.value) return
  try { await deleteDevice(deviceToDelete.value); message.success('设备已删除'); await loadData() }
  catch (e: any) { message.error(e.message || '删除设备失败') }
  finally { deviceToDelete.value = null }
}

const handleResetSubscription = async () => {
  try { await resetSubscription(); message.success('订阅地址已重置'); await loadData() }
  catch (e: any) { message.error(e.message || '重置订阅失败') }
}

const handleConvertToBalance = async () => {
  try { await convertToBalance(); message.success('转换成功'); await loadData() }
  catch (e: any) { message.error(e.message || '转换失败') }
}

const handleSendEmail = async () => {
  sendingEmail.value = true
  try { await sendSubscriptionEmail(); message.success('订阅信息已发送到您的邮箱') }
  catch (e: any) { message.error(e.message || '发送失败') }
  finally { sendingEmail.value = false }
}

onMounted(() => { loadData() })
</script>

<style scoped>
.subscription-page { max-width: 1200px; margin: 0 auto; padding: 24px; }

.empty-state { padding: 80px 0; text-align: center; }

.hero-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px; overflow: hidden;
}
.hero-card :deep(.n-card__content) { padding: 32px; }
.hero-content { color: white; }
.hero-top { display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 20px; }
.status-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 14px; border-radius: 20px; font-size: 14px; font-weight: 600; margin-bottom: 8px;
}
.status-badge.status-active { background: rgba(82,196,26,0.2); border: 2px solid rgba(82,196,26,0.6); }
.status-badge.status-expired { background: rgba(255,77,79,0.2); border: 2px solid rgba(255,77,79,0.6); }
.status-badge.status-none { background: rgba(250,173,20,0.2); border: 2px solid rgba(250,173,20,0.6); }
.package-name { font-size: 22px; font-weight: 700; margin: 0; }
.hero-stats { display: flex; align-items: center; gap: 24px; }
.hero-stat { text-align: center; }
.hero-stat-val { display: block; font-size: 28px; font-weight: 700; }
.hero-stat-label { font-size: 12px; opacity: 0.8; }
.hero-stat-divider { width: 1px; height: 40px; background: rgba(255,255,255,0.3); }
.hero-meta { margin-top: 16px; font-size: 14px; opacity: 0.9; display: flex; align-items: center; gap: 4px; }

.section-card { border-radius: 12px; }
.section-title { font-weight: 600; }
.section-header-row { display: flex; justify-content: space-between; align-items: center; width: 100%; }

.url-row { display: flex; flex-direction: column; gap: 6px; }
.url-label { font-size: 13px; color: #666; font-weight: 500; }
.url-input-wrapper { display: flex; gap: 8px; }
.url-input { flex: 1; }
.url-input :deep(.n-input__input-el) { font-family: 'Monaco','Menlo',monospace; font-size: 12px; }

.format-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 14px;
}
.format-card {
  background: white; border: 2px solid #e8e8e8; border-radius: 12px;
  padding: 20px 14px; text-align: center; cursor: pointer; transition: all 0.3s ease;
}
.format-card:hover, .format-card.active { border-color: #667eea; transform: translateY(-3px); box-shadow: 0 6px 20px rgba(102,126,234,0.12); }
.format-icon { font-size: 28px; margin-bottom: 8px; }
.format-name { font-size: 15px; font-weight: 600; color: #333; margin-bottom: 2px; }
.format-desc { font-size: 11px; color: #999; }

.device-card { border-radius: 10px; border: 1px solid #e8e8e8; transition: all 0.2s ease; }
.device-card:hover { border-color: #d0d0d0; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.device-content { display: flex; justify-content: space-between; align-items: center; }
.device-info { display: flex; align-items: center; gap: 14px; flex: 1; }
.device-icon { color: #667eea; }
.device-name { font-size: 14px; font-weight: 500; color: #333; margin-bottom: 2px; }
.device-meta { font-size: 12px; color: #999; }
.sep { margin: 0 6px; }

.action-section { margin-top: 8px; padding-top: 20px; border-top: 1px solid #e8e8e8; }

/* Mobile Responsive */
@media (max-width: 767px) {
  .subscription-page { padding: 0; }
  .empty-state { padding: 40px 0; }

  .hero-card :deep(.n-card__content) { padding: 20px 16px; }
  .hero-top { flex-direction: column; gap: 16px; }
  .hero-stats { gap: 16px; }
  .hero-stat-val { font-size: 22px; }
  .package-name { font-size: 18px; }

  .url-input-wrapper { flex-direction: column; }
  .url-input-wrapper .n-button { width: 100%; }

  .format-grid { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .format-card { padding: 14px 10px; }
  .format-icon { font-size: 22px; }
  .format-name { font-size: 13px; }
  .format-desc { font-size: 10px; }

  .device-content { flex-direction: column; align-items: flex-start; gap: 10px; }
  .device-info { width: 100%; }

  .action-section .n-space { flex-direction: column; width: 100%; }
  .action-section .n-button { width: 100%; }
}
</style>