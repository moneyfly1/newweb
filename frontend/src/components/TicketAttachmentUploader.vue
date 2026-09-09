<template>
  <div class="ticket-uploader">
    <input
      ref="fileInput"
      type="file"
      multiple
      :accept="acceptAttr"
      style="display: none"
      @change="onFilesChosen"
    />
    <div class="uploader-bar">
      <n-button size="small" :loading="uploading" :disabled="uploading" @click="fileInput?.click()">
        <template #icon>
          <n-icon><AttachOutline /></n-icon>
        </template>
        添加图片/视频/附件
      </n-button>
      <span class="upload-hint">支持图片、视频、PDF、文档、压缩包等，单个不超过 20MB</span>
    </div>

    <n-spin :show="uploading">
      <div v-if="items.length > 0" class="upload-list">
        <div v-for="item in items" :key="item.uid" class="upload-item">
          <div class="item-icon" :class="iconClass(item.file_name)">
            <n-icon><component :is="itemIcon(item.file_name)" /></n-icon>
          </div>
          <div class="item-info">
            <div class="item-name">{{ item.file_name }}</div>
            <div class="item-size">{{ formatSize(item.file_size) }}</div>
          </div>
          <n-button
            v-if="!uploading"
            text
            type="error"
            size="small"
            class="item-remove"
            @click="removeItem(item)"
          >
            <template #icon><n-icon><TrashOutline /></n-icon></template>
          </n-button>
        </div>
      </div>
    </n-spin>
    <div v-if="errorMsg" class="upload-error">{{ errorMsg }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { NButton, NIcon, NSpin, useMessage } from 'naive-ui'
import { AttachOutline, TrashOutline, DocumentOutline, ImageOutline, VideocamOutline } from '@vicons/ionicons5'
import { uploadTicketAttachment, deleteTicketAttachment } from '@/api/ticket'

const message = useMessage()

// 允许的扩展名（与后端白名单一致）
const ALLOWED_EXTS = [
  '.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg',
  '.mp4', '.mov', '.avi', '.mkv', '.webm', '.m4v',
  '.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx',
  '.txt', '.log', '.zip', '.rar', '.7z', '.tar', '.gz',
  '.csv', '.json', '.md',
]
const MAX_SIZE = 20 * 1024 * 1024

const props = defineProps<{
  // 已绑定附件数上限提示（可选）
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'change', ids: number[]): void
}>()

const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const errorMsg = ref('')

export interface UploadedItem {
  uid: number // 本地唯一 id
  id: number // 服务端附件 id（pending）
  file_name: string
  file_size: number
}

const items = ref<UploadedItem[]>([])

const acceptAttr = computed(() => ALLOWED_EXTS.join(',') + ',image/*,video/*')

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])
const VIDEO_EXTS = new Set(['.mp4', '.mov', '.avi', '.mkv', '.webm', '.m4v'])

function extOf(name: string): string {
  const idx = name.lastIndexOf('.')
  return idx >= 0 ? name.slice(idx).toLowerCase() : ''
}

function iconClass(name: string): string {
  const ext = extOf(name)
  if (IMAGE_EXTS.has(ext)) return 'icon-image'
  if (VIDEO_EXTS.has(ext)) return 'icon-video'
  return 'icon-doc'
}

function itemIcon(name: string) {
  const ext = extOf(name)
  if (IMAGE_EXTS.has(ext)) return ImageOutline
  if (VIDEO_EXTS.has(ext)) return VideocamOutline
  return DocumentOutline
}

function formatSize(size: number): string {
  if (!size && size !== 0) return ''
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  if (size < 1024 * 1024 * 1024) return (size / 1024 / 1024).toFixed(1) + ' MB'
  return (size / 1024 / 1024 / 1024).toFixed(2) + ' GB'
}

let uidSeq = 1

const onFilesChosen = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = '' // 允许重复选择同一文件
  if (files.length === 0) return
  if (props.disabled) return

  // 逐个校验
  const invalid: string[] = []
  const valid: File[] = []
  for (const f of files) {
    const ext = extOf(f.name)
    if (!ALLOWED_EXTS.includes(ext)) {
      invalid.push(`${f.name}（不支持该类型）`)
      continue
    }
    if (f.size > MAX_SIZE) {
      invalid.push(`${f.name}（超过 20MB）`)
      continue
    }
    valid.push(f)
  }
  if (invalid.length > 0) {
    message.error('以下文件无法上传：' + invalid.join('；'))
  }
  if (valid.length === 0) return
  if (items.value.length + valid.length > 5) {
    message.error('每个工单/回复最多附带 5 个文件')
    return
  }

  uploading.value = true
  errorMsg.value = ''
  try {
    const formData = new FormData()
    for (const f of valid) formData.append('files', f)
    const res = await uploadTicketAttachment(formData)
    const uploaded = (res.data?.files || []) as any[]
    // 失败时清掉这次成功项？不：本次整体成功才展示；部分失败由后端 400 整批拒绝
    for (const u of uploaded) {
      items.value.push({ uid: uidSeq++, id: u.id, file_name: u.file_name, file_size: u.file_size })
    }
    emitChange()
    if (uploaded.length > 0) message.success(`已上传 ${uploaded.length} 个文件`)
  } catch (err: any) {
    errorMsg.value = err.message || '上传失败'
    message.error(err.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

const removeItem = async (item: UploadedItem) => {
  const idx = items.value.findIndex(i => i.uid === item.uid)
  if (idx < 0) return
  items.value.splice(idx, 1)
  emitChange()
  // 异步删除服务端 pending 记录（失败静默——后端 24h 会自动清理孤儿）
  try {
    await deleteTicketAttachment(item.id)
  } catch { /* ignore */ }
}

function emitChange() {
  emit('change', items.value.map(i => i.id))
}

// 暴露给父组件：
// reset(deletePending): 提交成功后调用（false，附件已绑定不可删）；
//   抽屉取消/发送失败后调用（true，删除服务端 pending 记录避免孤儿）
function reset(deletePending = false) {
  if (deletePending) {
    const ids = items.value.map(i => i.id)
    items.value = []
    emitChange()
    for (const id of ids) {
      deleteTicketAttachment(id).catch(() => {})
    }
  } else {
    items.value = []
    emitChange()
  }
  errorMsg.value = ''
}

defineExpose({ reset, getIds: () => items.value.map(i => i.id), get count() { return items.value.length } })

onBeforeUnmount(() => {})
</script>

<style scoped>
.ticket-uploader {
  margin-top: 8px;
}
.uploader-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.upload-hint {
  font-size: 12px;
  color: var(--text-color-3, #999);
}
.upload-list {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.upload-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  border: 1px solid var(--divider-color, #eee);
  border-radius: 8px;
  background: var(--bg-color-secondary, #fafafa);
}
.item-icon {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}
.icon-image { background: rgba(32, 128, 240, 0.12); color: #2080f0; }
.icon-video { background: rgba(24, 160, 88, 0.12); color: #18a058; }
.icon-doc   { background: rgba(240, 160, 32, 0.12); color: #f0a020; }
.item-info {
  flex: 1;
  min-width: 0;
}
.item-name {
  font-size: 13px;
  color: var(--text-color, #333);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.item-size {
  font-size: 12px;
  color: var(--text-color-3, #999);
}
.item-remove { flex-shrink: 0; }
.upload-error {
  margin-top: 6px;
  font-size: 12px;
  color: #d03050;
}
</style>
