<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="appStore.themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <n-loading-bar-provider>
            <router-view />
          </n-loading-bar-provider>
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { zhCN, dateZhCN, darkTheme } from 'naive-ui'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const naiveTheme = computed(() => appStore.isDark ? darkTheme : null)

onMounted(() => {
  appStore.initTheme()
})

onUnmounted(() => {
  appStore.cleanup()
})
</script>
