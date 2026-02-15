<template>
  <div style="height: 100%">
  <!-- Desktop Layout -->
  <n-layout has-sider style="height: 100%" v-if="!appStore.isMobile">
    <n-layout-sider bordered :collapsed="appStore.sidebarCollapsed" collapse-mode="width" :collapsed-width="64" :width="220" show-trigger @collapse="appStore.sidebarCollapsed = true" @expand="appStore.sidebarCollapsed = false">
      <div class="logo" @click="router.push('/')">
        <span v-if="!appStore.sidebarCollapsed">CBoard</span>
        <span v-else>C</span>
      </div>
      <n-menu :collapsed="appStore.sidebarCollapsed" :collapsed-width="64" :options="menuOptions" :value="activeKey" @update:value="handleMenuClick" />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered class="desktop-header">
        <div style="font-size: 16px; font-weight: 500;">{{ currentTitle }}</div>
        <n-space align="center">
          <n-badge :value="unreadCount" :max="99">
            <n-button quaternary circle @click="router.push('/tickets')">
              <template #icon><n-icon><notifications-outline /></n-icon></template>
            </n-button>
          </n-badge>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary>{{ userStore.userInfo?.username || '用户' }}</n-button>
          </n-dropdown>
        </n-space>
      </n-layout-header>
      <!-- Admin return banner -->
      <div v-if="isAdminViewing" class="admin-return-banner" @click="returnToAdmin">
        <n-icon :size="16"><shield-outline /></n-icon>
        <span>正在以用户身份浏览 · 点击返回管理后台</span>
      </div>
      <n-layout-content content-style="padding: 24px;" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
<!-- MOBILE_SECTION -->
  <!-- Mobile Layout -->
  <n-layout style="height: 100%" v-else>
    <n-layout-header bordered class="mobile-header">
      <span class="mobile-logo">CBoard</span>
      <div class="mobile-header-right">
        <n-badge :value="unreadCount" :max="99" :offset="[-4, 4]">
          <n-button quaternary circle size="small" @click="router.push('/tickets')">
            <template #icon><n-icon :size="20"><notifications-outline /></n-icon></template>
          </n-button>
        </n-badge>
        <n-button quaternary circle size="small" @click="showMobileMore = true">
          <template #icon><n-icon :size="20"><ellipsis-horizontal-outline /></n-icon></template>
        </n-button>
      </div>
    </n-layout-header>
    <!-- Admin return banner -->
    <div v-if="isAdminViewing" class="admin-return-banner" @click="returnToAdmin">
      <n-icon :size="16"><shield-outline /></n-icon>
      <span>正在以用户身份浏览 · 点击返回管理后台</span>
    </div>
    <n-layout-content content-style="padding: 12px 14px; padding-bottom: 72px;" :native-scrollbar="false">
      <router-view />
    </n-layout-content>
    <div class="mobile-tabbar">
      <div v-for="tab in mobileTabs" :key="tab.key" class="mobile-tab" :class="{ active: activeKey === tab.key }" @click="handleMenuClick(tab.key)">
        <n-icon :size="22" :component="tab.icon" />
        <span class="mobile-tab-label">{{ tab.label }}</span>
      </div>
    </div>
    <n-drawer v-model:show="showMobileMore" placement="bottom" :height="420" closable>
      <n-drawer-content title="更多">
        <div class="mobile-more-grid">
          <div v-for="item in moreMenuItems" :key="item.key" class="mobile-more-item" @click="handleMoreClick(item.key)">
            <div class="mobile-more-icon"><n-icon :size="24" :component="item.icon" /></div>
            <span class="mobile-more-label">{{ item.label }}</span>
          </div>
        </div>
        <div class="mobile-theme-section">
          <div class="mobile-theme-label">主题</div>
          <div class="mobile-theme-grid">
            <div
              v-for="t in appStore.availableThemes"
              :key="t.value"
              class="mobile-theme-dot"
              :class="{ active: appStore.currentTheme === t.value }"
              :style="{ background: t.color }"
              @click="appStore.setTheme(t.value)"
            >
              <span class="mobile-theme-dot-label">{{ t.label }}</span>
            </div>
          </div>
        </div>
      </n-drawer-content>
    </n-drawer>
  </n-layout>

  <!-- Theme Picker Drawer (shared) -->
  <n-drawer v-model:show="showThemeDrawer" placement="right" :width="280">
    <n-drawer-content title="选择主题">
      <div class="theme-picker-grid">
        <div
          v-for="t in appStore.availableThemes"
          :key="t.value"
          class="theme-picker-item"
          :class="{ active: appStore.currentTheme === t.value }"
          @click="appStore.setTheme(t.value)"
        >
          <div class="theme-picker-color" :style="{ background: t.color }"></div>
          <span class="theme-picker-label">{{ t.label }}</span>
        </div>
      </div>
    </n-drawer-content>
  </n-drawer>
  </div>
</template>
<!-- SCRIPT_SECTION -->
<script setup lang="ts">
import { computed, ref, h, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon } from 'naive-ui'
import {
  HomeOutline, CloudOutline, CartOutline, StorefrontOutline,
  ChatbubblesOutline, ServerOutline, PhonePortraitOutline, PeopleOutline,
  SettingsOutline, NotificationsOutline, EllipsisHorizontalOutline,
  LogOutOutline, ShieldOutline, KeyOutline,
  TimeOutline, WalletOutline, HelpCircleOutline, GiftOutline,
} from '@vicons/ionicons5'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { getUnreadCount } from '@/api/common'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const userStore = useUserStore()
const unreadCount = ref(0)
const showMobileMore = ref(false)

onMounted(async () => {
  try { const res: any = await getUnreadCount(); unreadCount.value = res.data?.count || 0 } catch {}
})

function renderIcon(icon: any) { return () => h(NIcon, null, { default: () => h(icon) }) }

const menuOptions = [
  { label: '仪表盘', key: 'Dashboard', icon: renderIcon(HomeOutline) },
  { label: '我的订阅', key: 'Subscription', icon: renderIcon(CloudOutline) },
  { label: '购买套餐', key: 'Shop', icon: renderIcon(StorefrontOutline) },
  { label: '我的订单', key: 'Orders', icon: renderIcon(CartOutline) },
  { label: '工单', key: 'Tickets', icon: renderIcon(ChatbubblesOutline) },
  { label: '节点状态', key: 'Nodes', icon: renderIcon(ServerOutline) },
  { label: '我的设备', key: 'Devices', icon: renderIcon(PhonePortraitOutline) },
  { label: '邀请返利', key: 'Invite', icon: renderIcon(PeopleOutline) },
  { label: '卡密兑换', key: 'Redeem', icon: renderIcon(KeyOutline) },
  { label: '盲盒', key: 'MysteryBox', icon: renderIcon(GiftOutline) },
  { label: '充值', key: 'Recharge', icon: renderIcon(WalletOutline) },
  { label: '登录历史', key: 'LoginHistory', icon: renderIcon(TimeOutline) },
  { label: '帮助/下载', key: 'Help', icon: renderIcon(HelpCircleOutline) },
]

// Mobile: 5 main tabs at bottom
const mobileTabs = [
  { label: '首页', key: 'Dashboard', icon: HomeOutline },
  { label: '订阅', key: 'Subscription', icon: CloudOutline },
  { label: '商店', key: 'Shop', icon: StorefrontOutline },
  { label: '订单', key: 'Orders', icon: CartOutline },
  { label: '我的', key: 'Settings', icon: SettingsOutline },
]

// Mobile: more menu items (the rest)
const moreMenuItems = [
  { label: '工单', key: 'Tickets', icon: ChatbubblesOutline },
  { label: '节点状态', key: 'Nodes', icon: ServerOutline },
  { label: '我的设备', key: 'Devices', icon: PhonePortraitOutline },
  { label: '邀请返利', key: 'Invite', icon: PeopleOutline },
  { label: '卡密兑换', key: 'Redeem', icon: KeyOutline },
  { label: '盲盒', key: 'MysteryBox', icon: GiftOutline },
  { label: '充值', key: 'Recharge', icon: WalletOutline },
  { label: '登录历史', key: 'LoginHistory', icon: TimeOutline },
  { label: '帮助/下载', key: 'Help', icon: HelpCircleOutline },
  ...(userStore.isAdmin ? [{ label: '管理后台', key: 'admin', icon: ShieldOutline }] : []),
  { label: '退出登录', key: 'logout', icon: LogOutOutline },
]

const activeKey = computed(() => route.name as string)
const currentTitle = computed(() => {
  const item = menuOptions.find(m => m.key === route.name)
  return item?.label || 'CBoard'
})

function handleMenuClick(key: string) { router.push({ name: key }) }

function handleMoreClick(key: string) {
  showMobileMore.value = false
  if (key === 'logout') { userStore.logout(); router.push('/login') }
  else if (key === 'admin') { router.push('/admin') }
  else { router.push({ name: key }) }
}

const showThemeDrawer = ref(false)

const userMenuOptions = computed(() => {
  const opts: any[] = [
    { label: '个人设置', key: 'settings' },
    { label: '切换主题', key: 'theme-picker' },
  ]
  if (userStore.isAdmin) opts.push({ label: '管理后台', key: 'admin' })
  opts.push({ type: 'divider', key: 'd1' }, { label: '退出登录', key: 'logout' })
  return opts
})

function handleUserMenu(key: string) {
  if (key === 'logout') { userStore.logout(); router.push('/login') }
  else if (key === 'admin') { router.push('/admin') }
  else if (key === 'theme-picker') { showThemeDrawer.value = true }
  else if (key === 'settings') { router.push({ name: 'Settings' }) }
}

const isAdminViewing = computed(() => !!localStorage.getItem('admin_token'))

function returnToAdmin() {
  const adminToken = localStorage.getItem('admin_token')
  const adminUser = localStorage.getItem('admin_user')
  if (adminToken) {
    localStorage.setItem('token', adminToken)
    if (adminUser) localStorage.setItem('user', adminUser)
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_user')
    // Reload to refresh user state
    window.location.href = '/admin'
  }
}
</script>
<!-- STYLE_SECTION -->
<style scoped>
.logo { height: 56px; display: flex; align-items: center; justify-content: center; font-size: 20px; font-weight: bold; cursor: pointer; border-bottom: 1px solid var(--n-border-color); }
.desktop-header { height: 56px; display: flex; align-items: center; justify-content: space-between; padding: 0 24px; }

/* Mobile Header */
.mobile-header { height: 48px; display: flex; align-items: center; justify-content: space-between; padding: 0 14px; }
.mobile-logo { font-size: 18px; font-weight: 700; }
.mobile-header-right { display: flex; align-items: center; gap: 4px; }

/* Mobile Tab Bar */
.mobile-tabbar {
  position: fixed; bottom: 0; left: 0; right: 0; z-index: 100;
  height: 56px; display: flex; align-items: center; justify-content: space-around;
  background: var(--n-color); border-top: 1px solid var(--n-border-color);
  padding-bottom: env(safe-area-inset-bottom);
}
.mobile-tab {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 2px; flex: 1; padding: 6px 0; cursor: pointer; color: #999; transition: color 0.2s;
}
.mobile-tab.active { color: #667eea; }
.mobile-tab-label { font-size: 10px; line-height: 1; }

/* Mobile More Menu */
.mobile-more-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px 12px; padding: 8px 0; }
.mobile-more-item { display: flex; flex-direction: column; align-items: center; gap: 8px; cursor: pointer; }
.mobile-more-icon { width: 48px; height: 48px; border-radius: 12px; display: flex; align-items: center; justify-content: center; background: var(--n-color-embedded, #f5f5f5); color: #667eea; }
.mobile-more-label { font-size: 12px; color: #666; }

/* Mobile Theme Section */
.mobile-theme-section { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--n-border-color, #e8e8e8); }
.mobile-theme-label { font-size: 14px; font-weight: 500; margin-bottom: 12px; color: var(--text-color, #333); }
.mobile-theme-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.mobile-theme-dot {
  width: 100%; aspect-ratio: 1; border-radius: 12px; cursor: pointer;
  display: flex; align-items: flex-end; justify-content: center; padding-bottom: 6px;
  border: 2px solid transparent; transition: all 0.2s;
}
.mobile-theme-dot.active { border-color: var(--primary-color, #667eea); box-shadow: 0 0 0 2px var(--primary-color, #667eea)33; }
.mobile-theme-dot-label { font-size: 10px; color: #fff; text-shadow: 0 1px 2px rgba(0,0,0,0.5); }

/* Desktop Theme Picker Drawer */
.theme-picker-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
.theme-picker-item {
  display: flex; align-items: center; gap: 10px; padding: 10px 12px;
  border-radius: 8px; cursor: pointer; border: 2px solid transparent;
  transition: all 0.2s; background: var(--n-color-embedded, #f5f5f5);
}
.theme-picker-item:hover { border-color: var(--primary-color, #667eea)66; }
.theme-picker-item.active { border-color: var(--primary-color, #667eea); background: var(--primary-color, #667eea)11; }
.theme-picker-color { width: 24px; height: 24px; border-radius: 50%; flex-shrink: 0; border: 1px solid rgba(0,0,0,0.1); }
.theme-picker-label { font-size: 13px; }

/* Admin Return Banner */
.admin-return-banner {
  display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 8px 16px; background: #ff9800; color: #fff; cursor: pointer;
  font-size: 13px; font-weight: 500;
}
.admin-return-banner:hover { background: #f57c00; }
</style>
