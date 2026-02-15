import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { GlobalThemeOverrides } from 'naive-ui'

export interface ThemeConfig {
  primary: string
  success: string
  warning: string
  danger: string
  info: string
  bg: string
  bgPage: string
  text: string
  textSecondary: string
  border: string
  sidebarBg: string
  sidebarText: string
  sidebarHover: string
  sidebarActive: string
}

export interface ThemeOption {
  value: string
  label: string
  color: string
}

const themeConfigs: Record<string, ThemeConfig> = {
  light: {
    primary: '#667eea', success: '#18a058', warning: '#f0a020', danger: '#e03050', info: '#909399',
    bg: '#ffffff', bgPage: '#f2f3f5', text: '#303133', textSecondary: '#606266', border: '#dcdfe6',
    sidebarBg: '#f8f9fa', sidebarText: '#303133', sidebarHover: '#e9ecef', sidebarActive: '#667eea',
  },
  dark: {
    primary: '#667eea', success: '#63e2b7', warning: '#f2c97d', danger: '#e88080', info: '#909399',
    bg: '#1a1a1a', bgPage: '#141414', text: '#E5EAF3', textSecondary: '#CFD3DC', border: '#4C4D4F',
    sidebarBg: '#1f1f1f', sidebarText: '#E5EAF3', sidebarHover: '#2a2a2a', sidebarActive: '#667eea',
  },
  blue: {
    primary: '#1890ff', success: '#52c41a', warning: '#faad14', danger: '#ff4d4f', info: '#8c8c8c',
    bg: '#f0f2f5', bgPage: '#e6f7ff', text: '#262626', textSecondary: '#595959', border: '#d9d9d9',
    sidebarBg: '#e6f7ff', sidebarText: '#262626', sidebarHover: '#bae7ff', sidebarActive: '#1890ff',
  },
  green: {
    primary: '#52c41a', success: '#52c41a', warning: '#faad14', danger: '#ff4d4f', info: '#8c8c8c',
    bg: '#f6ffed', bgPage: '#f0f9ff', text: '#262626', textSecondary: '#595959', border: '#b7eb8f',
    sidebarBg: '#f6ffed', sidebarText: '#262626', sidebarHover: '#d9f7be', sidebarActive: '#52c41a',
  },
  purple: {
    primary: '#722ed1', success: '#52c41a', warning: '#faad14', danger: '#ff4d4f', info: '#8c8c8c',
    bg: '#f9f0ff', bgPage: '#f0f0ff', text: '#262626', textSecondary: '#595959', border: '#d3adf7',
    sidebarBg: '#f9f0ff', sidebarText: '#262626', sidebarHover: '#efdbff', sidebarActive: '#722ed1',
  },
  orange: {
    primary: '#fa8c16', success: '#52c41a', warning: '#faad14', danger: '#ff4d4f', info: '#8c8c8c',
    bg: '#fff7e6', bgPage: '#fffbe6', text: '#262626', textSecondary: '#595959', border: '#ffd591',
    sidebarBg: '#fff7e6', sidebarText: '#262626', sidebarHover: '#ffe7ba', sidebarActive: '#fa8c16',
  },
  red: {
    primary: '#f5222d', success: '#52c41a', warning: '#faad14', danger: '#ff4d4f', info: '#8c8c8c',
    bg: '#fff1f0', bgPage: '#fff0f0', text: '#262626', textSecondary: '#595959', border: '#ffccc7',
    sidebarBg: '#fff1f0', sidebarText: '#262626', sidebarHover: '#ffd4d0', sidebarActive: '#f5222d',
  },
  cyan: {
    primary: '#13c2c2', success: '#52c41a', warning: '#faad14', danger: '#ff4d4f', info: '#8c8c8c',
    bg: '#e6fffb', bgPage: '#e0f7ff', text: '#262626', textSecondary: '#595959', border: '#87e8de',
    sidebarBg: '#e6fffb', sidebarText: '#262626', sidebarHover: '#b5f5ec', sidebarActive: '#13c2c2',
  },
  luck: {
    primary: '#FFD700', success: '#32CD32', warning: '#FFA500', danger: '#FF6347', info: '#9370DB',
    bg: '#FFFEF0', bgPage: '#FFFACD', text: '#2C2416', textSecondary: '#5C4A3A', border: '#FFD700',
    sidebarBg: '#FFFEF0', sidebarText: '#2C2416', sidebarHover: '#FFF8DC', sidebarActive: '#FFD700',
  },
  aurora: {
    primary: '#7B68EE', success: '#00CED1', warning: '#FF69B4', danger: '#FF1493', info: '#9370DB',
    bg: '#0F0C1D', bgPage: '#1A1625', text: '#E6E6FA', textSecondary: '#D8BFD8', border: '#4B0082',
    sidebarBg: '#1A1625', sidebarText: '#E6E6FA', sidebarHover: '#2A1F3D', sidebarActive: '#7B68EE',
  },
}

const availableThemes: ThemeOption[] = [
  { value: 'light', label: '浅色', color: '#667eea' },
  { value: 'dark', label: '深色', color: '#1a1a1a' },
  { value: 'blue', label: '蓝色', color: '#1890ff' },
  { value: 'green', label: '绿色', color: '#52c41a' },
  { value: 'purple', label: '紫色', color: '#722ed1' },
  { value: 'orange', label: '橙色', color: '#fa8c16' },
  { value: 'red', label: '红色', color: '#f5222d' },
  { value: 'cyan', label: '青色', color: '#13c2c2' },
  { value: 'luck', label: 'Luck', color: '#FFD700' },
  { value: 'aurora', label: 'Aurora', color: '#7B68EE' },
  { value: 'auto', label: '跟随系统', color: '#909399' },
]

const darkThemes = new Set(['dark', 'aurora'])

function resolveTheme(theme: string): string {
  if (theme === 'auto') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return theme
}

function getThemeConfig(theme: string): ThemeConfig {
  const resolved = resolveTheme(theme)
  return themeConfigs[resolved] || themeConfigs.light
}

function buildNaiveOverrides(config: ThemeConfig): GlobalThemeOverrides {
  return {
    common: {
      primaryColor: config.primary,
      primaryColorHover: config.primary + 'cc',
      primaryColorPressed: config.primary + 'aa',
      primaryColorSuppl: config.primary,
      successColor: config.success,
      warningColor: config.warning,
      errorColor: config.danger,
      infoColor: config.info,
      bodyColor: config.bg,
      cardColor: config.bg,
      modalColor: config.bg,
      popoverColor: config.bg,
      tableColor: config.bg,
      inputColor: config.bg,
      baseColor: config.bg,
      textColorBase: config.text,
      textColor1: config.text,
      textColor2: config.textSecondary,
      textColor3: config.textSecondary,
      borderColor: config.border,
      dividerColor: config.border,
    },
  }
}

function applyCSSVariables(config: ThemeConfig) {
  const root = document.documentElement
  const vars: Record<string, string> = {
    '--primary-color': config.primary,
    '--success-color': config.success,
    '--warning-color': config.warning,
    '--danger-color': config.danger,
    '--info-color': config.info,
    '--bg-color': config.bg,
    '--bg-page-color': config.bgPage,
    '--text-color': config.text,
    '--text-color-secondary': config.textSecondary,
    '--border-color': config.border,
    '--sidebar-bg': config.sidebarBg,
    '--sidebar-text': config.sidebarText,
    '--sidebar-hover': config.sidebarHover,
    '--sidebar-active': config.sidebarActive,
  }
  for (const [k, v] of Object.entries(vars)) {
    root.style.setProperty(k, v)
  }
  document.body.style.backgroundColor = config.bgPage
  document.body.style.color = config.text
}

export const useAppStore = defineStore('app', () => {
  const currentTheme = ref(localStorage.getItem('cboard-theme') || 'light')
  const sidebarCollapsed = ref(false)
  const isMobile = ref(false)
  const mobileMenuOpen = ref(false)

  const isDark = computed(() => {
    const resolved = resolveTheme(currentTheme.value)
    return darkThemes.has(resolved)
  })

  const themeOverrides = computed<GlobalThemeOverrides>(() => {
    const config = getThemeConfig(currentTheme.value)
    return buildNaiveOverrides(config)
  })

  function setTheme(theme: string) {
    currentTheme.value = theme
    localStorage.setItem('cboard-theme', theme)
    const config = getThemeConfig(theme)
    applyCSSVariables(config)
  }

  function toggleTheme() {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  function checkMobile() {
    isMobile.value = window.innerWidth < 768
    if (isMobile.value) sidebarCollapsed.value = true
  }

  let resizeHandler: (() => void) | null = null
  let mediaHandler: ((e: MediaQueryListEvent) => void) | null = null
  let resizeTimer: ReturnType<typeof setTimeout> | null = null

  function initApp() {
    applyCSSVariables(getThemeConfig(currentTheme.value))
    checkMobile()
    resizeHandler = () => {
      if (resizeTimer) clearTimeout(resizeTimer)
      resizeTimer = setTimeout(() => checkMobile(), 150)
    }
    window.addEventListener('resize', resizeHandler)
    // Listen for system theme changes when in auto mode
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mediaHandler = () => {
      if (currentTheme.value === 'auto') {
        applyCSSVariables(getThemeConfig('auto'))
      }
    }
    mq.addEventListener('change', mediaHandler)
  }

  function cleanup() {
    if (resizeTimer) clearTimeout(resizeTimer)
    if (resizeHandler) window.removeEventListener('resize', resizeHandler)
    if (mediaHandler) {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', mediaHandler)
    }
  }

  function initTheme() { initApp() }
  function toggleSidebar() { sidebarCollapsed.value = !sidebarCollapsed.value }
  function toggleMobileMenu() { mobileMenuOpen.value = !mobileMenuOpen.value }
  function closeMobileMenu() { mobileMenuOpen.value = false }

  return {
    currentTheme, isDark, sidebarCollapsed, isMobile, mobileMenuOpen,
    themeOverrides, availableThemes,
    setTheme, toggleTheme, toggleSidebar, toggleMobileMenu, closeMobileMenu,
    initTheme, initApp, cleanup, checkMobile,
  }
})
