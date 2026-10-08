/**
 * 输入框「打字时下面出现一条线」专项探针
 *
 * 我们在 CSS 里没找到给输入框加下划线的规则，所以这个脚本用来定位真正的来源：
 *  1. 聚焦输入框前后，比对元素树 —— 有没有「新出现的」细长元素（高度 ≤ 3px）
 *  2. 打印原生 input 的 border / box-shadow / outline / appearance / text-decoration
 *  3. 检查 ::before/::after 伪元素的尺寸与背景
 *  4. 检查 autofill 相关样式（-webkit-box-shadow 覆盖等）
 */
import { chromium, devices } from '/opt/homebrew/lib/node_modules/playwright/index.mjs'

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000'
const PAGES = process.argv.slice(2).length
  ? process.argv.slice(2)
  : ['/settings', '/admin/users', '/admin/settings', '/login']

const browser = await chromium.launch()
const context = await browser.newContext({ ...devices['Pixel 5'], locale: 'zh-CN' })
const page = await context.newPage()

// 收集当前所有「细长」元素（可能是那条线）
const thinProbe = () => {
  const res = []
  for (const el of document.querySelectorAll('body *')) {
    const r = el.getBoundingClientRect()
    if (r.height > 0 && r.height <= 3.5 && r.width >= 40) {
      const s = getComputedStyle(el)
      res.push({
        el: el.tagName.toLowerCase() + '.' + (el.getAttribute('class') || '').split(/\s+/).slice(0, 2).join('.'),
        h: +r.height.toFixed(1),
        w: Math.round(r.width),
        bg: s.backgroundColor,
        bs: s.boxShadow.slice(0, 60),
        bb: s.borderBottomWidth + ' ' + s.borderBottomStyle + ' ' + s.borderBottomColor,
        pseudo: (() => {
          const a = getComputedStyle(el, '::after')
          const b = getComputedStyle(el, '::before')
          return {
            after: a.content !== 'none' ? `${a.height}/${a.backgroundColor}/${a.borderBottomWidth}` : null,
            before: b.content !== 'none' ? `${b.height}/${b.backgroundColor}` : null,
          }
        })(),
      })
    }
  }
  return res
}

const inputStyles = () => {
  const out = []
  for (const el of document.querySelectorAll('input, textarea')) {
    if (el.offsetParent === null && el.type !== 'hidden') continue
    const s = getComputedStyle(el)
    const parent = el.closest('.n-input, .n-input-wrapper')
    const ps = parent ? getComputedStyle(parent) : null
    out.push({
      type: el.type,
      placeholder: el.placeholder,
      autocomplete: el.getAttribute('autocomplete'),
      appearance: s.appearance + '/' + s.webkitAppearance,
      border: `${s.borderTopWidth} ${s.borderRightWidth} ${s.borderBottomWidth} ${s.borderLeftWidth}`,
      borderBottom: `${s.borderBottomWidth} ${s.borderBottomStyle} ${s.borderBottomColor}`,
      boxShadow: s.boxShadow,
      outline: s.outlineWidth + ' ' + s.outlineStyle,
      textDecoration: s.textDecorationLine,
      backgroundImage: s.backgroundImage,
      wrapperShadow: ps ? ps.boxShadow.slice(0, 80) : null,
      wrapperBorderBottom: ps ? ps.borderBottomWidth : null,
    })
  }
  return out
}

await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' })
await page.fill('input[placeholder="请输入邮箱或用户名"]', 'admin@example.com')
await page.fill('input[type="password"]', 'LocalTest123')
await page.locator('text=记住我').first().click().catch(() => {})
await page.click('button:has-text("登")')
await page.waitForURL(u => !u.pathname.includes('/login'), { timeout: 20000 })

for (const route of PAGES) {
  await page.goto(`${BASE_URL}${route}`, { waitUntil: 'networkidle' })
  await page.waitForTimeout(800)
  const before = await page.evaluate(thinProbe)
  const inputs = await page.evaluate(inputStyles)
  console.log(`\n===== ${route} =====`)
  console.log('输入框样式：')
  for (const i of inputs.slice(0, 4)) console.log('  ', JSON.stringify(i))
  // 聚焦第一个可见输入框，看冒不冒出线
  const target = page.locator('input:visible').first()
  if (await target.count()) {
    await target.click()
    await page.waitForTimeout(400)
    const after = await page.evaluate(thinProbe)
    const key = x => `${x.el}|${x.h}|${x.w}`
    const beforeKeys = new Set(before.map(key))
    const added = after.filter(x => !beforeKeys.has(key(x)))
    console.log('聚焦后「新出现」的细长元素：')
    if (!added.length) console.log('   无')
    for (const a of added.slice(0, 8)) console.log('  ', JSON.stringify(a))
    const focused = await page.evaluate(() => {
      const el = document.activeElement
      if (!el) return null
      const s = getComputedStyle(el)
      return {
        tag: el.tagName, appearance: s.appearance, borderBottom: `${s.borderBottomWidth} ${s.borderBottomStyle} ${s.borderBottomColor}`,
        boxShadow: s.boxShadow, outline: s.outline, backgroundImage: s.backgroundImage,
      }
    })
    console.log('聚焦中的元素：', JSON.stringify(focused))
    await page.keyboard.press('Escape').catch(() => {})
  } else {
    console.log('（该页没有可见输入框）')
  }
}

await browser.close()
