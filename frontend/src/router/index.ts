import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/Login.vue'),
    meta: { guest: true },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/auth/Register.vue'),
    meta: { guest: true },
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/auth/ForgotPassword.vue'),
    meta: { guest: true },
  },
  {
    path: '/admin/login',
    name: 'AdminLogin',
    component: () => import('@/views/admin/Login.vue'),
    meta: { guest: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/UserLayout.vue'),
    meta: { auth: true },
    children: [
      { path: '', name: 'Dashboard', component: () => import('@/views/dashboard/Index.vue') },
      { path: 'subscription', name: 'Subscription', component: () => import('@/views/subscription/Index.vue') },
      { path: 'orders', name: 'Orders', component: () => import('@/views/order/Index.vue') },
      { path: 'shop', name: 'Shop', component: () => import('@/views/order/Shop.vue') },
      { path: 'tickets', name: 'Tickets', component: () => import('@/views/ticket/Index.vue') },
      { path: 'tickets/:id', name: 'TicketDetail', component: () => import('@/views/ticket/Detail.vue') },
      { path: 'nodes', name: 'Nodes', component: () => import('@/views/node/Index.vue') },
      { path: 'devices', name: 'Devices', component: () => import('@/views/device/Index.vue') },
      { path: 'invite', name: 'Invite', component: () => import('@/views/invite/Index.vue') },
      { path: 'settings', name: 'Settings', component: () => import('@/views/settings/Index.vue') },
      { path: 'help', name: 'Help', component: () => import('@/views/help/Index.vue') },
      { path: 'payment/return', name: 'PaymentReturn', component: () => import('@/views/payment/Return.vue') },
      { path: 'login-history', name: 'LoginHistory', component: () => import('@/views/history/Index.vue') },
      { path: 'recharge', name: 'Recharge', component: () => import('@/views/recharge/Index.vue') },
      { path: 'redeem', name: 'Redeem', component: () => import('@/views/redeem/Index.vue') },
    ],
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    meta: { auth: true, admin: true },
    children: [
      { path: '', name: 'AdminDashboard', component: () => import('@/views/admin/Dashboard.vue') },
      { path: 'users', name: 'AdminUsers', component: () => import('@/views/admin/users/Index.vue') },
      { path: 'abnormal-users', name: 'AdminAbnormalUsers', component: () => import('@/views/admin/abnormal-users/Index.vue') },
      { path: 'orders', name: 'AdminOrders', component: () => import('@/views/admin/orders/Index.vue') },
      { path: 'packages', name: 'AdminPackages', component: () => import('@/views/admin/packages/Index.vue') },
      { path: 'nodes', name: 'AdminNodes', component: () => import('@/views/admin/nodes/Index.vue') },
      { path: 'custom-nodes', name: 'AdminCustomNodes', component: () => import('@/views/admin/custom-nodes/Index.vue') },
      { path: 'config-update', name: 'AdminConfigUpdate', component: () => import('@/views/admin/config-update/Index.vue') },
      { path: 'subscriptions', name: 'AdminSubscriptions', component: () => import('@/views/admin/subscriptions/Index.vue') },
      { path: 'coupons', name: 'AdminCoupons', component: () => import('@/views/admin/coupons/Index.vue') },
      { path: 'tickets', name: 'AdminTickets', component: () => import('@/views/admin/tickets/Index.vue') },
      { path: 'levels', name: 'AdminLevels', component: () => import('@/views/admin/levels/Index.vue') },
      { path: 'redeem', name: 'AdminRedeem', component: () => import('@/views/admin/redeem/Index.vue') },
      { path: 'settings', name: 'AdminSettings', component: () => import('@/views/admin/settings/Index.vue') },
      { path: 'announcements', name: 'AdminAnnouncements', component: () => import('@/views/admin/announcements/Index.vue') },
      { path: 'stats', name: 'AdminStats', component: () => import('@/views/admin/stats/Index.vue') },
      { path: 'logs', name: 'AdminLogs', component: () => import('@/views/admin/logs/Index.vue') },
      { path: 'email-queue', name: 'AdminEmailQueue', component: () => import('@/views/admin/email-queue/Index.vue') },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore()

  if (userStore.token && !userStore.userInfo) {
    try {
      await userStore.fetchUser()
    } catch {
      // Token invalid/expired — clear local state only (no API call)
      // The 401 interceptor already handles redirect
      userStore.logout()
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