<template>
  <div style="height: 100%">
  <!-- Desktop Layout -->
  <n-layout has-sider style="height: 100%" v-if="!appStore.isMobile">
    <n-layout-sider bordered :collapsed="appStore.sidebarCollapsed" collapse-mode="width" :collapsed-width="64" :width="220" show-trigger @collapse="appStore.sidebarCollapsed = true" @expand="appStore.sidebarCollapsed = false">
      <div class="logo" @click="router.push('/admin')">
        <span v-if="!appStore.sidebarCollapsed">CBoard Admin</span>
        <span v-else>A</span>
      </div>
      <n-menu :collapsed="appStore.sidebarCollapsed" :collapsed-width="64" :options="menuOptions" :value="activeKey" :default-expanded-keys="expandedKeys" @update:value="handleMenuClick" />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered class="desktop-header">
        <div style="font-size: 16px; font-weight: 500;">管理后台</div>
        <n-space align="center">
          <n-button quaternary size="small" @click="router.push('/')">返回前台</n-button>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary>{{ userStore.userInfo?.username || '管理员' }}</n-button>
          </n-dropdown>
        </n-space>
      </n-layout-header>
      <n-layout-content content-style="padding: 24px;" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
  <!-- Mobile Layout -->
  <n-layout style="height: 100%" v-else>
    <n-layout-header bordered class="mobile-header">
      <n-button quaternary circle size="small" @click="showDrawer = true">
        <template #icon><n-icon :size="22"><menu-outline /></n-icon></template>
      </n-button>
      <span class="mobile-title">管理后台</span>
      <n-button quaternary size="tiny" @click="router.push('/')">前台</n-button>
    </n-layout-header>
    <n-layout-content content-style="padding: 12px 14px;" :native-scrollbar="false">
      <router-view />
    </n-layout-content>
    <n-drawer v-model:show="showDrawer" placement="left" :width="260" closable>
      <n-drawer-content title="导航菜单" :native-scrollbar="false">
        <n-menu :options="menuOptions" :value="activeKey" :default-expanded-keys="expandedKeys" @update:value="handleMobileMenuClick" />
      </n-drawer-content>
    </n-drawer>
  </n-layout>

  <!-- Theme Picker Drawer -->
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
<script setup lang="ts">
import { computed, ref, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon } from 'naive-ui'
import {
  GridOutline, PeopleOutline, CartOutline, CubeOutline, ServerOutline,
  CloudOutline, PricetagOutline, ChatbubblesOutline, RibbonOutline,
  KeyOutline, SettingsOutline, MegaphoneOutline, StatsChartOutline,
  DocumentTextOutline, WarningOutline, GitNetworkOutline, MailOutline,
  RefreshOutline, MenuOutline,
} from '@vicons/ionicons5'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const userStore = useUserStore()
const showDrawer = ref(false)

function renderIcon(icon: any) { return () => h(NIcon, null, { default: () => h(icon) }) }

const menuOptions = [
  { label: '概览', key: 'group-overview', icon: renderIcon(GridOutline), children: [{ label: '仪表盘', key: 'AdminDashboard' }] },
  { label: '用户管理', key: 'group-users', icon: renderIcon(PeopleOutline), children: [
    { label: '用户列表', key: 'AdminUsers' }, { label: '异常用户', key: 'AdminAbnormalUsers' }, { label: '订阅管理', key: 'AdminSubscriptions' },
  ]},
  { label: '节点管理', key: 'group-nodes', icon: renderIcon(ServerOutline), children: [
    { label: '节点管理', key: 'AdminNodes' }, { label: '专线节点', key: 'AdminCustomNodes' }, { label: '节点更新', key: 'AdminConfigUpdate' },
  ]},
  { label: '订单管理', key: 'group-orders', icon: renderIcon(CartOutline), children: [
    { label: '订单列表', key: 'AdminOrders' }, { label: '套餐管理', key: 'AdminPackages' },
  ]},
  { label: '系统管理', key: 'group-system', icon: renderIcon(SettingsOutline), children: [
    { label: '系统设置', key: 'AdminSettings' }, { label: '公告管理', key: 'AdminAnnouncements' },
    { label: '优惠券', key: 'AdminCoupons' }, { label: '卡密管理', key: 'AdminRedeem' }, { label: '用户等级', key: 'AdminLevels' },
  ]},
  { label: '日志与分析', key: 'group-logs', icon: renderIcon(StatsChartOutline), children: [
    { label: '数据统计', key: 'AdminStats' }, { label: '系统日志', key: 'AdminLogs' }, { label: '邮件队列', key: 'AdminEmailQueue' },
  ]},
  { label: '工单管理', key: 'group-tickets', icon: renderIcon(ChatbubblesOutline), children: [{ label: '工单管理', key: 'AdminTickets' }] },
]

const routeToGroup: Record<string, string> = {}
for (const group of menuOptions) {
  if (group.children) { for (const child of group.children) { routeToGroup[child.key] = group.key } }
}

const activeKey = computed(() => route.name as string)
const expandedKeys = computed(() => { const g = routeToGroup[route.name as string]; return g ? [g] : [] })

function handleMenuClick(key: string) { router.push({ name: key }) }
function handleMobileMenuClick(key: string) { showDrawer.value = false; router.push({ name: key }) }

const showThemeDrawer = ref(false)

const userMenuOptions = [
  { label: '切换主题', key: 'theme-picker' },
  { type: 'divider', key: 'd1' },
  { label: '退出登录', key: 'logout' },
]

function handleUserMenu(key: string) {
  if (key === 'logout') { userStore.logout(); router.push('/login') }
  else if (key === 'theme-picker') { showThemeDrawer.value = true }
}
</script>
<style scoped>
.logo { height: 56px; display: flex; align-items: center; justify-content: center; font-size: 18px; font-weight: bold; cursor: pointer; border-bottom: 1px solid var(--n-border-color); }
.desktop-header { height: 56px; display: flex; align-items: center; justify-content: space-between; padding: 0 24px; }
.mobile-header { height: 48px; display: flex; align-items: center; justify-content: space-between; padding: 0 12px; }
.mobile-title { font-size: 16px; font-weight: 600; }

/* Theme Picker */
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
</style>
