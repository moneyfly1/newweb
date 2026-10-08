<template>
  <div class="ticket-detail-container">
    <n-card :bordered="false" class="info-card">
      <div class="ticket-header">
        <div class="header-left">
          <n-button text @click="window.history.length > 1 ? router.back() : router.push('/tickets')">
            <template #icon>
              <n-icon><ArrowBackOutline /></n-icon>
            </template>
          </n-button>
          <h2>{{ ticket.title }}</h2>
        </div>
        <div class="header-right">
          <n-button
            v-if="ticket.status !== 'closed'"
            type="error"
            ghost
            @click="handleClose"
            :loading="closing"
          >
            关闭工单
          </n-button>
        </div>
      </div>
      <n-divider style="margin: 16px 0" />
      <div class="ticket-meta">
        <div class="meta-item">
          <span class="meta-label">工单编号：</span>
          <span class="meta-value">{{ ticket.ticket_no }}</span>
        </div>
        <div class="meta-item">
          <span class="meta-label">状态：</span>
          <n-tag :type="getStatusType(ticket.status)">
            {{ getStatusText(ticket.status) }}
          </n-tag>
        </div>
        <div class="meta-item">
          <span class="meta-label">类型：</span>
          <n-tag>{{ getTypeText(ticket.type) }}</n-tag>
        </div>
        <div class="meta-item">
          <span class="meta-label">优先级：</span>
          <n-tag :type="getPriorityType(ticket.priority)">
            {{ getPriorityText(ticket.priority) }}
          </n-tag>
        </div>
        <div class="meta-item">
          <span class="meta-label">创建时间：</span>
          <span class="meta-value">{{ ticket.created_at }}</span>
        </div>
      </div>
    </n-card>

    <n-card :bordered="false" class="chat-card">
      <div class="td-thread" ref="chatContainer">
        <!-- 工单主内容（用户提问） -->
        <div class="td-msg-row user" v-if="ticket.content">
          <div class="td-msg">
            <div class="td-msg-head">
              <span class="td-msg-sender">我</span>
              <span class="td-msg-time">{{ ticket.created_at }}</span>
            </div>
            <div class="td-msg-content">{{ ticket.content }}</div>
            <TicketAttachmentList
              v-if="ticketAttachments.length > 0"
              :key="'main-' + ticket.id"
              :attachments="ticketAttachments"
            />
          </div>
        </div>
        <!-- 历史回复 -->
        <div
          v-for="reply in replies"
          :key="reply.id"
          :class="['td-msg-row', reply.is_admin ? 'admin' : 'user']"
        >
          <div class="td-msg">
            <div class="td-msg-head">
              <span class="td-msg-sender">
                {{ reply.is_admin ? '客服' : '我' }}
              </span>
              <span class="td-msg-time">{{ reply.created_at }}</span>
            </div>
            <div class="td-msg-content">{{ reply.content }}</div>
            <TicketAttachmentList
              v-if="attachmentsByReply[reply.id]?.length"
              :attachments="attachmentsByReply[reply.id]"
            />
          </div>
        </div>
        <div v-if="replies.length === 0 && !ticket.content" class="empty-state">
          暂无回复消息
        </div>
      </div>
    </n-card>

    <n-card
      v-if="ticket.status !== 'closed'"
      :bordered="false"
      class="reply-card"
    >
      <div class="td-composer">
        <n-input
          v-model:value="replyContent"
          type="textarea"
          placeholder="输入您的回复内容..."
          :rows="3"
          maxlength="2000"
          show-count
          @keydown.ctrl.enter="handleReply"
        />
        <TicketAttachmentUploader ref="uploaderRef" @change="(ids) => (attachmentIds = ids)" />
        <n-button
          class="reply-submit"
          type="primary"
          @click="handleReply"
          :loading="replying"
          :disabled="!replyContent.trim() && attachmentIds.length === 0"
        >
          <template #icon>
            <n-icon><SendOutline /></n-icon>
          </template>
          发送回复
        </n-button>
      </div>
    </n-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NIcon, useMessage, useDialog } from 'naive-ui'
import { ArrowBackOutline, SendOutline } from '@vicons/ionicons5'
import { getTicket, replyTicket, closeTicket } from '@/api/ticket'
import TicketAttachmentList from '@/components/TicketAttachmentList.vue'
import TicketAttachmentUploader from '@/components/TicketAttachmentUploader.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const ticket = ref({})
const replies = ref([])
const attachments = ref([])
const replyContent = ref('')
const attachmentIds = ref([])
const uploaderRef = ref(null)
const replying = ref(false)
const closing = ref(false)
const chatContainer = ref(null)

// 工单主内容附件（reply_id 为空）与各回复附件
const ticketAttachments = computed(() =>
  (attachments.value || []).filter(a => a.reply_id === null || a.reply_id === undefined || a.reply_id === 0)
)
const attachmentsByReply = computed(() => {
  const map = {}
  for (const a of attachments.value || []) {
    if (a.reply_id) {
      if (!map[a.reply_id]) map[a.reply_id] = []
      map[a.reply_id].push(a)
    }
  }
  return map
})

const getStatusType = (status) => {
  const map = {
    pending: 'warning',
    processing: 'info',
    resolved: 'success',
    closed: 'default'
  }
  return map[status] || 'default'
}

const getStatusText = (status) => {
  const map = {
    pending: '待处理',
    processing: '处理中',
    resolved: '已解决',
    closed: '已关闭'
  }
  return map[status] || status
}

const getPriorityType = (priority) => {
  const map = {
    low: 'default',
    normal: 'info',
    high: 'warning',
    urgent: 'error'
  }
  return map[priority] || 'default'
}

const getPriorityText = (priority) => {
  const map = {
    low: '低',
    normal: '普通',
    high: '高',
    urgent: '紧急'
  }
  return map[priority] || priority
}

const getTypeText = (type) => {
  const map = {
    technical: '技术问题',
    billing: '账单问题',
    account: '账户问题',
    other: '其他问题'
  }
  return map[type] || type
}

const loadTicket = async () => {
  try {
    const res = await getTicket(route.params.id)
    ticket.value = res.data.ticket || {}
    replies.value = res.data.replies || []
    attachments.value = res.data.attachments || []
    await nextTick()
    scrollToBottom()
  } catch (error) {
    message.error(error.message || '加载工单详情失败')
  }
}

const handleReply = async () => {
  if (replying.value) return // 防 Ctrl+Enter 快速双发
  if (!replyContent.value.trim() && attachmentIds.value.length === 0) {
    message.warning('请输入回复内容或添加附件')
    return
  }

  replying.value = true
  try {
    await replyTicket(route.params.id, {
      content: replyContent.value,
      attachment_ids: attachmentIds.value
    })
    message.success('回复成功')
    replyContent.value = ''
    attachmentIds.value = []
    uploaderRef.value?.reset(false)
    await loadTicket()
  } catch (error) {
    message.error(error.message || '回复失败')
  } finally {
    replying.value = false
  }
}

const handleClose = () => {
  dialog.warning({
    title: '确认关闭',
    content: '关闭后将无法继续回复，确定要关闭此工单吗？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      closing.value = true
      try {
        await closeTicket(route.params.id)
        message.success('工单已关闭')
        await loadTicket()
      } catch (error) {
        message.error(error.message || '关闭工单失败')
      } finally {
        closing.value = false
      }
    }
  })
}

const scrollToBottom = () => {
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}

onMounted(() => {
  loadTicket()
})
</script>

<style scoped>
.ticket-detail-container {
  padding: 20px;
}

.info-card {
  margin-bottom: 20px;
  border-radius: 12px;
}

.ticket-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-left h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.ticket-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-label {
  color: var(--text-color-secondary);
  font-size: 14px;
}

.meta-value {
  font-size: 14px;
}

.chat-card {
  margin-bottom: 20px;
  border-radius: 12px;
}

.td-thread {
  min-height: 400px;
  max-height: 600px;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.td-msg-row {
  display: flex;
  width: 100%;
}

.td-msg-row.user {
  justify-content: flex-end;
}

.td-msg-row.admin {
  justify-content: flex-start;
}

.td-msg {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 12px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.td-msg-row.user .td-msg {
  background: #2080f0;
  color: white;
}

.td-msg-row.admin .td-msg {
  background: var(--bg-page-color);
  color: var(--text-color);
}

.td-msg-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  gap: 12px;
}

.td-msg-sender {
  font-size: 12px;
  font-weight: 600;
  opacity: 0.9;
}

.td-msg-time {
  font-size: 12px;
  opacity: 0.7;
}

.td-msg-content {
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
  color: var(--text-color-secondary);
  font-size: 14px;
}

.reply-card {
  border-radius: 12px;
}

.td-composer {
  display: flex;
  flex-direction: column;
}

@media (max-width: 767px) {
  /* 根容器不再自带左右内边距（全局已统一 10px） */
  .ticket-detail-container { padding-left: 0; padding-right: 0; padding-top: 0; }

  .info-card { margin-bottom: 10px; border-radius: 16px; }
  .chat-card { margin-bottom: 10px; border-radius: 16px; }
  .reply-card { border-radius: 16px; }

  /* 顶部：返回 + 标题 + 关闭工单，可换行不挤压 */
  .ticket-header { flex-wrap: wrap; align-items: flex-start; gap: 8px; }
  .header-left { flex: 1 1 auto; gap: 6px; min-width: 0; }
  .header-left h2 { font-size: 17px; line-height: 1.35; word-break: break-word; }
  .header-right { flex-shrink: 0; }
  .header-right :deep(.n-button) { min-height: 40px; }

  /* 工单元信息：App 的 label 左灰字 / value 右深字 行 */
  .ticket-meta { flex-direction: column; gap: 0; }
  .meta-item {
    justify-content: space-between; gap: 10px; min-height: 40px; padding: 7px 0;
    border-bottom: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 55%, transparent);
  }
  .meta-item:last-child { border-bottom: none; }
  .meta-label { flex-shrink: 0; font-size: 13px; }
  .meta-value { min-width: 0; font-size: 13px; text-align: right; word-break: break-word; }

  /* 消息列表：整宽气泡，聊天/时间线观感 */
  .td-thread { padding: 0; gap: 10px; min-height: 240px; max-height: 58vh; }
  .td-msg-row { width: 100%; }
  .td-msg { width: 100%; max-width: 100%; padding: 12px 14px; border-radius: 14px; box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04); }
  .td-msg-row.user .td-msg { background: var(--primary-color, #2080f0); color: #fff; }
  .td-msg-row.admin .td-msg {
    background: var(--bg-page-color, #f5f6f8); color: var(--text-color);
    border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 60%, transparent);
  }
  .td-msg-head { margin-bottom: 6px; gap: 8px; }
  .td-msg-sender { font-size: 12px; }
  .td-msg-time { font-size: 12px; opacity: 0.8; text-align: right; word-break: break-word; }
  .td-msg-content { font-size: 14px; line-height: 1.65; word-break: break-word; overflow-wrap: anywhere; }

  /* 附件（共享组件，从页面侧兜住长文件名/大图，避免横向溢出） */
  .td-msg :deep(.att-file-card) { width: 100%; min-width: 0; max-width: 100%; }
  .td-msg :deep(.att-file-name) { font-size: 13px; white-space: normal; word-break: break-word; }
  .td-msg :deep(.att-file-size) { font-size: 12px; }
  .td-msg :deep(.att-img), .td-msg :deep(.att-video) { max-width: 100%; }

  /* 底部回复区：整宽提交按钮，触控目标 ≥ 40px */
  .td-composer :deep(.n-input) { min-height: 96px; }
  .reply-submit { width: 100%; align-self: stretch; margin-top: 12px; min-height: 44px; font-size: 15px; border-radius: 12px; }

  .empty-state { height: 140px; font-size: 14px; }
}

@media (max-width: 400px) {
  .td-msg-head { flex-wrap: wrap; }
}
</style>
