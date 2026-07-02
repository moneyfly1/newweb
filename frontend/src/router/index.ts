import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { prefetchCSRFToken } from '@/utils/request'

const viewLoaders = {
  Login: () => import('@/views/auth/Login.vue'),
  Register: () => import('@/views/auth/Register.vue'),
  ForgotPassword: () => import('@/views/auth/ForgotPassword.vue'),
  AdminLogin: () => import('@/views/admin/Login.vue'),
  UserLayout: () => import('@/layouts/UserLayout.vue'),
  Dashboard: () => import('@/views/dashboard/Index.vue'),
  Subscription: () => import('@/views/subscription/Index.vue'),
  Orders: () => import('@/views/order/Index.vue'),
  Shop: () => import('@/views/order/Shop.vue'),
  Tickets: () => import('@/views/ticket/Index.vue'),
  TicketDetail: () => import('@/views/ticket/Detail.vue'),
  Nodes: () => import('@/views/node/Index.vue'),
  Devices: () => import('@/views/device/Index.vue'),
  Invite: () => import('@/views/invite/Index.vue'),
  Settings: () => import('@/views/settings/Index.vue'),
  Help: () => import('@/views/help/Index.vue'),
  PaymentReturn: () => import('@/views/payment/Return.vue'),
  LoginHistory: () => import('@/views/history/Index.vue'),
  Recharge: () => import('@/views/recharge/Index.vue'),
  Redeem: () => import('@/views/redeem/Index.vue'),
  MysteryBox: () => import('@/views/mystery-box/Index.vue'),
  AdminLayout: () => import('@/layouts/AdminLayout.vue'),
  AdminDashboard: () => import('@/views/admin/Dashboard.vue'),
  AdminUsers: () => import('@/views/admin/users/Index.vue'),
  AdminAbnormalUsers: () => import('@/views/admin/abnormal-users/Index.vue'),
  AdminOrders: () => import('@/views/admin/orders/Index.vue'),
  AdminPackages: () => import('@/views/admin/packages/Index.vue'),
  AdminNodes: () => import('@/views/admin/nodes/Index.vue'),
  AdminCustomNodes: () => import('@/views/admin/custom-nodes/Index.vue'),
  AdminConfigUpdate: () => import('@/views/admin/config-update/Index.vue'),
  AdminSubscriptions: () => import('@/views/admin/subscriptions/Index.vue'),
  AdminCoupons: () => import('@/views/admin/coupons/Index.vue'),
  AdminTickets: () => import('@/views/admin/tickets/Index.vue'),
  AdminLevels: () => import('@/views/admin/levels/Index.vue'),
  AdminRedeem: () => import('@/views/admin/redeem/Index.vue'),
  AdminInvites: () => import('@/views/admin/invites/Index.vue'),
  AdminMysteryBox: () => import('@/views/admin/mystery-box/Index.vue'),
  AdminSettings: () => import('@/views/admin/settings/Index.vue'),
  AdminAnnouncements: () => import('@/views/admin/announcements/Index.vue'),
  AdminStats: () => import('@/views/admin/stats/Index.vue'),
  AdminLogs: () => import('@/views/admin/logs/Index.vue'),
  AdminEmailQueue: () => import('@/views/admin/email-queue/Index.vue'),
  NotFound: () => import('@/views/NotFound.vue'),
} satisfies Record<string, () => Promise<unknown>>

const preloadedViews = new Set<string>()

export function preloadRouteComponent(routeName: string) {
  const loader = viewLoaders[routeName as keyof typeof viewLoaders]
  if (!loader || preloadedViews.has(routeName)) return
  preloadedViews.add(routeName)
  loader().catch(() => preloadedViews.delete(routeName))
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: viewLoaders.Login,
    meta: { guest: true },
  },
  {
    path: '/register',
    name: 'Register',
    component: viewLoaders.Register,
    meta: { guest: true },
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: viewLoaders.ForgotPassword,
    meta: { guest: true },
  },
  {
    path: '/admin/login',
    name: 'AdminLogin',
    component: viewLoaders.AdminLogin,
    meta: { guest: true },
  },
  {
    path: '/',
    component: viewLoaders.UserLayout,
    meta: { auth: true },
    children: [
      { path: '', name: 'Dashboard', component: viewLoaders.Dashboard },
      { path: 'subscription', name: 'Subscription', component: viewLoaders.Subscription },
      { path: 'orders', name: 'Orders', component: viewLoaders.Orders },
      { path: 'shop', name: 'Shop', component: viewLoaders.Shop },
      { path: 'tickets', name: 'Tickets', component: viewLoaders.Tickets },
      { path: 'tickets/:id', name: 'TicketDetail', component: viewLoaders.TicketDetail },
      { path: 'nodes', name: 'Nodes', component: viewLoaders.Nodes },
      { path: 'devices', name: 'Devices', component: viewLoaders.Devices },
      { path: 'invite', name: 'Invite', component: viewLoaders.Invite },
      { path: 'settings', name: 'Settings', component: viewLoaders.Settings },
      { path: 'help', name: 'Help', component: viewLoaders.Help },
      { path: 'payment/return', name: 'PaymentReturn', component: viewLoaders.PaymentReturn },
      { path: 'login-history', name: 'LoginHistory', component: viewLoaders.LoginHistory },
      { path: 'recharge', name: 'Recharge', component: viewLoaders.Recharge },
      { path: 'redeem', name: 'Redeem', component: viewLoaders.Redeem },
      { path: 'mystery-box', name: 'MysteryBox', component: viewLoaders.MysteryBox },
    ],
  },
  {
    path: '/admin',
    component: viewLoaders.AdminLayout,
    meta: { auth: true, admin: true },
    children: [
      { path: '', name: 'AdminDashboard', component: viewLoaders.AdminDashboard },
      { path: 'users', name: 'AdminUsers', component: viewLoaders.AdminUsers },
      { path: 'abnormal-users', name: 'AdminAbnormalUsers', component: viewLoaders.AdminAbnormalUsers },
      { path: 'orders', name: 'AdminOrders', component: viewLoaders.AdminOrders },
      { path: 'packages', name: 'AdminPackages', component: viewLoaders.AdminPackages },
      { path: 'nodes', name: 'AdminNodes', component: viewLoaders.AdminNodes },
      { path: 'custom-nodes', name: 'AdminCustomNodes', component: viewLoaders.AdminCustomNodes },
      { path: 'config-update', name: 'AdminConfigUpdate', component: viewLoaders.AdminConfigUpdate },
      { path: 'subscriptions', name: 'AdminSubscriptions', component: viewLoaders.AdminSubscriptions },
      { path: 'coupons', name: 'AdminCoupons', component: viewLoaders.AdminCoupons },
      { path: 'tickets', name: 'AdminTickets', component: viewLoaders.AdminTickets },
      { path: 'levels', name: 'AdminLevels', component: viewLoaders.AdminLevels },
      { path: 'redeem', name: 'AdminRedeem', component: viewLoaders.AdminRedeem },
      { path: 'invites', name: 'AdminInvites', component: viewLoaders.AdminInvites },
      { path: 'mystery-box', name: 'AdminMysteryBox', component: viewLoaders.AdminMysteryBox },
      { path: 'settings', name: 'AdminSettings', component: viewLoaders.AdminSettings },
      { path: 'announcements', name: 'AdminAnnouncements', component: viewLoaders.AdminAnnouncements },
      { path: 'stats', name: 'AdminStats', component: viewLoaders.AdminStats },
      { path: 'logs', name: 'AdminLogs', component: viewLoaders.AdminLogs },
      { path: 'email-queue', name: 'AdminEmailQueue', component: viewLoaders.AdminEmailQueue },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: viewLoaders.NotFound,
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore()

  if (userStore.token && !userStore.userInfo) {
    try {
      prefetchCSRFToken()
      await userStore.fetchUser()
    } catch {
      userStore.logout(true)
      return next('/login')
    }
  }

  if (to.meta.auth && !userStore.isLoggedIn) {
    return next('/login')
  }
  if (to.meta.guest && userStore.isLoggedIn) {
    return next(userStore.isAdmin ? '/admin' : '/')
  }
  if (to.meta.admin && !userStore.isAdmin) {
    return next('/')
  }
  next()
})

export default router
