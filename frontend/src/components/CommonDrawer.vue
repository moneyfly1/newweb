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
    <n-drawer-content :title="title" :closable="closable" :body-content-style="bodyContentStyle" :footer-style="footerStyle" class="common-drawer-content">
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
  if (!appStore.isMobile) {
    return props.width ?? 640
  }
  if (props.placement === 'left' || props.placement === 'right') {
    return 'calc(100vw - 24px)'
  }
  return '100%'
})

const bodyContentStyle = computed(() => {
  if (!appStore.isMobile) {
    return {
      padding: '20px 24px',
      overflow: 'auto'
    }
  }
  return {
    padding: '16px 14px',
    overflow: 'auto'
  }
})

const footerStyle = computed(() => {
  if (!appStore.isMobile) {
    return {
      padding: '16px 24px'
    }
  }
  return {
    padding: '12px 14px 16px',
    justifyContent: 'stretch'
  }
})

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
  emit('update:show', false)
}
</script>

<style scoped>
.common-drawer-content :deep(.n-drawer-body-content-wrapper) {
  overflow: auto;
}

@media (max-width: 767px) {
  :deep(.n-drawer) {
    max-width: calc(100vw - 24px);
  }

  :deep(.n-drawer--right-placement),
  :deep(.n-drawer--left-placement) {
    top: 12px;
    bottom: 12px;
    height: auto !important;
    border-radius: 18px;
    overflow: hidden;
  }

  .common-drawer-content :deep(.n-drawer-header) {
    padding: 14px 14px 10px;
  }

  .common-drawer-content :deep(.n-drawer-header__main) {
    font-size: 18px;
    line-height: 1.4;
  }

  .common-drawer-content :deep(.n-drawer-footer) {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .common-drawer-content :deep(.n-drawer-footer .n-space) {
    width: 100%;
    flex-wrap: wrap !important;
    justify-content: stretch !important;
  }

  .common-drawer-content :deep(.n-drawer-footer .n-space > .n-button),
  .common-drawer-content :deep(.n-drawer-footer .n-space .n-button) {
    flex: 1 1 calc(50% - 5px);
    min-width: 0;
  }
}
</style>
