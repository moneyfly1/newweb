<!-- 统一的详情抽屉组件 -->
<template>
  <n-drawer
    :show="show"
    :width="drawerWidth"
    :placement="placement"
    :close-on-esc="closeOnEsc"
    :mask-closable="maskClosable"
    @update:show="(val: boolean) => emit('update:show', val)"
  >
    <n-drawer-content :title="title" :closable="closable">
      <template v-if="$slots.header" #header>
        <slot name="header" />
      </template>

      <slot />

      <template #footer>
        <slot v-if="$slots.footer" name="footer" />
        <n-space v-else-if="showFooter" justify="end">
          <n-button v-if="showCancel" @click="handleCancel">{{ cancelText }}</n-button>
          <n-button v-if="showConfirm" type="primary" :loading="loading" @click="handleConfirm">{{ confirmText }}</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'

interface Props {
  show: boolean
  title?: string
  width?: number | string
  placement?: 'left' | 'right' | 'top' | 'bottom'
  closable?: boolean
  closeOnEsc?: boolean
  maskClosable?: boolean
  showFooter?: boolean
  showCancel?: boolean
  showConfirm?: boolean
  cancelText?: string
  confirmText?: string
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  width: undefined,
  placement: 'right',
  closable: true,
  closeOnEsc: true,
  maskClosable: true,
  showFooter: false,
  showCancel: true,
  showConfirm: true,
  cancelText: '取消',
  confirmText: '确定',
  loading: false
})

const emit = defineEmits<{
  'update:show': [value: boolean]
  'confirm': []
  'cancel': []
}>()

const appStore = useAppStore()

// 响应式宽度：移动端全屏，桌面端固定宽度
const drawerWidth = computed(() => {
  if (props.width) return props.width
  return appStore.isMobile ? '100%' : 640
})

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
  emit('update:show', false)
}
</script>
