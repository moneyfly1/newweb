<template>
  <div class="help-page">
    <n-space vertical :size="24">
      <h1 class="title">帮助中心</h1>

      <!-- 手机端：板块导航条（App 里的分段控件，点一下跳到对应板块，不跟着页面滚走） -->
      <div
        v-if="appStore.isMobile"
        ref="navSlot"
        class="help-nav-slot"
        :style="navStuck ? { height: navHeight + 'px' } : undefined"
      >
        <div class="app-sticky-toolbar help-nav-bar" :class="{ 'is-stuck': navStuck }">
          <button
            v-for="s in pageSections"
            :key="s.id"
            type="button"
            class="help-nav-chip"
            :class="{ 'is-active': activeSection === s.id }"
            @click="jumpTo(s.id)"
          >
            {{ s.label }}
          </button>
        </div>
      </div>

      <n-card id="sec-faq" title="常见问题" :bordered="false">
        <n-collapse>
          <n-collapse-item title="如何购买套餐" name="buy">
            <p>登录后前往「购买套餐」页面，选择适合您的套餐并完成支付即可。支持余额支付和在线支付方式。</p>
          </n-collapse-item>
          <n-collapse-item title="如何使用订阅链接" name="subscribe">
            <p>购买套餐后，前往「我的订阅」页面复制订阅链接，然后将链接导入到您使用的客户端中即可。不同客户端的导入方式略有不同，请参考下方客户端说明。</p>
          </n-collapse-item>
          <n-collapse-item title="支持哪些客户端" name="clients">
            <p><strong>推荐使用本站自研客户端 Mclash</strong> —— 官方开发、界面简洁、方便好用，登录后自动同步订阅，无需手动复制链接。支持 Windows、Android、macOS（含 Apple 芯片 / Intel 芯片）。</p>
            <p>同时也兼容主流第三方客户端：Clash Verge、Clash for Windows、V2rayN、Clash Party、Hiddify、FlClash、Shadowrocket、Stash 等，请参考下方软件下载区域选择对应平台。</p>
          </n-collapse-item>
          <n-collapse-item title="如何重置订阅" name="reset">
            <p>前往「我的订阅」页面，点击「重置订阅链接」按钮即可生成新的订阅链接。重置后旧链接将失效，请及时更新客户端中的订阅地址。</p>
          </n-collapse-item>
          <n-collapse-item title="设备限制说明" name="device">
            <p>每个套餐有对应的设备数量限制。同一订阅链接在不同设备上使用会占用设备名额。如需释放设备名额，请前往「设备管理」页面删除不再使用的设备。</p>
          </n-collapse-item>
          <n-collapse-item title="如何联系客服" name="contact">
            <p>您可以通过提交工单的方式联系客服，前往「工单」页面创建新工单即可。我们会尽快回复您的问题。</p>
          </n-collapse-item>
        </n-collapse>
      </n-card>

      <n-card id="sec-tutorial" title="使用教程" :bordered="false">
        <!-- Desktop: segment tabs -->
        <n-tabs v-if="!appStore.isMobile" type="segment" size="small" animated>
          <n-tab-pane v-for="t in tutorials" :key="t.key" :name="t.key" :tab="t.tab">
            <div class="tutorial">
              <h4>{{ t.title }}</h4>
              <p v-if="t.note" class="tut-note">{{ t.note }}</p>
              <template v-for="(section, si) in t.sections" :key="si">
                <h5 v-if="section.subtitle">{{ section.subtitle }}</h5>
                <ol><li v-for="(step, i) in section.steps" :key="i" v-html="step" /></ol>
              </template>
              <p v-if="t.tip" class="tut-tip" v-html="t.tip" />
            </div>
          </n-tab-pane>
        </n-tabs>

        <!-- Mobile: card list with collapse -->
        <div v-else class="tut-card-list">
          <div v-for="t in tutorials" :key="t.key" class="tut-card" @click="toggleTut(t.key)">
            <div class="tut-card-header">
              <span class="tut-card-icon">{{ t.icon }}</span>
              <div class="tut-card-meta">
                <span class="tut-card-name">{{ t.tab }}</span>
                <span class="tut-card-platform">{{ t.platform }}</span>
              </div>
              <n-icon :component="expandedTut === t.key ? ChevronUpOutline : ChevronDownOutline" size="18" color="#999" />
            </div>
            <div v-if="expandedTut === t.key" class="tut-card-body" @click.stop>
              <p v-if="t.note" class="tut-note">{{ t.note }}</p>
              <template v-for="(section, si) in t.sections" :key="si">
                <div v-if="section.subtitle" class="tut-subtitle">{{ section.subtitle }}</div>
                <ol><li v-for="(step, i) in section.steps" :key="i" v-html="step" /></ol>
              </template>
              <p v-if="t.tip" class="tut-tip" v-html="t.tip" />
            </div>
          </div>
        </div>
      </n-card>

      <n-card id="sec-download" title="软件下载" :bordered="false">
        <n-spin :show="loadingConfig">
          <!-- 手机端：平台 tabs（n-tabs 会把 tab 条和面板一起渲染，套吸顶条会把
               整块面板都变成工具条，所以这里只做样式对齐，吸顶交给上方板块导航条） -->
          <div v-if="hasAnyClient" class="help-download">
            <n-tabs type="segment" :size="appStore.isMobile ? 'medium' : 'small'" animated>
              <n-tab-pane name="windows" tab="Windows" v-if="windowsClients.length">
                <div class="client-grid">
                  <button v-for="c in windowsClients" :key="c.key" class="client-card" :class="{ 'client-card-self': c.self }" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
                        <span v-if="c.self" class="client-badge-self">自研推荐</span>
                        <span v-if="c.chip" class="client-chip" :class="c.chip === 'Apple 芯片' ? 'chip-arm' : 'chip-intel'">{{ c.chip }}</span>
                      </span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-spin v-if="downloadingKey === c.key" size="small" />
                    <n-icon v-else :component="DownloadOutline" size="18" color="var(--primary-color)" />
                  </button>
                </div>
              </n-tab-pane>
              <n-tab-pane name="android" tab="Android" v-if="androidClients.length">
                <div class="client-grid">
                  <button v-for="c in androidClients" :key="c.key" class="client-card" :class="{ 'client-card-self': c.self }" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
                        <span v-if="c.self" class="client-badge-self">自研推荐</span>
                        <span v-if="c.chip" class="client-chip" :class="c.chip === 'Apple 芯片' ? 'chip-arm' : 'chip-intel'">{{ c.chip }}</span>
                      </span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-spin v-if="downloadingKey === c.key" size="small" />
                    <n-icon v-else :component="DownloadOutline" size="18" color="var(--primary-color)" />
                  </button>
                </div>
              </n-tab-pane>
              <n-tab-pane name="macos" tab="macOS" v-if="macClients.length">
                <div class="client-grid">
                  <button v-for="c in macClients" :key="c.key" class="client-card" :class="{ 'client-card-self': c.self }" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
                        <span v-if="c.self" class="client-badge-self">自研推荐</span>
                        <span v-if="c.chip" class="client-chip" :class="c.chip === 'Apple 芯片' ? 'chip-arm' : 'chip-intel'">{{ c.chip }}</span>
                      </span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-spin v-if="downloadingKey === c.key" size="small" />
                    <n-icon v-else :component="DownloadOutline" size="18" color="var(--primary-color)" />
                  </button>
                </div>
              </n-tab-pane>
              <n-tab-pane name="ios" tab="iOS" v-if="iosClients.length">
                <div class="client-grid">
                  <button v-for="c in iosClients" :key="c.key" class="client-card" :class="{ 'client-card-self': c.self }" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
                        <span v-if="c.self" class="client-badge-self">自研推荐</span>
                        <span v-if="c.chip" class="client-chip" :class="c.chip === 'Apple 芯片' ? 'chip-arm' : 'chip-intel'">{{ c.chip }}</span>
                      </span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-spin v-if="downloadingKey === c.key" size="small" />
                    <n-icon v-else :component="DownloadOutline" size="18" color="var(--primary-color)" />
                  </button>
                </div>
              </n-tab-pane>
            </n-tabs>
          </div>
          <n-empty v-else-if="!loadingConfig" description="管理员暂未配置下载链接" />
        </n-spin>
      </n-card>

      <n-card id="sec-contact" title="联系我们" :bordered="false">
        <n-space vertical :size="12">
          <n-text>如果您在使用过程中遇到任何问题，可以通过以下方式联系我们：</n-text>
          <div v-if="supportItems.length" class="contact-list">
            <a
              v-for="item in supportItems"
              :key="item.key"
              class="contact-item"
              :href="item.href"
              target="_blank"
              rel="noopener"
            >
              <n-icon :component="item.icon" :size="18" class="contact-icon" />
              <span class="contact-label">{{ item.label }}</span>
              <span class="contact-value">{{ item.display }}</span>
            </a>
          </div>
          <n-text v-else depth="3">客服联系方式暂未配置，请向管理员索取。</n-text>
          <n-text>提交工单：前往「工单」页面创建新工单，我们会尽快回复您。</n-text>
          <n-text depth="3">工作时间：周一至周五 9:00 - 18:00，工单通常在 24 小时内回复。</n-text>
          <n-divider style="margin: 4px 0 8px;" />
          <n-space :size="12" wrap align="center">
            <n-text depth="3" style="font-size: 13px;">相关协议：</n-text>
            <n-button text type="primary" size="small" @click="$router.push('/terms')">《服务条款》</n-button>
            <n-button text type="primary" size="small" @click="$router.push('/privacy')">《隐私政策》</n-button>
          </n-space>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, onUnmounted, onDeactivated } from 'vue'
import { DownloadOutline, ChevronDownOutline, ChevronUpOutline, MailOutline, ChatbubblesOutline, SendOutline } from '@vicons/ionicons5'
import { getPublicConfig } from '@/api/common'
import { resolvePanDownloadUrl } from '@/utils/githubDownload'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const loadingConfig = ref(false)
const config = ref<Record<string, string>>({})
const expandedTut = ref('')

const toggleTut = (key: string) => {
  expandedTut.value = expandedTut.value === key ? '' : key
}

/* ---------------- 手机端板块导航（吸顶） ----------------
   为什么不用纯 CSS：全局 .app-sticky-toolbar 是 position: sticky，但内容被
   .n-scrollbar-container（overflow: scroll）包着，而真正滚动的是 window，
   该容器自身不滚动 —— 实测滚动 600px 后 sticky 元素 top 变成 -588，照样滚走。
   公共文件不可改，所以这里用等效实现：滚出视口顶部就切 fixed（.is-stuck），
   未吸顶时留在文档流里（不会挡住顶部内容），滚动时给插槽补上等高占位避免跳动。 */
const pageSections = [
  { id: 'sec-faq', label: '常见问题' },
  { id: 'sec-tutorial', label: '使用教程' },
  { id: 'sec-download', label: '软件下载' },
  { id: 'sec-contact', label: '联系我们' },
]

const navSlot = ref<HTMLElement | null>(null)
const navStuck = ref(false)
const navHeight = ref(56)
const activeSection = ref(pageSections[0].id)

function jumpTo(id: string) {
  activeSection.value = id
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

let scrollBound = false
let navRaf = 0
function measureNav() {
  navRaf = 0
  const slot = navSlot.value
  if (!slot || !appStore.isMobile) {
    navStuck.value = false
    return
  }
  const bar = slot.firstElementChild as HTMLElement | null
  const h = bar ? Math.round(bar.getBoundingClientRect().height) : 0
  if (h > 0) navHeight.value = h
  navStuck.value = slot.getBoundingClientRect().top < 0
  // 当前板块 = 视口上沿往下 72px 处所在的那个卡片（让导航条有 App 的选中态）
  let current = pageSections[0].id
  for (const s of pageSections) {
    const el = document.getElementById(s.id)
    if (el && el.getBoundingClientRect().top <= 72) current = s.id
  }
  // 滚到底时高亮最后一个板块（末节通常滚不到顶部，否则选中态会停在上一节）
  if (window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 4) {
    current = pageSections[pageSections.length - 1].id
  }
  activeSection.value = current
}

function onPageScroll() {
  if (!navRaf) navRaf = requestAnimationFrame(measureNav)
}

function bindScroll() {
  if (scrollBound) return
  scrollBound = true
  window.addEventListener('scroll', onPageScroll, { passive: true })
  window.addEventListener('resize', onPageScroll, { passive: true })
  measureNav()
}

function unbindScroll() {
  if (!scrollBound) return
  scrollBound = false
  window.removeEventListener('scroll', onPageScroll)
  window.removeEventListener('resize', onPageScroll)
  if (navRaf) {
    cancelAnimationFrame(navRaf)
    navRaf = 0
  }
}

interface SupportItem {
  key: string
  label: string
  display: string
  href: string
  icon: any
}

function buildTelegramUrl(tg: string) {
  const value = tg.trim()
  if (/^https?:\/\//i.test(value)) return value
  return `https://t.me/${value.replace(/^@/, '')}`
}

const supportItems = computed<SupportItem[]>(() => {
  const items: SupportItem[] = []
  const email = (config.value.support_email || '').trim()
  const qq = (config.value.support_qq || '').trim()
  const telegram = (config.value.support_telegram || '').trim()
  if (email) items.push({ key: 'email', label: '邮箱', display: email, href: `mailto:${email}`, icon: MailOutline })
  if (qq) items.push({ key: 'qq', label: 'QQ 群', display: qq, href: `tencent://message/?uin=${encodeURIComponent(qq)}`, icon: ChatbubblesOutline })
  if (telegram) items.push({ key: 'telegram', label: 'Telegram', display: `@${telegram.replace(/^@/, '')}`, href: buildTelegramUrl(telegram), icon: SendOutline })
  return items
})

interface TutSection { subtitle?: string; steps: string[] }
interface Tutorial {
  key: string; tab: string; icon: string; platform: string; title: string;
  note?: string; tip?: string; sections: TutSection[]
}

const tutorials: Tutorial[] = [
  {
    key: 'mclash', tab: 'Mclash（推荐）', icon: '🚀', platform: '全平台',
    title: 'Mclash 自研客户端使用教程',
    note: 'Mclash 是本站官方自研客户端，界面简洁、方便好用，登录后订阅自动同步，无需手动复制链接。推荐优先使用。',
    sections: [
      { subtitle: '第一步：下载安装', steps: [
        '在上方「软件下载」区域选择 <strong>Mclash</strong>，按你的平台下载（Windows / Android / macOS）。',
        'macOS 用户请注意区分：<strong>Apple 芯片</strong>（M1/M2/M3/M4）与 <strong>Intel 芯片</strong>，选错会无法安装。',
        '下载完成后安装并打开 Mclash。',
      ] },
      { subtitle: '第二步：登录账号（自动同步订阅）', steps: [
        '在 Mclash 登录页输入你<strong>本站的邮箱或用户名</strong>与密码，与网页端同一套账号。',
        '登录成功后，Mclash 会<strong>自动拉取你的订阅</strong>并更新节点，无需手动导入。',
        '若提示暂无订阅，请先在网页端「购买套餐」完成订阅。',
      ] },
      { subtitle: '第三步：选择节点并连接', steps: [
        '在节点列表中选择一个节点（推荐延迟较低的）。',
        '打开主页的连接开关即可开始使用。',
        '后续订阅到期时间、节点更新都会自动同步，无需手动操作。',
      ] },
    ],
    tip: '提示：登录后本设备会自动登记到你的账号下，可在「我的设备」查看。同一台设备升级软件版本不会重复计算设备数。',
  },
  {
    key: 'shadowrocket', tab: 'Shadowrocket', icon: '🚀', platform: 'iOS',
    title: 'Shadowrocket 使用教程',
    note: 'Shadowrocket 需要使用非中国大陆 Apple ID 在 App Store 购买下载（售价约 $2.99）。',
    sections: [{ steps: [
      '打开本站「我的订阅」页面，复制<strong>通用订阅链接</strong>。',
      '打开 Shadowrocket，点击右上角 <strong>+</strong> 按钮。',
      '类型选择 <strong>Subscribe</strong>（订阅）。',
      '在 URL 栏粘贴刚才复制的订阅链接，备注可填写任意名称。',
      '点击右上角<strong>完成</strong>保存。',
      '回到首页，点击订阅右侧的刷新按钮更新节点列表。',
      '选择一个节点，打开顶部的<strong>连接开关</strong>即可使用。',
    ] }],
    tip: '提示：也可以直接在「我的订阅」页面点击 Shadowrocket 二维码，用 Shadowrocket 扫码一键导入。',
  },
  {
    key: 'clash', tab: 'Clash', icon: '🔵', platform: 'Windows',
    title: 'Clash for Windows 使用教程',
    sections: [{ steps: [
      '下载并安装 Clash for Windows（见下方下载区域）。',
      '打开本站「我的订阅」页面，复制 <strong>Clash 订阅链接</strong>。',
      '打开 Clash for Windows，点击左侧 <strong>Profiles</strong>（配置）。',
      '在顶部输入框粘贴 Clash 订阅链接，点击 <strong>Download</strong>。',
      '下载完成后，点击该配置文件使其高亮选中。',
      '切换到 <strong>Proxies</strong>（代理）页面，选择一个节点。',
      '切换到 <strong>General</strong>（常规）页面，打开 <strong>System Proxy</strong>（系统代理）。',
    ] }],
    tip: '提示：建议开启 Clash 的「开机自启」功能，避免每次手动启动。',
  },
  {
    key: 'v2rayn', tab: 'V2rayN', icon: '🟢', platform: 'Windows',
    title: 'V2rayN 使用教程',
    sections: [{ steps: [
      '下载并解压 V2rayN（见下方下载区域）。',
      '打开本站「我的订阅」页面，复制<strong>通用订阅链接</strong>。',
      '运行 V2rayN，右键系统托盘图标 → <strong>订阅分组设置</strong>。',
      '点击<strong>添加</strong>，在地址栏粘贴订阅链接，备注填写任意名称，点击确定。',
      '右键托盘图标 → <strong>更新订阅</strong>（不通过代理）。',
      '在主界面选择一个节点，右键 → <strong>设为活动服务器</strong>。',
      '右键托盘图标 → 系统代理 → <strong>自动配置系统代理</strong>。',
    ] }],
  },
  {
    key: 'android', tab: 'Android', icon: '🤖', platform: 'Android',
    title: 'V2rayNG / Clash Meta 使用教程',
    sections: [
      { subtitle: 'V2rayNG', steps: [
        '下载并安装 V2rayNG（见下方下载区域）。',
        '打开本站「我的订阅」页面，复制<strong>通用订阅链接</strong>。',
        '打开 V2rayNG，点击左上角菜单 → <strong>订阅分组设置</strong>。',
        '点击右上角 <strong>+</strong>，在地址栏粘贴订阅链接，点击右上角 ✓ 保存。',
        '返回主界面，点击右上角菜单 → <strong>更新订阅</strong>。',
        '选择一个节点，点击右下角 <strong>V</strong> 按钮连接。',
      ] },
      { subtitle: 'Clash Meta for Android', steps: [
        '下载并安装 Clash Meta（见下方下载区域）。',
        '打开本站「我的订阅」页面，复制 <strong>Clash 订阅链接</strong>。',
        '打开 Clash Meta，点击 <strong>Profile</strong>（配置）。',
        '点击右上角 <strong>+</strong> → <strong>URL</strong>，粘贴 Clash 订阅链接，点击保存。',
        '选中该配置，返回主页点击<strong>启动</strong>按钮。',
      ] },
    ],
  },
  {
    key: 'stash', tab: 'Stash', icon: '🟡', platform: 'iOS',
    title: 'Stash 使用教程',
    note: 'Stash 需要使用非中国大陆 Apple ID 在 App Store 购买下载。',
    sections: [{ steps: [
      '打开本站「我的订阅」页面，复制 <strong>Clash 订阅链接</strong>。',
      '打开 Stash，进入<strong>设置</strong> → <strong>配置</strong> → <strong>从 URL 下载</strong>。',
      '粘贴 Clash 订阅链接，点击<strong>下载</strong>。',
      '下载完成后选中该配置文件。',
      '返回首页，选择节点并打开连接开关。',
    ] }],
  },
  {
    key: 'clashparty', tab: 'Clash Party', icon: '🟣', platform: 'Win / macOS',
    title: 'Clash Party 使用教程',
    sections: [{ steps: [
      '下载并安装 Clash Party（见下方下载区域）。',
      '打开本站「我的订阅」页面，复制 <strong>Clash 订阅链接</strong>。',
      '打开 Clash Party，进入<strong>订阅管理</strong>。',
      '点击<strong>导入</strong>，粘贴 Clash 订阅链接，确认导入。',
      '选中导入的配置，返回主页。',
      '选择节点，开启<strong>系统代理</strong>即可使用。',
    ] }],
  },
  {
    key: 'flclash', tab: 'FlClash', icon: '⚡', platform: '全平台',
    title: 'FlClash 使用教程',
    sections: [{ steps: [
      '下载并安装 FlClash（见下方下载区域）。',
      '打开本站「我的订阅」页面，复制 <strong>Clash 订阅链接</strong>。',
      '打开 FlClash，进入<strong>配置</strong>页面。',
      '点击 <strong>+</strong> 添加配置 → 选择 <strong>URL 导入</strong>。',
      '粘贴 Clash 订阅链接，点击确认。',
      '选中配置后，在<strong>代理</strong>页面选择节点。',
      '开启<strong>系统代理</strong>或 <strong>TUN 模式</strong>即可使用。',
    ] }],
  },
  {
    key: 'hiddify', tab: 'Hiddify', icon: '🟠', platform: 'Win / Android',
    title: 'Hiddify 使用教程',
    sections: [{ steps: [
      '下载并安装 Hiddify（见下方下载区域）。',
      '打开本站「我的订阅」页面，复制<strong>通用订阅链接</strong>。',
      '打开 Hiddify，点击 <strong>+</strong> 添加配置。',
      '选择<strong>从链接添加</strong>，粘贴订阅链接。',
      '等待节点加载完成，选择一个节点。',
      '点击底部的<strong>连接</strong>按钮即可使用。',
    ] }],
  },
]

interface ClientItem {
  key: string
  name: string
  icon: string
  desc: string
  clientKey?: string
  armKey?: string
  chip?: string
  /** 自研客户端：用户端会打上「自研推荐」标签并高亮显示 */
  self?: boolean
}

const allClients: Record<'windows' | 'android' | 'macos' | 'ios', ClientItem[]> = {
  windows: [
    { key: 'client_mclash_windows_url', name: 'Mclash', icon: '🚀', self: true, desc: '官方自研客户端，方便好用，登录即自动同步订阅' },
    { key: 'client_clash_windows_url', name: 'Clash for Windows', icon: '🔵', desc: 'Clash 内核，支持多种协议' },
    { key: 'client_v2rayn_url', name: 'V2rayN', clientKey: 'v2rayN', icon: '🟢', desc: 'V2Ray 图形化客户端' },
    { key: 'client_clashparty_windows_url', name: 'Clash Party', clientKey: 'clash-party', icon: '🟣', desc: 'Clash Party GUI 客户端' },
    { key: 'client_clashverge_windows_url', name: 'Clash Verge', clientKey: 'clash-verge', icon: '🟣', desc: 'Clash Verge GUI 客户端' },
    { key: 'client_hiddify_windows_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', desc: '多协议代理客户端' },
    { key: 'client_flclash_windows_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', desc: 'Flutter 跨平台 Clash 客户端' },
  ],
  android: [
    { key: 'client_mclash_android_url', name: 'Mclash', icon: '🚀', self: true, desc: '官方自研客户端，方便好用，登录即自动同步订阅' },
    { key: 'client_clash_android_url', name: 'Clash Meta', clientKey: 'clash-meta', icon: '🔵', desc: 'Android Clash 客户端' },
    { key: 'client_v2rayng_url', name: 'V2rayNG', clientKey: 'v2rayNG', icon: '🟢', desc: 'Android V2Ray 客户端' },
    { key: 'client_hiddify_android_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', desc: 'Android 多协议客户端' },
    { key: 'client_flclash_android_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', desc: 'Android FlClash 客户端' },
  ],
  macos: [
    { key: 'client_mclash_macos_url', armKey: 'client_mclash_macos_arm_url', name: 'Mclash', icon: '🚀', self: true, desc: '官方自研客户端，方便好用，登录即自动同步订阅' },
    { key: 'client_flclash_macos_url', armKey: 'client_flclash_macos_arm_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', desc: 'macOS Clash 客户端' },
    { key: 'client_clashparty_macos_url', armKey: 'client_clashparty_macos_arm_url', name: 'Clash Party', clientKey: 'clash-party', icon: '🟣', desc: 'macOS Clash Party 客户端' },
    { key: 'client_clashverge_macos_url', armKey: 'client_clashverge_macos_arm_url', name: 'Clash Verge', clientKey: 'clash-verge', icon: '🟣', desc: 'macOS Clash Verge 客户端' },
    { key: 'client_v2rayn_macos_url', armKey: 'client_v2rayn_macos_arm_url', name: 'V2rayN', clientKey: 'v2rayN', icon: '🟢', desc: 'macOS V2Ray 客户端' },
    { key: 'client_hiddify_macos_url', armKey: 'client_hiddify_macos_arm_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', desc: 'macOS 多协议客户端' },
  ],
  ios: [
    { key: 'client_shadowrocket_url', name: 'Shadowrocket', icon: '🚀', desc: '需外区 Apple ID 购买' },
    { key: 'client_stash_url', name: 'Stash', icon: '🟡', desc: '基于规则的代理客户端' },
  ],
}

// 显示规则：配置了 URL 或配置了 clientKey（GitHub 自动解析）的客户端都显示；
// macOS 配置了 armKey 的拆分为 Intel / Apple 芯片两个下载选项。
const filterClients = (list: typeof allClients.windows) => {
  const out: any[] = []
  list.forEach((c: any) => {
    const url = config.value[c.key] || ''
    const armUrl = c.armKey ? (config.value[c.armKey] || '') : ''
    const showIntel = url || c.clientKey
    const showArm = c.armKey && (armUrl || c.clientKey)
    if (showIntel) {
      out.push({
        ...c, url,
        auto: !url || String(url).startsWith('pan://'),
        chip: c.armKey ? 'Intel' : '',
        forcedArch: c.armKey ? 'intel' : null,
      })
    }
    if (showArm) {
      out.push({
        ...c, key: c.armKey, url: armUrl,
        auto: !armUrl || String(armUrl).startsWith('pan://'),
        chip: 'Apple 芯片',
        forcedArch: 'apple',
      })
    }
  })
  return out
}

const windowsClients = computed(() => filterClients(allClients.windows))
const androidClients = computed(() => filterClients(allClients.android))
const macClients = computed(() => filterClients(allClients.macos))
const iosClients = computed(() => filterClients(allClients.ios))
const hasAnyClient = computed(() =>
  Object.values(allClients).flat().some(c => config.value[c.key] || (c as any).clientKey)
)

// 自动客户端点击：动态解析 GitHub 最新版直链
const downloadingKey = ref('')
async function handleClientClick(c: any) {
  if (downloadingKey.value) return
  // 无配置或 pan:// 标记：交给后端 /download/gh 解析（VPS 查 GitHub 最新版 + 加速镜像）
  const needAuto = !c.url || String(c.url).startsWith('pan://')
  if (needAuto) {
    downloadingKey.value = c.key
    try {
      window.open(`/api/v1/download/gh?key=${encodeURIComponent(c.key)}`, '_blank')
    } finally {
      downloadingKey.value = ''
    }
    return
  }
  if (c.url) window.open(resolvePanDownloadUrl(c.url), '_blank')
}
onMounted(async () => {
  loadingConfig.value = true
  try {
    const res: any = await getPublicConfig()
    if (res.data) config.value = res.data
  } catch {}
  finally { loadingConfig.value = false }
})

// 手机端吸顶导航需要监听 window 滚动；keep-alive 缓存的页面在重新激活时重新绑定
onMounted(bindScroll)
onActivated(bindScroll)
onDeactivated(unbindScroll)
onUnmounted(unbindScroll)
</script>

<style scoped>
/* ============================================================
   帮助中心 —— 手机端按 docs/mobile-app-spec.md 第 1 / 3 节改造：
   · 根容器手机端不再自带左右内边距（全局统一 10px，原来叠成 36px 白边）
   · 教程卡 / 客户端卡：16px 圆角 + 纯色底 + 轻阴影（去掉渐变与重阴影）
   · 正文 ≥13px、说明文字 ≥12px（清掉 11px 小字）
   · 板块导航条与下载平台 tabs 用 .app-sticky-toolbar 吸顶
   ============================================================ */

.help-page {
  padding: 24px;
}

.title {
  font-size: 28px;
  font-weight: 600;
  margin: 0;
  background: var(--brand-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

/* ---------------- 手机端板块导航条 ---------------- */
.help-nav-slot {
  display: block;
}

.help-nav-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  /* 全局 .app-sticky-toolbar 的 -12px 负边距是给「页面根容器自带 12px 内边距」的
     页面用的；本页根容器手机端无左右内边距，负边距会把导航条撑出屏幕，这里归零。
     边框左右留 1px 透明边，保证吸顶前后高度一致（不跳动）。 */
  margin: 0;
  padding: 8px 10px;
  border: 1px solid transparent;
}
.help-nav-bar::-webkit-scrollbar {
  display: none;
}

/* 吸顶态：sticky 在本布局被 .n-scrollbar-container（overflow: scroll 且不滚动）吃掉，
   所以滚出视口顶部后改用 fixed 顶到屏幕最上方（未吸顶时仍在文档流里，不遮挡内容）。 */
.help-nav-bar.is-stuck {
  position: fixed;
  top: 0;
  left: 10px;
  right: 10px;
  z-index: 30;
  padding: 8px 10px;
  border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 70%, transparent);
  border-radius: 0 0 16px 16px;
}

.help-nav-chip {
  flex: 0 0 auto;
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
.help-nav-chip.is-active {
  background: var(--primary-color, #4f46e5);
  border-color: transparent;
  color: #fff;
  font-weight: 600;
}

/* ---------------- 软件下载：客户端卡片 ---------------- */
.client-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 0;
}

/* 下载平台 tabs：手机端加高到 40px 可点区域 */
.help-download {
  padding-bottom: 4px;
}

.client-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  min-height: 60px;
  border-radius: 16px;
  /* App 卡片：纯色底 + 轻阴影（原来是 10px 圆角 + 灰色 rgba 底） */
  background: var(--bg-color, #fff);
  border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 70%, transparent);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 16px rgba(15, 23, 42, 0.04);
  cursor: pointer;
  transition: transform 0.12s ease, border-color 0.12s ease, background-color 0.12s ease;
  text-decoration: none;
  color: inherit;
  /* button 元素默认样式重置（上面已给真实边框） */
  font-family: inherit;
  text-align: left;
  width: 100%;
  box-sizing: border-box;
}
.client-card:active {
  transform: scale(0.99);
}
@media (hover: hover) {
  .client-card:hover {
    border-color: color-mix(in srgb, var(--primary-color, #4f46e5) 35%, transparent);
  }
}

/* 自研客户端：用主色描边 + 浅色底突出（原来是渐变底，手机上看像网页） */
.client-card-self {
  background: var(--primary-color-soft, rgba(79, 70, 229, 0.08));
  border-color: color-mix(in srgb, var(--primary-color, #4f46e5) 45%, transparent);
}

.client-badge-self {
  display: inline-block;
  margin-left: 6px;
  padding: 2px 7px;
  /* 12px 起（原来是 11px，手机端体检报小字） */
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  border-radius: 6px;
  color: #fff;
  background: var(--primary-color);
  vertical-align: middle;
}

/* 架构标签（Intel / Apple 芯片）：原来本页没有样式，是纯文本 */
.client-chip {
  display: inline-block;
  margin-left: 6px;
  padding: 2px 7px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  border-radius: 6px;
  white-space: nowrap;
  flex-shrink: 0;
}
.chip-arm {
  background: var(--primary-color-soft, rgba(102, 126, 234, 0.15));
  color: var(--primary-color, #4f46e5);
}
.chip-intel {
  background: rgba(52, 199, 89, 0.15);
  color: #1b8a3f;
}

.client-icon { font-size: 24px; flex-shrink: 0; }
.client-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.client-name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
}
.client-desc { font-size: 13px; line-height: 1.5; color: var(--text-color-secondary); }

/* Contact */
.contact-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.contact-item {
  display: flex;
  align-items: center;
  gap: 8px;
  /* 触控目标 ≥40px（a 元素，手机端手指点得准） */
  min-height: 44px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--primary-color-soft, rgba(102, 126, 234, 0.06));
  text-decoration: none;
  color: inherit;
  transition: background 0.2s;
}
.contact-item:hover {
  background: var(--primary-color-soft, rgba(102, 126, 234, 0.12));
}
.contact-icon {
  color: var(--primary-color);
  flex-shrink: 0;
}
.contact-label {
  font-size: 13px;
  color: var(--text-color-secondary);
  flex-shrink: 0;
}
.contact-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--primary-color);
  min-width: 0;
  word-break: break-all;
}

.tutorial { padding: 12px 0; }
.tutorial h4 { margin: 0 0 12px; font-size: 16px; font-weight: 600; }
.tutorial h5 { margin: 16px 0 8px; font-size: 14px; font-weight: 600; color: var(--text-color); }
.tutorial ol { padding-left: 20px; margin: 0; }
.tutorial li { margin: 6px 0; font-size: 14px; line-height: 1.7; color: var(--text-color); }
.tutorial li strong { color: var(--text-color); }
.tut-note { background: color-mix(in srgb, #d97706 8%, var(--bg-color, #fff)); border: 1px solid color-mix(in srgb, #d97706 30%, transparent); border-radius: 12px; padding: 10px 12px; font-size: 13px; line-height: 1.6; color: #a15c07; margin-bottom: 12px; }
.tut-tip { background: color-mix(in srgb, #059669 8%, var(--bg-color, #fff)); border: 1px solid color-mix(in srgb, #059669 30%, transparent); border-radius: 12px; padding: 10px 12px; font-size: 13px; line-height: 1.6; color: #0f7a4d; margin-top: 12px; }

/* Mobile tutorial cards（保留本页类名，观感对齐全局 .mobile-card：
   16px 圆角 / 纯色底 / 轻阴影） */
.tut-card-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tut-card {
  border-radius: 16px;
  background: var(--bg-color, #fff);
  border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 60%, transparent);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 16px rgba(15, 23, 42, 0.03);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.12s ease, background-color 0.12s ease;
}
.tut-card:active {
  transform: scale(0.99);
}

.tut-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 56px;
  padding: 12px 14px;
}

.tut-card-icon {
  font-size: 22px;
  flex-shrink: 0;
}

.tut-card-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tut-card-name {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--text-color, #333);
}

.tut-card-platform {
  /* 12px 起（原来是 11px，手机端体检报小字） */
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-color-secondary);
}

.tut-card-body {
  padding: 0 14px 14px;
  cursor: default;
}

.tut-card-body ol {
  padding-left: 18px;
  margin: 0;
}

.tut-card-body li {
  margin: 6px 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-color);
}

.tut-subtitle {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  margin: 12px 0 4px;
}

.tut-card-body .tut-note,
.tut-card-body .tut-tip {
  font-size: 12px;
  line-height: 1.6;
  padding: 8px 10px;
}

@media (max-width: 767px) {
  /* 契约 §1：手机端根容器不再自带左右内边距，只保留纵向留白。
     （全局 user-mobile.css 用 !important 把 .help-page 的 padding 全清零了，
     这里用标题外边距补回顶部呼吸感，不动公共文件。） */
  .help-page { padding: 12px 0 0; }
  .title { font-size: 22px; margin-top: 8px; }

  /* 手机端：卡片内容贴齐卡片（去掉 .client-grid 的纵向大间距） */
  .client-grid { padding: 4px 0 0; }
  .client-card { padding: 12px; gap: 10px; }

  /* 下载平台 tabs 的可点区域撑到 40px */
  .help-download :deep(.n-tabs-tab) {
    min-height: 40px;
  }

  /* FAQ 手风琴行：App 里一行至少 48px，原来只有 38px，点不准 */
  .help-page :deep(.n-collapse-item__header) {
    min-height: 48px;
    padding-top: 0;
    padding-bottom: 0;
  }
  .help-page :deep(.n-collapse-item__header__title) {
    font-size: 15px;
    font-weight: 600;
  }
  .help-page :deep(.n-collapse-item__content-inner) {
    font-size: 13px;
    line-height: 1.7;
  }
}
</style>