<template>
  <div class="legal-page">
    <div class="legal-header">
      <h1 class="title">隐私政策</h1>
      <p class="subtitle">最后更新：2026 年 8 月</p>
    </div>

    <!-- 手机端：目录（点一下跳到对应条款；吸顶不跟着页面滚走） -->
    <div
      v-if="appStore.isMobile"
      ref="tocSlot"
      class="legal-toc-slot"
      :style="tocStuck ? { height: tocHeight + 'px' } : undefined"
    >
      <div class="app-sticky-toolbar legal-toc-bar" :class="{ 'is-stuck': tocStuck }">
        <button
          v-for="s in tocSections"
          :key="s.id"
          type="button"
          class="legal-toc-chip"
          :class="{ 'is-active': activeSection === s.id }"
          @click="jumpTo(s.id)"
        >
          {{ s.label }}
        </button>
      </div>
    </div>

    <n-card :bordered="false" class="legal-card">
      <p class="lead">
        本《隐私政策》说明了本平台（以下简称「我们」）如何收集、使用、存储、共享和保护您的个人信息。
        我们非常重视您的隐私，将严格遵守相关法律法规，并采取合理的安全措施保障您的个人信息安全。
        请您在使用本服务前仔细阅读本政策。
      </p>

      <section id="sec-collect" class="legal-section">
        <h2 class="section-title">一、我们收集的信息</h2>
        <p class="section-para">为向您提供服务，我们可能收集以下类型的信息：</p>
        <ul class="section-list">
          <li><strong>账户信息</strong>：注册时提供的用户名、邮箱（或手机号）、密码（经加密存储）等；</li>
          <li><strong>订单与支付信息</strong>：套餐选择、订单金额、支付方式、支付渠道单号等；</li>
          <li><strong>设备信息</strong>：用于设备数量管理的设备标识、客户端类型等；</li>
          <li><strong>登录与访问日志</strong>：IP 地址、登录时间、浏览器/客户端标识等，用于安全审计与异常检测；</li>
          <li><strong>工单与沟通内容</strong>：您提交工单或与客服沟通时提供的信息。</li>
        </ul>
      </section>

      <section id="sec-use" class="legal-section">
        <h2 class="section-title">二、信息的使用</h2>
        <p class="section-para">我们收集的信息仅用于以下目的：</p>
        <ul class="section-list">
          <li>提供、维护与优化订阅服务，包括订单处理、服务开通与续期；</li>
          <li>身份验证、账户管理与安全防护（如检测异常登录、防止滥用）；</li>
          <li>处理工单、提供技术支持与售后服务；</li>
          <li>发送必要的服务通知（如订单状态、公告提醒）；</li>
          <li>在法律允许的范围内改善产品体验与服务质量。</li>
        </ul>
      </section>

      <section id="sec-store" class="legal-section">
        <h2 class="section-title">三、信息的存储</h2>
        <p class="section-para">
          1. 您的个人信息存储于受保护的服务器环境中，传输过程采用加密协议保护；
          我们仅在本政策所述目的所需的期限内保留必要的信息，超出期限后将进行删除或匿名化处理。
        </p>
        <p class="section-para">
          2. 我们采取访问控制、加密、日志审计等合理的技术与管理措施，
          防止个人信息被未经授权的访问、泄露、篡改或丢失。
        </p>
      </section>

      <section id="sec-share" class="legal-section">
        <h2 class="section-title">四、信息的共享</h2>
        <p class="section-para">
          1. 我们不会向任何第三方出售或出租您的个人信息。
        </p>
        <p class="section-para">
          2. 仅在以下情形下，我们可能向特定第三方提供必要的信息：
        </p>
        <ul class="section-list">
          <li>支付渠道：为完成在线支付，向支付服务商提供必要的订单信息；</li>
          <li>法律法规要求：根据司法机关、行政机关的合法要求提供；</li>
          <li>获得您的明确授权同意。</li>
        </ul>
      </section>

      <section id="sec-rights" class="legal-section">
        <h2 class="section-title">五、您的权利</h2>
        <p class="section-para">依据相关法律法规，您对自己的个人信息享有以下权利：</p>
        <ul class="section-list">
          <li><strong>查阅与更正</strong>：在「个人设置」页面查看并修改您的账户资料；</li>
          <li><strong>删除</strong>：请求删除与您相关的个人信息，法律法规另有规定的除外；</li>
          <li><strong>注销账户</strong>：通过工单或客服联系我们申请注销账户；</li>
          <li><strong>撤回同意</strong>：撤回对个人信息处理的授权（撤回可能影响部分功能的正常使用）。</li>
        </ul>
        <p class="section-para">如您需要行使上述权利，请通过文末的联系方式与我们联系，我们将在合理期限内处理您的请求。</p>
      </section>

      <section id="sec-cookie" class="legal-section">
        <h2 class="section-title">六、Cookie 与本地存储</h2>
        <p class="section-para">
          1. 为改善使用体验，我们可能使用 Cookie 或浏览器本地存储保存您的登录状态、主题偏好等必要信息；
          这些信息仅保存在您的设备上，不会被用于追踪您的其他网络行为。
        </p>
        <p class="section-para">
          2. 您可以在浏览器设置中清除或禁用 Cookie，但这可能导致您需要重新登录或部分功能无法正常使用。
        </p>
      </section>

      <section id="sec-minor" class="legal-section">
        <h2 class="section-title">七、未成年人保护</h2>
        <p class="section-para">
          本服务面向成年人提供。若您为未成年人，请在监护人阅读并同意本政策的前提下使用本服务，
          并请在监护人指导下进行注册与支付。
        </p>
      </section>

      <section id="sec-update" class="legal-section">
        <h2 class="section-title">八、政策更新</h2>
        <p class="section-para">
          我们可能适时更新本政策。更新后的政策将在站内公布，重大变更将通过公告或显著方式通知您。
          您继续使用本服务即视为接受更新后的政策。
        </p>
      </section>

      <section id="sec-contact" class="legal-section">
        <h2 class="section-title">九、联系方式</h2>
        <p class="section-para">
          如您对本政策有任何疑问、意见或需要行使个人信息相关权利，请通过以下方式联系我们：
        </p>
        <ul class="section-list">
          <li>站内「工单」：前往帮助中心或工单页面提交问题；</li>
          <li>帮助中心：查看「联系我们」板块中展示的客服邮箱、Telegram、QQ 群等联系方式。</li>
        </ul>
      </section>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onActivated, onUnmounted, onDeactivated } from 'vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

/* ---------------- 手机端目录（吸顶） ----------------
   为什么不用纯 CSS：全局 .app-sticky-toolbar 是 position: sticky，但页面内容被
   .n-scrollbar-container（overflow: scroll）包着，真正滚动的是 window，该容器自身
   不滚动 —— 实测滚动 600px 后 sticky 元素 top 变成 -588，照样滚走。公共文件不可改，
   所以这里用等效实现：滚出视口顶部就切 fixed（.is-stuck），未吸顶时留在文档流里，
   吸顶时给插槽补上等高占位避免内容跳动。 */
const tocSections = [
  { id: 'sec-collect', label: '信息收集' },
  { id: 'sec-use', label: '信息使用' },
  { id: 'sec-store', label: '信息存储' },
  { id: 'sec-share', label: '信息共享' },
  { id: 'sec-rights', label: '您的权利' },
  { id: 'sec-cookie', label: 'Cookie' },
  { id: 'sec-minor', label: '未成年人' },
  { id: 'sec-update', label: '政策更新' },
  { id: 'sec-contact', label: '联系方式' },
]

const tocSlot = ref<HTMLElement | null>(null)
const tocStuck = ref(false)
const tocHeight = ref(58)
const activeSection = ref(tocSections[0].id)

function jumpTo(id: string) {
  activeSection.value = id
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

let scrollBound = false
let tocRaf = 0

function measureToc() {
  tocRaf = 0
  const slot = tocSlot.value
  if (!slot || !appStore.isMobile) {
    tocStuck.value = false
    return
  }
  const bar = slot.firstElementChild as HTMLElement | null
  const h = bar ? Math.round(bar.getBoundingClientRect().height) : 0
  if (h > 0) tocHeight.value = h
  tocStuck.value = slot.getBoundingClientRect().top < 0
  // 当前条款 = 视口上沿往下 72px 处所在的那一节（目录要有选中态才像 App）
  let current = tocSections[0].id
  for (const s of tocSections) {
    const el = document.getElementById(s.id)
    if (el && el.getBoundingClientRect().top <= 72) current = s.id
  }
  // 滚到底时高亮最后一节（末节通常滚不到顶部，否则选中态会停在上一节）
  if (window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 4) {
    current = tocSections[tocSections.length - 1].id
  }
  activeSection.value = current
}

function onPageScroll() {
  if (!tocRaf) tocRaf = requestAnimationFrame(measureToc)
}

function bindScroll() {
  if (scrollBound) return
  scrollBound = true
  window.addEventListener('scroll', onPageScroll, { passive: true })
  window.addEventListener('resize', onPageScroll, { passive: true })
  measureToc()
}

function unbindScroll() {
  if (!scrollBound) return
  scrollBound = false
  window.removeEventListener('scroll', onPageScroll)
  window.removeEventListener('resize', onPageScroll)
  if (tocRaf) {
    cancelAnimationFrame(tocRaf)
    tocRaf = 0
  }
}

onMounted(bindScroll)
onActivated(bindScroll)
onDeactivated(unbindScroll)
onUnmounted(unbindScroll)
</script>

<style scoped>
/* ============================================================
   隐私政策 —— 手机端按 docs/mobile-app-spec.md 第 1 / 3 节改造：
   · 根容器手机端不再自带左右内边距（全局统一 10px），原来是 8px 叠加成白边
   · 正文 13px / 行高 1.75，条款按卡片分组，不再是小字密集排版
   · 目录吸顶（.app-sticky-toolbar），每个条目可点区域 ≥40px
   ============================================================ */

.legal-page {
  padding: 0;
}

.legal-header {
  margin-bottom: 16px;
}

.title {
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 8px 0;
  background: var(--brand-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  font-size: 13px;
  color: var(--text-color-secondary, #666);
  margin: 0;
}

/* ---------------- 手机端目录条 ---------------- */
.legal-toc-slot {
  display: block;
}

.legal-toc-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  /* 全局 .app-sticky-toolbar 的 -12px 负边距是给「页面根容器自带 12px 内边距」的
     页面用的；本页根容器手机端无左右内边距，负边距会把目录条撑出屏幕（横向溢出），
     这里归零；左右留 1px 透明边保证吸顶前后高度一致。 */
  margin: 0 0 10px;
  padding: 8px 10px;
  border: 1px solid transparent;
}
.legal-toc-bar::-webkit-scrollbar {
  display: none;
}

/* 吸顶态：sticky 在本布局被 .n-scrollbar-container（overflow: scroll 且不滚动）吃掉，
   滚出视口顶部后改用 fixed 顶到最上方（未吸顶时留在文档流里，不挡内容）。 */
.legal-toc-bar.is-stuck {
  position: fixed;
  top: 0;
  left: 10px;
  right: 10px;
  z-index: 30;
  margin: 0;
  border-color: color-mix(in srgb, var(--border-color, #e5e7eb) 70%, transparent);
  border-radius: 0 0 16px 16px;
}

.legal-toc-chip {
  flex: 0 0 auto;
  /* 触控目标 ≥40px */
  min-height: 40px;
  padding: 0 14px;
  border-radius: 20px;
  border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 70%, transparent);
  background: var(--bg-color, #fff);
  color: var(--text-color-secondary, #64748b);
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}
.legal-toc-chip.is-active {
  background: var(--primary-color, #4f46e5);
  border-color: transparent;
  color: #fff;
  font-weight: 600;
}

.legal-card {
  border-radius: 16px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 16px rgba(15, 23, 42, 0.04);
}

.lead {
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-color);
  margin: 0 0 8px;
}

.legal-section {
  margin-bottom: 28px;
}

.legal-section:last-child {
  margin-bottom: 0;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0 0 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color, #eee);
}

.section-para {
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-color);
  margin: 0 0 8px;
}

.section-list {
  margin: 0 0 8px;
  padding-left: 20px;
}

.section-list li {
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-color);
  margin: 4px 0;
}

@media (max-width: 767px) {
  /* 契约 §1：手机端根容器不再自带左右内边距，只保留纵向留白 */
  .legal-page { padding: 8px 0 0; }
  .title { font-size: 22px; }
  .subtitle { font-size: 13px; }
  .legal-header { margin-bottom: 10px; }

  /* 条款分组：每条像一张 App 卡片，长文阅读不再是一整块小字墙 */
  .legal-section {
    padding: 12px 14px;
    margin-bottom: 10px;
    border-radius: 14px;
    background: var(--bg-page-color, #f8fafc);
    border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 60%, transparent);
  }
  .legal-section:last-child { margin-bottom: 0; }

  .section-title {
    font-size: 16px;
    margin: 0 0 8px;
    padding-bottom: 0;
    border-bottom: none;
  }

  /* 正文 13px / 行高 1.75；说明文字字号不再掉到 12px 以下 */
  .lead,
  .section-para,
  .section-list li {
    font-size: 13px;
    line-height: 1.75;
  }
  .section-para { margin: 0 0 6px; }
  .section-list { padding-left: 18px; }
  .section-list li { margin: 5px 0; }

  /* 从目录跳过来时，标题不被吸顶的目录条挡住 */
  .legal-section { scroll-margin-top: 72px; }
}
</style>
