<template>
  <div class="legal-page">
    <div class="legal-header">
      <h1 class="title">服务条款</h1>
      <p class="subtitle">最后更新：2026 年 8 月</p>
    </div>

    <!-- 手机端：目录（点一下跳到对应条款）。
         吸顶交给全局 .app-sticky-toolbar（贴在吸顶顶栏下沿）；sticky 只能在
         「直接父盒」里移动，所以直接挂在 .legal-page 下，别再套 wrapper。 -->
    <div v-if="appStore.isMobile" class="app-sticky-toolbar legal-toc-bar">
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

    <n-card :bordered="false" class="legal-card">
      <p class="lead">
        欢迎使用本平台提供的订阅服务。请您在使用本服务前仔细阅读本《服务条款》（以下简称「本条款」）。
        您通过注册、登录或使用本服务的任何行为，即表示您已阅读、理解并同意受本条款约束。
        如您不同意本条款的任何内容，请停止注册或使用本服务。
      </p>

      <section id="sec-service" class="legal-section">
        <h2 class="section-title">一、服务说明</h2>
        <p class="section-para">
          1. 本平台为纯时间制网络加速订阅服务：套餐按「使用时长」与「设备数量」售卖，
          不包含流量计量，服务有效期以订单确认时展示的时长为准。
        </p>
        <p class="section-para">
          2. 服务内容包括：订阅链接、客户端配置信息、节点访问服务以及相关的技术支持与帮助文档。
          支付成功且订单确认后，系统将自动为您开通对应时长的服务。
        </p>
        <p class="section-para">
          3. 我们有权根据运营需要对服务内容、套餐价格、节点资源进行调整，
          调整前将在站内进行公告；已生效的订单不受价格调整影响。
        </p>
      </section>

      <section id="sec-account" class="legal-section">
        <h2 class="section-title">二、账户</h2>
        <p class="section-para">
          1. 您在注册时应提供真实、准确、完整的信息，并妥善保管账户密码。
          因您保管不善导致的账户被盗用、信息泄露等损失，由您自行承担。
        </p>
        <p class="section-para">
          2. 账户仅限本人使用，不得转让、出借、出售或与他人共享。
          如发现账户异常登录或使用行为，请立即修改密码并联系客服处理。
        </p>
        <p class="section-para">
          3. 同一账户下多个设备的并发使用受套餐设备数限制，超出限制的设备将无法正常接入，
          请通过「设备管理」页面及时清理不再使用的设备。
        </p>
      </section>

      <section id="sec-pay" class="legal-section">
        <h2 class="section-title">三、支付与退款政策</h2>
        <p class="section-para">
          1. 本平台支持余额支付及多种在线支付方式，具体以购买页面展示为准。
          订单金额、优惠及实付金额以支付确认页展示为准。
        </p>
        <p class="section-para">
          2. 订单支付成功后服务即时生效；余额充值原则上不支持提现，仅可用于本站内消费。
        </p>
        <p class="section-para">
          3. 退款政策：购买后 <strong>7 天内未激活</strong>（未使用订阅链接在任何设备上接入）的订单，
          可联系客服申请全额退款；已激活、超过退款期限或违反本条款使用规范的订单，不支持退款。
        </p>
        <p class="section-para">
          4. 因支付渠道退回、重复支付等客观原因产生的款项差异，请联系客服核实后退回原支付账户或账户余额。
        </p>
      </section>

      <section id="sec-usage" class="legal-section">
        <h2 class="section-title">四、使用规范</h2>
        <p class="section-para">
          1. 本服务仅限个人合法使用。您承诺不将本服务用于任何违反中华人民共和国法律法规
          及所在地法律的活动，包括但不限于网络攻击、诈骗、传播违法信息、侵犯他人权益等。
        </p>
        <p class="section-para">
          2. 您不得转售、批量分发、共享订阅链接或将其用于商业用途，
          不得对节点进行恶意扫描、攻击或采取任何影响服务稳定性的行为。
        </p>
        <p class="section-para">
          3. 违反上述使用规范的，我们有权在不另行通知的情况下暂停或终止您的服务，
          且已支付的费用不予退还；情节严重者将依法配合有关部门处理。
        </p>
      </section>

      <section id="sec-disclaimer" class="legal-section">
        <h2 class="section-title">五、免责声明</h2>
        <p class="section-para">
          1. 本服务按「现状」提供。因不可抗力（如自然灾害、网络故障、运营商调整、第三方服务中断等）
          导致的服务中断或异常，本平台不承担由此产生的责任，但会尽最大努力尽快恢复。
        </p>
        <p class="section-para">
          2. 在法律允许的最大范围内，本平台不对因使用或无法使用本服务而产生的任何直接或间接损失
          （包括但不限于利润损失、数据丢失、业务中断）承担责任。
        </p>
        <p class="section-para">
          3. 您应自行负责设备安全与重要数据的备份，因使用第三方客户端或自行修改配置造成的问题，
          本平台仅提供协助排查，不承担相应责任。
        </p>
      </section>

      <section id="sec-privacy" class="legal-section">
        <h2 class="section-title">六、隐私简述</h2>
        <p class="section-para">
          我们仅收集为您提供服务所必需的信息（如账户信息、订单信息、设备信息等），
          并采取合理的加密与安全措施加以保护。我们不会向任何第三方出售您的个人信息。
          完整的个人信息处理规则，请查阅《隐私政策》。
        </p>
      </section>

      <section id="sec-contact" class="legal-section">
        <h2 class="section-title">七、联系方式</h2>
        <p class="section-para">
          如您对本条款有任何疑问，或需要咨询、投诉、退款，可以通过以下方式联系我们：
        </p>
        <ul class="section-list">
          <li>站内「工单」：前往帮助中心或工单页面提交问题，我们通常会在 24 小时内回复；</li>
          <li>帮助中心：查看「联系我们」板块中展示的客服邮箱、Telegram、QQ 群等联系方式；</li>
          <li>工作时间：周一至周五 9:00 - 18:00（法定节假日除外）。</li>
        </ul>
      </section>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onActivated, onUnmounted, onDeactivated } from 'vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

/* ---------------- 手机端目录 ----------------
   吸顶由全局 .app-sticky-toolbar 负责（position: sticky，top: var(--mobile-header-h)），
   页面只管两件事：点一下平滑跳到对应条款 + 滚动时高亮当前条款。 */
const tocSections = [
  { id: 'sec-service', label: '服务说明' },
  { id: 'sec-account', label: '账户' },
  { id: 'sec-pay', label: '支付与退款' },
  { id: 'sec-usage', label: '使用规范' },
  { id: 'sec-disclaimer', label: '免责声明' },
  { id: 'sec-privacy', label: '隐私简述' },
  { id: 'sec-contact', label: '联系方式' },
]

const activeSection = ref(tocSections[0].id)

function jumpTo(id: string) {
  activeSection.value = id
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

let scrollBound = false
let spyRaf = 0

function updateActiveSection() {
  spyRaf = 0
  if (!appStore.isMobile) return
  // 当前条款 = 吸顶顶栏 + 目录条（52 + 58 ≈ 110px）下方那一屏里最靠上的一节
  let current = tocSections[0].id
  for (const s of tocSections) {
    const el = document.getElementById(s.id)
    if (el && el.getBoundingClientRect().top <= 130) current = s.id
  }
  // 滚到底时高亮最后一节（末节通常滚不到顶部，否则选中态会停在上一节）
  if (window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 4) {
    current = tocSections[tocSections.length - 1].id
  }
  activeSection.value = current
}

function onPageScroll() {
  if (!spyRaf) spyRaf = requestAnimationFrame(updateActiveSection)
}

// keep-alive 缓存的页面在重新激活时重新绑定滚动监听
function bindScroll() {
  if (scrollBound) return
  scrollBound = true
  window.addEventListener('scroll', onPageScroll, { passive: true })
  updateActiveSection()
}

function unbindScroll() {
  if (!scrollBound) return
  scrollBound = false
  window.removeEventListener('scroll', onPageScroll)
  if (spyRaf) {
    cancelAnimationFrame(spyRaf)
    spyRaf = 0
  }
}

onMounted(bindScroll)
onActivated(bindScroll)
onDeactivated(unbindScroll)
onUnmounted(unbindScroll)
</script>

<style scoped>
/* ============================================================
   服务条款 —— 手机端按 docs/mobile-app-spec.md 第 1 / 3 节改造：
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

/* ---------------- 手机端目录条 ----------------
   外观与吸顶（position: sticky + top: var(--mobile-header-h) + 毛玻璃 + 发丝线）
   全部来自全局 .app-sticky-toolbar，这里只排一行可横滑的 chip。 */
.legal-toc-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}
.legal-toc-bar::-webkit-scrollbar {
  display: none;
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

  /* 从目录跳过来时，标题不被顶栏（52px）+ 吸顶目录条挡住 */
  .legal-section { scroll-margin-top: calc(var(--mobile-header-h, 52px) + 66px); }
}
</style>
