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
      <div class="chat-container" ref="chatContainer">
        <div
          v-for="reply in replies"
          :key="reply.id"
          :class="['message-wrapper', reply.is_admin ? 'admin' : 'user']"
        >
          <div class="message-bubble">
            <div class="message-header">
              <span class="message-sender">
                {{ reply.is_admin ? '客服' : '我' }}
              </span>
              <span class="message-time">{{ reply.created_at }}</span>
            </div>
            <div class="message-content">{{ reply.content }}</div>
          </div>
        </div>
        <div v-if="replies.length === 0" class="empty-state">
          暂无回复消息
        </div>
      </div>
    </n-card>

    <n-card
      v-if="ticket.status !== 'closed'"
      :bordered="false"
      class="reply-card"
    >
      <div class="reply-input-wrapper">
        <n-input
          v-model:value="replyContent"
          type="textarea"
          placeholder="输入您的回复内容..."
          :rows="3"
          maxlength="2000"
          show-count
          @keydown.ctrl.enter="handleReply"
        />
        <n-button
          type="primary"
          @click="handleReply"
          :loading="replying"
          :disabled="!replyContent.trim()"
          style="margin-top: 12px; align-self: flex-end"
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
import { ref, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NIcon, useMessage, useDialog } from 'naive-ui'
import { ArrowBackOutline, SendOutline } from '@vicons/ionicons5'
import { getTicket, replyTicket, closeTicket } from '@/api/ticket'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const ticket = ref({})
const replies = ref([])
const replyContent = ref('')
const replying = ref(false)
const closing = ref(false)
const chatContainer = ref(null)

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
    await nextTick()
    scrollToBottom()
  } catch (error) {
    message.error(error.message || '加载工单详情失败')
  }
}

const handleReply = async () => {
  if (!replyContent.value.trim()) {
    message.warning('请输入回复内容')
    return
  }
  
  replying.value = true
  try {
    await replyTicket(route.params.id, {
      content: replyContent.value
    })
    message.success('回复成功')
    replyContent.value = ''
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
  color: #999;
  font-size: 14px;
}

.meta-value {
  font-size: 14px;
}

.chat-card {
  margin-bottom: 20px;
  border-radius: 12px;
}

.chat-container {
  min-height: 400px;
  max-height: 600px;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-wrapper {
  display: flex;
  width: 100%;
}

.message-wrapper.user {
  justify-content: flex-end;
}

.message-wrapper.admin {
  justify-content: flex-start;
}

.message-bubble {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 12px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.message-wrapper.user .message-bubble {
  background: #2080f0;
  color: white;
}

.message-wrapper.admin .message-bubble {
  background: #f5f5f5;
  color: #333;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  gap: 12px;
}

.message-sender {
  font-size: 12px;
  font-weight: 600;
  opacity: 0.9;
}

.message-time {
  font-size: 11px;
  opacity: 0.7;
}

.message-content {
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
  color: #999;
  font-size: 14px;
}

.reply-card {
  border-radius: 12px;
}

.reply-input-wrapper {
  display: flex;
  flex-direction: column;
}

@media (max-width: 767px) {
  .ticket-detail-container { padding: 0 12px; }
  .ticket-meta { flex-direction: column; gap: 8px; }
  .message-bubble { max-width: 85%; }
  .chat-container { min-height: 300px; max-height: 50vh; }
}
</style>
