<template>
  <div v-if="attachments.length > 0" class="att-list">
    <div
      v-for="att in attachments"
      :key="att.id"
      class="att-item"
      :class="{ 'att-media': isImage(att) || isVideo(att) }"
    >
      <!-- 图片：加载成功后展示 -->
      <template v-if="isImage(att)">
        <div v-if="!blobUrls[att.id]" class="att-loading">
          <n-spin size="small" />
        </div>
        <n-image
          v-else
          :src="blobUrls[att.id]"
          :alt="att.file_name"
          object-fit="cover"
          class="att-img"
          :preview-src="blobUrls[att.id]"
        />
      </template>
      <!-- 视频 -->
      <template v-else-if="isVideo(att)">
        <div v-if="!blobUrls[att.id]" class="att-loading">
          <n-spin size="small" />
        </div>
        <video
          v-else
          :src="blobUrls[att.id]"
          controls
          preload="metadata"
          class="att-video"
        ></video>
      </template>
      <!-- 其他文件：下载卡片 -->
      <template v-else>
        <div class="att-file-card" @click="download(att)">
          <div class="att-file-icon">
            <n-icon><DocumentOutline /></n-icon>
          </div>
          <div class="att-file-info">
            <div class="att-file-name" :title="att.file_name">{{ att.file_name }}</div>
            <div class="att-file-size">{{ formatSize(att.file_size) }}</div>
          </div>
          <n-button text size="small" type="primary">
            <template #icon><n-icon><DownloadOutline /></n-icon></template>
          </n-button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch, onBeforeUnmount } from 'vue'
import { NIcon, NSpin, NImage, useMessage } from 'naive-ui'
import { DocumentOutline, DownloadOutline } from '@vicons/ionicons5'
import request from '@/utils/request'

export interface TicketAttachmentItem {
  id: number
  file_name: string
  file_type?: string | null
  file_size?: number | null
  url: string
}

const props = defineProps<{ attachments: TicketAttachmentItem[] }>()

const message = useMessage()
const blobUrls = reactive<Record<number, string>>({})
const objectUrls: string[] = []

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/bmp', 'image/svg+xml']
const VIDEO_TYPES = ['video/mp4', 'video/webm', 'video/quicktime', 'video/x-msvideo', 'video/x-matroska']

function isImage(att: TicketAttachmentItem): boolean {
  return IMAGE_TYPES.includes(att.file_type || '') || /\.(jpe?g|png|gif|webp|bmp|svg)$/i.test(att.file_name)
}
function isVideo(att: TicketAttachmentItem): boolean {
  return VIDEO_TYPES.includes(att.file_type || '') || /\.(mp4|webm|mov|avi|mkv|m4v)$/i.test(att.file_name)
}

function formatSize(size?: number | null): string {
  if (!size) return ''
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

async function loadBlob(att: TicketAttachmentItem) {
  if (blobUrls[att.id]) return
  try {
    const res: any = await request.get(att.url, { responseType: 'blob' })
    const blob = res?.data instanceof Blob ? res.data : res instanceof Blob ? res : null
    if (!blob) return
    // 服务端可能返回 JSON 错误（如 403/404），此时 data 不是 Blob 而是字符串
    if (blob.type && blob.type.includes('application/json')) {
      const text = await blob.text()
      const json = JSON.parse(text)
      message.error(json.message || '加载附件失败')
      return
    }
    const url = URL.createObjectURL(blob)
    objectUrls.push(url)
    blobUrls[att.id] = url
  } catch { /* 静默：单个附件加载失败不影响其余 */ }
}

async function download(att: TicketAttachmentItem) {
  if (!blobUrls[att.id]) {
    await loadBlob(att)
  }
  if (!blobUrls[att.id]) return
  const a = document.createElement('a')
  a.href = blobUrls[att.id]
  a.download = att.file_name
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

watch(() => props.attachments, (list) => {
  for (const att of list || []) {
    if (isImage(att) || isVideo(att)) loadBlob(att)
  }
}, { immediate: true, deep: false })

onBeforeUnmount(() => {
  objectUrls.forEach(u => URL.revokeObjectURL(u))
  objectUrls.length = 0
})
</script>

<style scoped>
.att-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.att-item {
  position: relative;
}
.att-media {
  max-width: 100%;
}
.att-img {
  max-width: 220px;
  max-height: 220px;
  border-radius: 8px;
  display: block;
  cursor: zoom-in;
}
.att-video {
  max-width: 300px;
  max-height: 220px;
  border-radius: 8px;
  display: block;
}
.att-loading {
  width: 120px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 8px;
}
.att-file-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--divider-color, #e5e5e5);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  min-width: 200px;
  max-width: 280px;
}
.att-file-icon {
  font-size: 22px;
  color: #f0a020;
  display: flex;
  align-items: center;
}
.att-file-info {
  flex: 1;
  min-width: 0;
}
.att-file-name {
  font-size: 13px;
  color: var(--text-color, #333);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.att-file-size {
  font-size: 12px;
  color: var(--text-color-3, #999);
}
</style>
