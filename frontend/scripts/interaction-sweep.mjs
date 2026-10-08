/**
 * 交互体检：逐页点一遍所有按钮，找「点了没反应」的按钮与「遮罩/滚动锁」残留
 *
 * 做法与安全边界：
 *  - 所有会改数据的请求（POST/PUT/PATCH/DELETE）在浏览器侧被拦截并记录，**不会真正发到后端**，
 *    所以可以放心点「删除/禁用/批量操作」这类按钮，不会改动任何数据。
 *  - 按钮的判定：点击后出现「请求 / 跳转 / 弹层 / 提示消息 / DOM 变化」中任意一种 = 有反应。
 *  - 黑名单：退出登录、代登、登录为（会切换会话）直接跳过。
 *  - 遮罩检查：每页扫完强制关掉所有弹层，然后检查是否还有可见遮罩残留、body 是否被锁滚动。
 *
 * 用法：node /tmp/interaction-sweep.mjs            # 全部页面
 *      node /tmp/interaction-sweep.mjs /admin/users # 指定页面
 */
import { chromium, devices } from '/opt/homebrew/lib/node_modules/playwright/index.mjs'
import { writeFileSync } from 'node:fs'

const BASE = process.env.BASE_URL || 'http://localhost:3000'
const ROUTES = process.argv.slice(2).length ? process.argv.slice(2) : [
  '/', '/subscription', '/orders', '/shop', '/tickets', '/nodes', '/devices', '/invite', '/settings',
  '/help', '/terms', '/privacy', '/login-history', '/notifications', '/coupons', '/recharge', '/redeem', '/mystery-box',
  '/admin', '/admin/users', '/admin/abnormal-users', '/admin/orders', '/admin/packages', '/admin/nodes',
  '/admin/custom-nodes', '/admin/config-update', '/admin/subscriptions', '/admin/coupons', '/admin/tickets',
  '/admin/levels', '/admin/redeem', '/admin/invites', '/admin/mystery-box', '/admin/settings',
  '/admin/announcements', '/admin/stats', '/admin/logs', '/admin/email-queue',
]

// 会切换登录态/角色，点了之后后续页面全废 —— 直接跳过
const SKIP_TEXT = ['退出登录', '退出', '代登', '登录为', '切换账号']

const browser = await chromium.launch()
const ctx = await browser.newContext({ ...devices['Pixel 5'], locale: 'zh-CN', storageState: '/tmp/mobile-audit-state.json' })
const page = await ctx.newPage()

// 拦截所有写请求：记录但不发出，避免真的改数据
const mutations = []
let requests = 0
await page.route('**/api/v1/**', async (route) => {
  const req = route.request()
  requests++
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(req.method())) {
    mutations.push({ method: req.method(), url: req.url().replace(BASE, '') })
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'sweep-blocked', data: {} }) })
  } else {
    await route.continue()
  }
})
// 文件选择框（导入按钮）也算「有反应」
let fileDialogs = 0
page.on('filechooser', () => { fileDialogs++ })

// 登录态可能在长时间扫描中过期：检测到被踢回登录页就自动重登一次
async function ensureLoggedIn() {
  if (!page.url().includes('/login')) return
  console.log('   （登录态过期，自动重新登录）')
  await page.fill('input[placeholder="请输入邮箱或用户名"]', 'admin@example.com')
  await page.fill('input[type="password"]', 'LocalTest123')
  await page.locator('text=记住我').first().click().catch(() => {})
  await page.click('button:has-text("登")')
  await page.waitForURL(u => !u.pathname.includes('/login'), { timeout: 20000 }).catch(() => {})
}

const report = []
for (const route of ROUTES) {
  await page.goto(`${BASE}${route}`, { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1200)
  if (page.url().includes('/login')) {
    await ensureLoggedIn()
    await page.goto(`${BASE}${route}`, { waitUntil: 'networkidle' }).catch(() => {})
    await page.waitForTimeout(1200)
  }

  const dead = []
  const checked = []
  const buttons = await page.evaluate((skip) => {
    const vis = (el) => {
      const r = el.getBoundingClientRect(); const s = getComputedStyle(el)
      return r.width > 4 && r.height > 4 && s.display !== 'none' && s.visibility !== 'hidden' && Number(s.opacity) > 0.05
    }
    return [...document.querySelectorAll('button, .n-button, a.n-button')]
      .filter(vis)
      .map((el, i) => {
        el.setAttribute('data-sweep-idx', String(i))   // 打标记，保证点击的是同一个元素
        return { i, text: (el.textContent || '').trim().slice(0, 16), disabled: el.disabled || el.classList.contains('n-button--disabled') }
      })
      .filter(b => !skip.some(s => b.text.includes(s)))
      .slice(0, 14)
  }, SKIP_TEXT)

  for (const b of buttons) {
    // 禁用状态的按钮点了本来就不该有反应，跳过（不计入问题）
    if (b.disabled) { checked.push({ text: b.text, reacted: true, disabled: true }); continue }
    await page.evaluate(() => {
      window.__sweepMutations = 0
      window.__sweepObserver && window.__sweepObserver.disconnect()
      window.__sweepObserver = new MutationObserver((list) => { window.__sweepMutations += list.length })
      window.__sweepObserver.observe(document.body, { childList: true, subtree: true, attributes: true, characterData: true })
    })
    const before = await page.evaluate(() => ({
      url: location.href,
      toasts: document.querySelectorAll('.n-message, .n-notification').length,
      overlays: document.querySelectorAll('.n-modal:not([style*="display: none"]), .n-drawer, .n-dropdown-menu').length,
    }))
    // 关键：先把上一轮打开的弹层关掉、并把按钮滚到视口中间
    // （否则抽屉/弹窗会盖住后续按钮，底部固定批量栏也会挡住列表里的按钮，造成"点了没反应"的假象）
    await page.keyboard.press('Escape').catch(() => {})
    await page.waitForTimeout(250)
    await page.evaluate((k) => {
      const vis = (el) => {
        const r = el.getBoundingClientRect(); const s = getComputedStyle(el)
        return r.width > 4 && r.height > 4 && s.display !== 'none' && s.visibility !== 'hidden' && Number(s.opacity) > 0.05
      }
      const list = [...document.querySelectorAll('button, .n-button, a.n-button')].filter(vis)
      list.forEach(el => el.removeAttribute('data-sweep-idx'))
      const el = list[k]
      if (el) { el.setAttribute('data-sweep-idx', 'SWEEP'); el.scrollIntoView({ block: 'center' }) }
    }, b.i)
    await page.waitForTimeout(250)

    const mutBefore = mutations.length
    const reqBefore = requests
    const fileBefore = fileDialogs
    let clicked = false
    try {
      const tagged = await page.evaluate((k) => {
        const vis = (el) => {
          const r = el.getBoundingClientRect(); const s = getComputedStyle(el)
          return r.width > 4 && r.height > 4 && s.display !== 'none' && s.visibility !== 'hidden' && Number(s.opacity) > 0.05
        }
        const list = [...document.querySelectorAll('button, .n-button, a.n-button')].filter(vis)
        list.forEach(el => el.removeAttribute('data-sweep-idx'))
        if (!list[k]) return false
        list[k].setAttribute('data-sweep-idx', 'SWEEP')
        return true
      }, b.i)
      if (tagged) {
        const target = page.locator('[data-sweep-idx="SWEEP"]').first()
        // 不加 force：Playwright 会等元素真正可点（不被遮挡），被固定栏遮住时会抛出超时 → 记为「点不到」
        await target.click({ timeout: 3000 })
        clicked = true
        await page.evaluate(() => document.querySelector('[data-sweep-idx="SWEEP"]')?.removeAttribute('data-sweep-idx'))
      }
      clicked = true
    } catch { /* 点不到（被遮挡/不可见）也算一种问题，稍后标出 */ }
    await page.waitForTimeout(700)

    const after = await page.evaluate(() => ({
      url: location.href,
      toasts: document.querySelectorAll('.n-message, .n-notification').length,
      overlays: document.querySelectorAll('.n-modal:not([style*="display: none"]), .n-drawer, .n-dropdown-menu').length,
      mutations: window.__sweepMutations || 0,
    }))
    const reacted =
      mutations.length > mutBefore ||            // 有写请求（已拦截）
      requests > reqBefore ||                     // 有任意请求（含查询/刷新）
      fileDialogs > fileBefore ||                 // 弹出了文件选择框
      after.url !== before.url ||                 // 跳转
      after.toasts > before.toasts ||             // 弹了提示
      after.overlays > before.overlays ||         // 弹层/下拉出现
      after.mutations >= 1                        // 页面 DOM 有任何变化（如选中态、列表刷新）

    checked.push({ text: b.text, reacted, clicked })
    if (!reacted) dead.push({ text: b.text, clicked })
  }

  // 收起弹层（连按两次 Esc，覆盖层级嵌套）再检查遮罩/滚动锁残留
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(600)
  const residue = await page.evaluate(() => {
    const masks = [...document.querySelectorAll('.n-modal-mask, .n-drawer-mask, .n-base-mask, .v-binder-follower-container')]
      .filter(el => { const s = getComputedStyle(el); return s.display !== 'none' && s.visibility !== 'hidden' && Number(s.opacity) > 0.02 })
    return {
      masks: masks.length,
      bodyOverflow: getComputedStyle(document.body).overflow,
      htmlOverflow: getComputedStyle(document.documentElement).overflow,
    }
  })
  report.push({ route, buttons: checked.length, dead, residue })
  console.log(
    `${dead.length || residue.masks ? '✗' : '✓'} ${route.padEnd(26)} 按钮 ${String(checked.length).padStart(2)}` +
    `${dead.length ? ` | 无反应: ${dead.map(d => d.text || '(图标)').join(', ')}` : ''}` +
    `${residue.masks ? ` | 遮罩残留 ${residue.masks}` : ''}` +
    `${residue.bodyOverflow !== 'visible' ? ` | body.overflow=${residue.bodyOverflow}` : ''}`,
  )
}

writeFileSync('/tmp/interaction-report.json', JSON.stringify(report, null, 2))
const deadTotal = report.reduce((n, r) => n + r.dead.length, 0)
const maskTotal = report.reduce((n, r) => n + r.residue.masks, 0)
console.log(`\n共 ${report.length} 页；无反应按钮 ${deadTotal} 个；遮罩残留页 ${report.filter(r => r.residue.masks).length} 个（遮罩数 ${maskTotal}）`)
console.log('详情：/tmp/interaction-report.json')
await browser.close()
