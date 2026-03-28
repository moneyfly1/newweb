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
import { getSettings } from '@/api/admin'

const appStore = useAppStore()
const naiveTheme = computed(() => appStore.isDark ? darkTheme : null)

onMounted(async () => {
  appStore.initTheme()

  // 动态设置网站标题、描述和图标
  try {
    const res = await getSettings()
    if (res.code === 0 && res.data) {
      const siteName = res.data.site_name || 'CBoard'
      const siteDesc = res.data.site_description || '高效流畅的管理面板'
      const siteIcon = res.data.site_icon

      document.title = siteName

      const metaDesc = document.querySelector('meta[name="description"]')
      if (metaDesc) {
        metaDesc.setAttribute('content', siteDesc)
      }

      if (siteIcon) {
        const link = document.querySelector("link[rel*='icon']") as HTMLLinkElement
        if (link) {
          link.href = siteIcon
        }
      }
    }
  } catch {
    document.title = 'CBoard'
  }
})

onUnmounted(() => {
  appStore.cleanup()
})
</script>
