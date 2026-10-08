/**
 * 移动端体检脚本（Pixel 5 视口 393×851，触摸模拟）
 *
 * 目的：把「手机端看起来不像 App」这种主观问题变成可复现的量化指标，
 * 每次改完页面跑一遍，防止改一个页面又把别的页面弄坏。
 *
 * 检查项：
 *  1. 横向溢出：整页 / 卡片 / 表格是否被撑出横向滚动
 *  2. 越界元素：右边界超出视口的元素（手机端最常见的「内容被切掉」）
 *  3. 桌面表格硬塞：手机宽度下仍在渲染 .n-data-table（应改为卡片列表）
 *  4. 触控目标：可点元素小于 40×40（App 里手指点不准）
 *  5. 全选缺失：页面有可批量操作的列表，却没有任何「全选」入口
 *  6. 输入框默认痕迹：聚焦输入框后是否冒出细线（浏览器下划线/描边）
 *  7. 字号过小：正文小于 12px
 *  8. 控制台报错
 *
 * 用法：
 *   node scripts/mobile-audit.mjs                     # 全部页面
 *   node scripts/mobile-audit.mjs /admin/users /nodes # 指定页面
 *   BASE_URL=http://localhost:3000 node scripts/mobile-audit.mjs
 *   AUDIT_USER=admin@example.com AUDIT_PASS=xxx node scripts/mobile-audit.mjs
 */
import { chromium, devices } from '/opt/homebrew/lib/node_modules/playwright/index.mjs'
import { writeFileSync, mkdirSync, existsSync } from 'node:fs'

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000'
const USER = process.env.AUDIT_USER || 'admin@example.com'
const PASS = process.env.AUDIT_PASS || 'LocalTest123'
const OUT_DIR = process.env.AUDIT_OUT || '/tmp/mobile-audit'

// 用户端 + 后台全部页面（深链接类页面用列表页代替）
const ROUTES = [
  '/', '/subscription', '/orders', '/shop', '/tickets', '/nodes', '/devices', '/invite',
  '/settings', '/help', '/terms', '/privacy', '/login-history', '/notifications', '/coupons',
  '/recharge', '/redeem', '/mystery-box',
  '/admin', '/admin/users', '/admin/abnormal-users', '/admin/orders', '/admin/packages',
  '/admin/nodes', '/admin/custom-nodes', '/admin/config-update', '/admin/subscriptions',
  '/admin/coupons', '/admin/tickets', '/admin/levels', '/admin/redeem', '/admin/invites',
  '/admin/mystery-box', '/admin/settings', '/admin/announcements', '/admin/stats',
  '/admin/logs', '/admin/email-queue',
]

const targets = process.argv.slice(2).length ? process.argv.slice(2) : ROUTES

/** 页面内执行的体检函数（交给浏览器跑，避免大量来回通信） */
function auditInPage() {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const out = { viewport: { w: vw, h: vh }, issues: [], stats: {} }

  const visible = (el) => {
    const r = el.getBoundingClientRect()
    const s = getComputedStyle(el)
    if (r.width < 1 || r.height < 1) return false
    if (s.visibility === 'hidden' || s.display === 'none' || Number(s.opacity) === 0) return false
    if (r.bottom < -50 || r.top > document.documentElement.scrollHeight + 50) return false
    // 屏幕上方之外（如 a11y 跳过链接停在 top:-100%）不算「可见」
    if (r.bottom < 0 || r.top < -20) return false
    return true
  }
  const describe = (el) => {
    const cls = (el.getAttribute('class') || '').split(/\s+/).filter(Boolean).slice(0, 3).join('.')
    const tag = el.tagName.toLowerCase()
    const text = (el.textContent || '').trim().slice(0, 24)
    return `${tag}${cls ? '.' + cls : ''}${text ? ` "${text}"` : ''}`
  }

  // 1) 横向溢出
  const de = document.documentElement
  out.stats.docOverflow = de.scrollWidth - de.clientWidth
  if (out.stats.docOverflow > 1) {
    out.issues.push({ kind: 'doc-overflow-x', detail: `整页横向溢出 ${out.stats.docOverflow}px` })
  }

  // 2) 越界元素（右边界超出视口 2px 以上）
  const overflowing = []
  for (const el of document.querySelectorAll('body *')) {
    if (!visible(el)) continue
    const r = el.getBoundingClientRect()
    if (r.right > vw + 2 && r.width > 24) {
      // 自身或祖先带横向滚动/裁剪（naive tabs、横向 chip 条等）都是有意设计，不算问题
      let clipped = false
      let node = el
      for (let i = 0; i < 6 && node; i++) {
        const cs = getComputedStyle(node)
        if (['auto', 'scroll', 'hidden', 'clip'].includes(cs.overflowX)) { clipped = true; break }
        node = node.parentElement
      }
      overflowing.push({ el: describe(el), right: Math.round(r.right), width: Math.round(r.width), scrollable: clipped })
    }
  }
  const realOverflow = overflowing.filter(x => !x.scrollable)
  out.stats.overflowCount = realOverflow.length
  if (realOverflow.length) {
    out.issues.push({
      kind: 'element-overflow-x',
      detail: `${realOverflow.length} 个元素超出屏幕右侧`,
      samples: realOverflow.slice(0, 6),
    })
  }

  // 3) 桌面表格是否还在手机上硬撑
  const tables = [...document.querySelectorAll('.n-data-table')].filter(visible)
  const wideTables = tables.filter(t => t.getBoundingClientRect().width > vw * 0.98)
  out.stats.tables = tables.length
  out.stats.wideTables = wideTables.length
  if (wideTables.length) {
    out.issues.push({ kind: 'desktop-table-on-mobile', detail: `手机宽度下仍有 ${wideTables.length} 个桌面表格` })
  }

  // 4) 触控目标过小
  const small = []
  for (const el of document.querySelectorAll('button, a, .n-button, .n-checkbox, .n-switch, [role="button"], .mobile-tab')) {
    if (!visible(el)) continue
    const r = el.getBoundingClientRect()
    const s = getComputedStyle(el)
    if (s.pointerEvents === 'none') continue
    if (r.width < 40 || r.height < 40) {
      small.push({ el: describe(el), w: Math.round(r.width), h: Math.round(r.height) })
    }
  }
  out.stats.smallTargets = small.length
  if (small.length) {
    out.issues.push({ kind: 'small-tap-target', detail: `${small.length} 个可点元素小于 40×40`, samples: small.slice(0, 6) })
  }

  // 5) 全选：页面有 checkbox 列表或批量按钮，却没有「全选」入口
  const bodyText = document.body.innerText || ''
  const hasSelectAllWord = /全选/.test(bodyText)
  // 只统计「列表里的」复选框：表格行内 / 移动端卡片列表内
  const listCheckboxes = [...document.querySelectorAll('.n-data-table .n-checkbox, .mobile-card-list .n-checkbox, .app-list .n-checkbox')].filter(visible)
  const checkboxes = listCheckboxes
  const hasBatchButton = [...document.querySelectorAll('button')].some(b => /批量/.test(b.textContent || ''))
  out.stats.checkboxes = checkboxes.length
  out.stats.hasSelectAll = hasSelectAllWord
  out.stats.hasBatchButton = hasBatchButton
  if ((checkboxes.length >= 2 || hasBatchButton) && !hasSelectAllWord) {
    out.issues.push({
      kind: 'missing-select-all',
      detail: `有 ${checkboxes.length} 个复选框${hasBatchButton ? '和批量按钮' : ''}，但页面没有「全选」入口`,
    })
  }

  // 5.5) 内容宽度利用率：用户反馈「内容居中、边上很多空白」
  //      取页面里最宽的主要内容块，与视口宽度比；低于 88% 说明左右白边太多
  const host = document.querySelector('.mobile-admin-content, .user-mobile-content') || document.body
  let widest = 0
  let widestEl = null
  for (const el of host.querySelectorAll('*')) {
    const r = el.getBoundingClientRect()
    if (r.width < 120 || r.height < 30) continue
    const s2 = getComputedStyle(el)
    if (s2.position === 'fixed') continue
    // 只统计「有内容感」的块：卡片、列表、表格容器
    const cls = (el.getAttribute('class') || '')
    if (!/card|list|table|panel|section|shell|container/.test(cls)) continue
    if (r.width > widest) { widest = r.width; widestEl = describe(el) }
  }
  out.stats.contentWidth = Math.round(widest)
  out.stats.widthRatio = +((widest / vw) * 100).toFixed(1)
  if (widest > 0 && widest / vw < 0.88) {
    out.issues.push({
      kind: 'narrow-content',
      detail: `内容只占屏宽 ${out.stats.widthRatio}%（${Math.round(widest)}/${vw}px），左右白边过多`,
      samples: [{ el: widestEl }],
    })
  }

  // 5.6) 卡片还带渐变底：旧版「网页味」样式（App 卡片应是纯色）
  const gradientCards = []
  for (const el of document.querySelectorAll('.mobile-card, .n-card, .app-list-item')) {
    if (!visible(el)) continue
    const bg = getComputedStyle(el).backgroundImage
    if (bg && bg !== 'none' && /gradient/.test(bg)) gradientCards.push({ el: describe(el) })
  }
  out.stats.gradientCards = gradientCards.length
  if (gradientCards.length) {
    out.issues.push({ kind: 'gradient-card', detail: `${gradientCards.length} 个卡片还在用渐变底（旧网页味样式）`, samples: gradientCards.slice(0, 4) })
  }

  // 5.7) 顶部空白过大：首屏内容离顶部太远
  let firstTop = Infinity
  for (const el of host.querySelectorAll('*')) {
    const r = el.getBoundingClientRect()
    const cls = (el.getAttribute('class') || '')
    if (r.height < 32 || r.width < 120) continue
    if (!/card|list|table|panel|section|shell|container|header/.test(cls)) continue
    if (r.top < firstTop) firstTop = r.top
  }
  out.stats.firstContentTop = Number.isFinite(firstTop) ? Math.round(firstTop) : null
  if (Number.isFinite(firstTop) && firstTop > 200) {
    out.issues.push({ kind: 'top-gap', detail: `首屏内容从 ${Math.round(firstTop)}px 才开始，顶部空白过多` })
  }

  // 5.8) 「半截颜色」：只在元素左侧露出一条色（border-left / inset 左阴影 / 细窄伪元素）
  //      用户反馈「点列表时列表前面出现半截颜色」「很多按钮左侧有半截不一样的颜色」。
  //      App 里选中/强调应该是整块变色，所以这类左侧细条一律算问题。
  const stripes = []
  for (const el of document.querySelectorAll('body *')) {
    if (!visible(el)) continue
    const r = el.getBoundingClientRect()
    if (r.width < 30 || r.height < 12) continue
    const s2 = getComputedStyle(el)
    const blw = parseFloat(s2.borderLeftWidth) || 0
    const blc = s2.borderLeftColor
    const transparentLeft = !blc || blc === 'rgba(0, 0, 0, 0)' || blc === 'transparent'
    if (blw >= 2 && blw <= 8 && !transparentLeft && parseFloat(s2.borderTopWidth) < blw) {
      stripes.push({ el: describe(el), why: `border-left ${blw}px ${blc}` })
      continue
    }
    if (/inset\s+[2-9]px\s+0(px)?\s+0/.test(s2.boxShadow)) {
      stripes.push({ el: describe(el), why: `inset 左阴影 ${s2.boxShadow.slice(0, 40)}` })
    }
  }
  out.stats.leftStripes = stripes.length
  if (stripes.length) {
    out.issues.push({ kind: 'left-accent-stripe', detail: `${stripes.length} 处左侧「半截颜色」细条（应改成整块变色）`, samples: stripes.slice(0, 6) })
  }

  // 6) 字号过小
  const tiny = []
  for (const el of document.querySelectorAll('body *')) {
    if (!visible(el)) continue
    if (el.children.length) continue
    const txt = (el.textContent || '').trim()
    if (txt.length < 2) continue
    const fs = parseFloat(getComputedStyle(el).fontSize)
    if (fs && fs < 12) tiny.push({ el: describe(el), fs })
  }
  out.stats.tinyText = tiny.length
  if (tiny.length) out.issues.push({ kind: 'tiny-text', detail: `${tiny.length} 处正文小于 12px`, samples: tiny.slice(0, 6) })

  return out
}

/** 全局顺滑度检查：页面转场、按压反馈、底部安全区、吸顶顶栏是否都在 */
function smoothnessProbe() {
  const cssText = []
  for (const ss of document.styleSheets) {
    try { for (const r of ss.cssRules) cssText.push(r.cssText) } catch (e) { /* 跨域表忽略 */ }
  }
  const all = cssText.join('\n')
  const out = {
    pageTransition: /\.page-slide-enter-active/.test(all) && /transition/.test(all),
    pressFeedback: (() => {
      // 有 :active 的规则里必须包含 transform/scale 之类的按压反馈
      const activeRules = cssText.filter(t => /:active/.test(t))
      const withFeedback = activeRules.filter(t => /transform|scale|opacity|background/.test(t))
      return withFeedback.length >= 3
    })(),
    bottomSafeArea: /safe-area-inset-bottom/.test(all),
    stickyHeaderCss: /\.mobile-header[^{]*\{[^}]*position:\s*sticky/.test(all) || /position:\s*sticky/.test(all),
  }
  return out
}

/** 聚焦输入框后，原生 focus 描边是否出现（移动端这条描边看起来就是输入框下面的一条线） */
function inputFocusOutlineProbe() {
  const res = []
  for (const el of document.querySelectorAll('input:not([type=hidden]), textarea')) {
    const r = el.getBoundingClientRect()
    if (r.width < 20 || r.height < 8) continue
    const s = getComputedStyle(el)
    if (s.outlineStyle !== 'none' && parseFloat(s.outlineWidth) > 0) {
      res.push({ placeholder: el.placeholder || el.type, outline: `${s.outlineWidth} ${s.outlineStyle} ${s.outlineColor}` })
    }
  }
  return res
}

const browser = await chromium.launch()
const STATE_FILE = process.env.AUDIT_STATE || '/tmp/mobile-audit-state.json'
const context = await browser.newContext({
  ...devices['Pixel 5'],
  locale: 'zh-CN',
  ...(existsSync(STATE_FILE) ? { storageState: STATE_FILE } : {}),
})
const page = await context.newPage()

const consoleErrors = []
page.on('console', m => { if (m.type() === 'error') consoleErrors.push(m.text().slice(0, 160)) })
page.on('pageerror', e => consoleErrors.push('pageerror: ' + String(e).slice(0, 160)))

// 登录（勾「记住我」→ token 落 localStorage，跨页面刷新不掉）。
// 登录接口有 10 次/分钟限流，反复跑脚本会触发，所以优先复用已保存的登录态。
if (!existsSync(STATE_FILE)) {
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' })
  await page.fill('input[placeholder="请输入邮箱或用户名"]', USER)
  await page.fill('input[type="password"]', PASS)
  await page.locator('text=记住我').first().click().catch(() => {})
  await page.click('button:has-text("登")')
  await page.waitForURL(u => !u.pathname.includes('/login'), { timeout: 20000 })
  await context.storageState({ path: STATE_FILE })
}

mkdirSync(OUT_DIR, { recursive: true })
const report = []
for (const route of targets) {
  const errorsBefore = consoleErrors.length
  let result
  try {
    await page.goto(`${BASE_URL}${route}`, { waitUntil: 'networkidle', timeout: 30000 })
    await page.waitForTimeout(900)
    result = await page.evaluate(auditInPage)
    // 顺滑度只在第一个页面检查一次（全局性质）
    if (targets[0] === route) {
      const smooth = await page.evaluate(smoothnessProbe)
      result.stats.smoothness = smooth
      // 吸顶顶栏：页面能滚时，滚动后顶栏应停在顶部
      const scrollable = await page.evaluate(() => document.documentElement.scrollHeight > document.documentElement.clientHeight + 50)
      if (scrollable) {
        const stickyTop = await page.evaluate(async () => {
          window.scrollTo(0, 320)
          await new Promise(r => setTimeout(r, 250))
          const h = document.querySelector('.mobile-header')
          const t = h ? Math.round(h.getBoundingClientRect().top) : null
          window.scrollTo(0, 0)
          return t
        })
        smooth.stickyHeaderActual = stickyTop
        smooth.stickyHeaderWorks = stickyTop !== null && Math.abs(stickyTop) <= 2
      }
      for (const [key, ok] of Object.entries(smooth)) {
        if (key === 'stickyHeaderActual') continue
        if (ok === false) {
          result.issues.push({ kind: 'smoothness-missing', detail: `缺少顺滑度要素: ${key}` })
        }
      }
    }
    // 聚焦第一个可见输入框，检查是否冒出浏览器默认描边
    const firstInput = page.locator('.n-input__input-el:visible, input[type=text]:visible, textarea:visible').first()
    if (await firstInput.count().catch(() => 0)) {
      await firstInput.click({ timeout: 3000 }).catch(() => {})
      await page.waitForTimeout(250)
      const outlines = await page.evaluate(inputFocusOutlineProbe)
      if (outlines.length) {
        result.issues.push({ kind: 'input-focus-outline', detail: `聚焦后出现浏览器默认描边（看起来像输入框下的线）：${outlines.length} 个`, samples: outlines.slice(0, 3) })
      }
      await page.keyboard.press('Escape').catch(() => {})
    }
  } catch (e) {
    result = { issues: [{ kind: 'load-failed', detail: String(e).slice(0, 120) }], stats: {} }
  }
  result.route = route
  result.consoleErrors = consoleErrors.slice(errorsBefore)
  report.push(result)
  const bad = result.issues.length + (result.consoleErrors.length ? 1 : 0)
  console.log(`${bad ? '✗' : '✓'} ${route.padEnd(28)} ${result.issues.map(i => i.kind).join(', ') || 'ok'}${result.consoleErrors.length ? ' [console]' : ''}`)
}

// ---- 游客页面（无登录态）：登录/注册/找回密码/404 也要体检 ----
const GUEST_ROUTES = ['/login', '/register', '/forgot-password', '/admin/login', '/nonexistent-page']
if (!process.argv.slice(2).length) {
  const guestCtx = await browser.newContext({ ...devices['Pixel 5'], locale: 'zh-CN' })
  const guestPage = await guestCtx.newPage()
  for (const route of GUEST_ROUTES) {
    const errorsBefore = consoleErrors.length
    let result
    try {
      await guestPage.goto(`${BASE_URL}${route}`, { waitUntil: 'networkidle', timeout: 30000 })
      await guestPage.waitForTimeout(700)
      result = await guestPage.evaluate(auditInPage)
    } catch (e) {
      result = { issues: [{ kind: 'load-failed', detail: String(e).slice(0, 120) }], stats: {} }
    }
    result.route = route
    result.consoleErrors = consoleErrors.slice(errorsBefore)
    report.push(result)
    const bad = result.issues.length + (result.consoleErrors.length ? 1 : 0)
    console.log(`${bad ? '✗' : '✓'} ${route.padEnd(28)} ${result.issues.map(i => i.kind).join(', ') || 'ok'}${result.consoleErrors.length ? ' [console]' : ''}`)
  }
  await guestCtx.close()
}

writeFileSync(`${OUT_DIR}/report.json`, JSON.stringify(report, null, 2))
const total = report.reduce((n, r) => n + r.issues.length, 0)
console.log(`\n共 ${report.length} 个页面，${total} 个问题；详情：${OUT_DIR}/report.json`)
await browser.close()
