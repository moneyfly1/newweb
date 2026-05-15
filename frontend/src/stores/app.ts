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
  indigo: {
    primary: '#4f46e5', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#6b7280',
    bg: '#ffffff', bgPage: '#f8fafc', text: '#0f172a', textSecondary: '#64748b', border: '#e2e8f0',
    sidebarBg: '#f1f5f9', sidebarText: '#334155', sidebarHover: '#e2e8f0', sidebarActive: '#4f46e5',
  },
  sky: {
    primary: '#0284c7', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#64748b',
    bg: '#ffffff', bgPage: '#f8fafc', text: '#0f172a', textSecondary: '#475569', border: '#e0f2fe',
    sidebarBg: '#f0f9ff', sidebarText: '#0c4a6e', sidebarHover: '#e0f2fe', sidebarActive: '#0284c7',
  },
  teal: {
    primary: '#0f766e', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#64748b',
    bg: '#ffffff', bgPage: '#f8fdfb', text: '#0f172a', textSecondary: '#475569', border: '#ccfbf1',
    sidebarBg: '#f0fdfa', sidebarText: '#134e4a', sidebarHover: '#ccfbf1', sidebarActive: '#0f766e',
  },
  mint: {
    primary: '#059669', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#64748b',
    bg: '#ffffff', bgPage: '#fafdfa', text: '#0f172a', textSecondary: '#475569', border: '#d1fae5',
    sidebarBg: '#f5fdf8', sidebarText: '#14532d', sidebarHover: '#d1fae5', sidebarActive: '#059669',
  },
  amber: {
    primary: '#b45309', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#78716c',
    bg: '#ffffff', bgPage: '#fefefe', text: '#1c1917', textSecondary: '#78716c', border: '#fef3c7',
    sidebarBg: '#fffbeb', sidebarText: '#78350f', sidebarHover: '#fef3c7', sidebarActive: '#b45309',
  },
  rose: {
    primary: '#be185d', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#6b7280',
    bg: '#ffffff', bgPage: '#fdf8fa', text: '#1a0f14', textSecondary: '#6b5a62', border: '#fce7f3',
    sidebarBg: '#fdf2f8', sidebarText: '#831843', sidebarHover: '#fce7f3', sidebarActive: '#be185d',
  },
  slate: {
    primary: '#374151', success: '#059669', warning: '#d97706', danger: '#dc2626', info: '#6b7280',
    bg: '#ffffff', bgPage: '#f9fafb', text: '#111827', textSecondary: '#6b7280', border: '#e5e7eb',
    sidebarBg: '#f3f4f6', sidebarText: '#374151', sidebarHover: '#e5e7eb', sidebarActive: '#374151',
  },
  midnight: {
    primary: '#a78bfa', success: '#34d399', warning: '#fbbf24', danger: '#f87171', info: '#9ca3af',
    bg: '#1a1a2e', bgPage: '#12121f', text: '#e2e8f0', textSecondary: '#94a3b8', border: '#2a2a40',
    sidebarBg: '#12121f', sidebarText: '#c4b5fd', sidebarHover: '#1e1e35', sidebarActive: '#a78bfa',
  },
}

const availableThemes: ThemeOption[] = [
  { value: 'indigo', label: '靛蓝', color: '#4f46e5' },
  { value: 'sky', label: '天空', color: '#0284c7' },
  { value: 'teal', label: '青碧', color: '#0f766e' },
  { value: 'mint', label: '薄荷', color: '#059669' },
  { value: 'amber', label: '琥珀', color: '#b45309' },
  { value: 'rose', label: '玫瑰', color: '#be185d' },
  { value: 'slate', label: '岩灰', color: '#374151' },
  { value: 'midnight', label: '午夜', color: '#a78bfa' },
  { value: 'auto', label: '跟随系统', color: '#6b7280' },
]

const darkThemes = new Set(['midnight'])

function resolveTheme(theme: string): string {
  if (theme === 'auto') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'midnight' : 'indigo'
  }
  return theme
}

function getThemeConfig(theme: string): ThemeConfig {
  const resolved = resolveTheme(theme)
  return themeConfigs[resolved] || themeConfigs.indigo
}

function withAlpha(color: string, alpha: number): string {
  const a = Math.round(Math.min(Math.max(alpha, 0), 1) * 255).toString(16).padStart(2, '0')
  if (/^#[0-9a-fA-F]{6}$/.test(color)) return `${color}${a}`
  if (/^#[0-9a-fA-F]{3}$/.test(color)) {
    const hex = color.slice(1).split('').map(c => c + c).join('')
    return `#${hex}${a}`
  }
  return color
}

function buildNaiveOverrides(config: ThemeConfig): GlobalThemeOverrides {
  const primarySoft = withAlpha(config.primary, 0.08)
  const primaryHover = withAlpha(config.primary, 0.12)
  const primaryActive = withAlpha(config.primary, 0.18)
  const primaryStrong = withAlpha(config.primary, 0.24)

  return {
    common: {
      primaryColor: config.primary,
      primaryColorHover: withAlpha(config.primary, 0.82),
      primaryColorPressed: withAlpha(config.primary, 0.7),
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
      placeholderColor: withAlpha(config.textSecondary, 0.78),
      placeholderColorDisabled: withAlpha(config.textSecondary, 0.52),
      borderColor: config.border,
      dividerColor: config.border,
      borderRadius: '8px',
      borderRadiusSmall: '6px',
      fontSizeSmall: '13px',
      fontSizeMedium: '14px',
      fontSizeLarge: '15px',
      fontWeightStrong: '600',
    },
    Menu: {
      color: config.sidebarBg,
      itemColorHover: primarySoft,
      itemColorActive: primaryActive,
      itemColorActiveHover: primaryStrong,
      itemColorActiveCollapsed: primaryActive,
      itemTextColor: config.sidebarText,
      itemTextColorHover: config.text,
      itemTextColorActive: config.text,
      itemTextColorActiveHover: config.text,
      itemTextColorChildActive: config.text,
      itemTextColorChildActiveHover: config.text,
      itemIconColor: config.textSecondary,
      itemIconColorHover: config.text,
      itemIconColorActive: config.text,
      itemIconColorActiveHover: config.text,
      itemIconColorChildActive: config.text,
      itemIconColorChildActiveHover: config.text,
      arrowColor: config.textSecondary,
      arrowColorHover: config.text,
      arrowColorActive: config.text,
      arrowColorActiveHover: config.text,
      arrowColorChildActive: config.text,
      arrowColorChildActiveHover: config.text,
      groupTextColor: config.textSecondary,
    },
    DataTable: {
      borderColor: config.border,
      thColor: primarySoft,
      thColorHover: primaryHover,
      thColorSorting: primaryHover,
      tdColor: config.bg,
      tdColorHover: primaryHover,
      tdColorSorting: primarySoft,
      tdTextColor: config.text,
      thTextColor: config.text,
      thIconColorActive: config.primary,
      borderRadius: '10px',
    },
    List: {
      textColor: config.text,
      color: config.bg,
      colorHover: primaryHover,
      borderColor: config.border,
      borderRadius: '10px',
    },
    Card: {
      color: config.bg,
      colorTarget: primarySoft,
      textColor: config.text,
      titleTextColor: config.text,
      borderColor: config.border,
      actionColor: primarySoft,
      borderRadius: '12px',
      paddingMedium: '20px',
    },
    Thing: {
      titleTextColor: config.text,
      textColor: config.textSecondary,
    },
    Tabs: {
      barColor: config.primary,
      tabTextColorLine: config.textSecondary,
      tabTextColorHoverLine: config.text,
      tabTextColorActiveLine: config.text,
      paneTextColor: config.text,
    },
    Tag: {
      borderRadius: '6px',
    },
    Button: {
      borderRadiusSmall: '6px',
      borderRadiusMedium: '8px',
      borderRadiusLarge: '10px',
    },
    Input: {
      borderRadius: '8px',
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
    '--primary-color-soft': withAlpha(config.primary, 0.08),
    '--primary-color-hover': withAlpha(config.primary, 0.12),
    '--primary-color-active': withAlpha(config.primary, 0.2),
    '--list-hover-color': withAlpha(config.primary, 0.12),
    '--list-active-color': withAlpha(config.primary, 0.2),
    '--sidebar-bg': config.sidebarBg,
    '--sidebar-text': config.sidebarText,
    '--sidebar-hover': config.sidebarHover,
    '--sidebar-active': config.sidebarActive,
    '--radius-sm': '6px',
    '--radius-md': '8px',
    '--radius-lg': '12px',
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
  const viewMode = ref<'table' | 'grid'>((localStorage.getItem('cboard-view-mode') as 'table' | 'grid') || 'table')

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

  function setViewMode(mode: 'table' | 'grid') {
    viewMode.value = mode
    localStorage.setItem('cboard-view-mode', mode)
  }

  function toggleViewMode() {
    setViewMode(viewMode.value === 'table' ? 'grid' : 'table')
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
    currentTheme, isDark, sidebarCollapsed, isMobile, mobileMenuOpen, viewMode,
    themeOverrides, availableThemes,
    setTheme, toggleTheme, toggleSidebar, toggleMobileMenu, closeMobileMenu,
    setViewMode, toggleViewMode,
    initTheme, initApp, cleanup, checkMobile,
  }
})
