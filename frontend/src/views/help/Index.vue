<template>
  <div class="help-page">
    <n-space vertical :size="24">
      <h1 class="title">帮助中心</h1>

      <n-card title="常见问题" :bordered="false">
        <n-collapse>
          <n-collapse-item title="如何购买套餐" name="buy">
            <p>登录后前往「购买套餐」页面，选择适合您的套餐并完成支付即可。支持余额支付和在线支付方式。</p>
          </n-collapse-item>
          <n-collapse-item title="如何使用订阅链接" name="subscribe">
            <p>购买套餐后，前往「我的订阅」页面复制订阅链接，然后将链接导入到您使用的客户端中即可。不同客户端的导入方式略有不同，请参考下方客户端说明。</p>
          </n-collapse-item>
          <n-collapse-item title="支持哪些客户端" name="clients">
            <p>我们支持主流的代理客户端，包括 Clash for Windows、V2rayN、Clash Party、Hiddify、FlClash、Shadowrocket、Stash 等。请参考下方客户端下载区域获取对应平台的客户端。</p>
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

      <n-card title="使用教程" :bordered="false">
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

      <n-card title="软件下载" :bordered="false">
        <n-spin :show="loadingConfig">
          <div v-if="hasAnyClient">
            <n-tabs type="segment" size="small" animated>
              <n-tab-pane name="windows" tab="Windows" v-if="windowsClients.length">
                <div class="client-grid">
                  <button v-for="c in windowsClients" :key="c.key" class="client-card" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
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
                  <button v-for="c in androidClients" :key="c.key" class="client-card" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
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
                  <button v-for="c in macClients" :key="c.key" class="client-card" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
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
                  <button v-for="c in iosClients" :key="c.key" class="client-card" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
                        <span v-if="c.chip" class="client-chip" :class="c.chip === 'Apple 芯片' ? 'chip-arm' : 'chip-intel'">{{ c.chip }}</span>
                      </span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-spin v-if="downloadingKey === c.key" size="small" />
                    <n-icon v-else :component="DownloadOutline" size="18" color="var(--primary-color)" />
                  </button>
                </div>
              </n-tab-pane>
              <n-tab-pane name="linux" tab="Linux" v-if="linuxClients.length">
                <div class="client-grid">
                  <button v-for="c in linuxClients" :key="c.key" class="client-card" type="button" @click="handleClientClick(c)">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">
                        {{ c.name }}
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

      <n-card title="联系我们" :bordered="false">
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
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { DownloadOutline, ChevronDownOutline, ChevronUpOutline, MailOutline, ChatbubblesOutline, SendOutline } from '@vicons/ionicons5'
import { getPublicConfig } from '@/api/common'
import { resolvePanDownloadUrl } from '@/utils/githubDownload'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const message = useMessage()
const loadingConfig = ref(false)
const config = ref<Record<string, string>>({})
const expandedTut = ref('')

const toggleTut = (key: string) => {
  expandedTut.value = expandedTut.value === key ? '' : key
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

const allClients = {
  windows: [
    { key: 'client_clash_windows_url', name: 'Clash for Windows', icon: '🔵', desc: 'Clash 内核，支持多种协议' },
    { key: 'client_v2rayn_url', name: 'V2rayN', clientKey: 'v2rayN', icon: '🟢', desc: 'V2Ray 图形化客户端' },
    { key: 'client_clashparty_windows_url', name: 'Clash Party', clientKey: 'clash-party', icon: '🟣', desc: 'Clash Party GUI 客户端' },
    { key: 'client_clashverge_windows_url', name: 'Clash Verge', clientKey: 'clash-verge', icon: '🟣', desc: 'Clash Verge GUI 客户端' },
    { key: 'client_hiddify_windows_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', desc: '多协议代理客户端' },
    { key: 'client_flclash_windows_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', desc: 'Flutter 跨平台 Clash 客户端' },
  ],
  android: [
    { key: 'client_clash_android_url', name: 'Clash Meta', clientKey: 'clash-meta', icon: '🔵', desc: 'Android Clash 客户端' },
    { key: 'client_v2rayng_url', name: 'V2rayNG', clientKey: 'v2rayNG', icon: '🟢', desc: 'Android V2Ray 客户端' },
    { key: 'client_hiddify_android_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', desc: 'Android 多协议客户端' },
    { key: 'client_flclash_android_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', desc: 'Android FlClash 客户端' },
  ],
  macos: [
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
  linux: [
    { key: 'client_clash_linux_url', name: 'Clash', icon: '🐧', desc: 'Linux Clash 客户端' },
    { key: 'client_singbox_url', name: 'Sing-box', icon: '📦', desc: '通用代理平台' },
    { key: 'client_flclash_linux_url', armKey: 'client_flclash_linux_arm_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', desc: 'Linux FlClash 客户端' },
    { key: 'client_hiddify_linux_url', armKey: 'client_hiddify_linux_arm_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', desc: 'Linux Hiddify 客户端' },
    { key: 'client_clashverge_linux_url', armKey: 'client_clashverge_linux_arm_url', name: 'Clash Verge', clientKey: 'clash-verge', icon: '🟣', desc: 'Linux Clash Verge 客户端' },
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
const linuxClients = computed(() => filterClients(allClients.linux))
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
</script>

<style scoped>
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

.client-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 0;
}

.client-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  border-radius: 10px;
  background: rgba(0,0,0,0.03);
  cursor: pointer;
  transition: all 0.2s;
  text-decoration: none;
  color: inherit;
  /* button 元素默认样式重置 */
  border: none;
  font-family: inherit;
  text-align: left;
  width: 100%;
}
.client-card:hover {
  background: rgba(0,0,0,0.04);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.client-icon { font-size: 24px; flex-shrink: 0; }
.client-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.client-name { font-size: 15px; font-weight: 600; }
.client-desc { font-size: 12px; color: var(--text-color-secondary); }

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
  padding: 10px 12px;
  border-radius: 8px;
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
  word-break: break-all;
}

.tutorial { padding: 12px 0; }
.tutorial h4 { margin: 0 0 12px; font-size: 16px; font-weight: 600; }
.tutorial h5 { margin: 16px 0 8px; font-size: 14px; font-weight: 600; color: var(--text-color); }
.tutorial ol { padding-left: 20px; margin: 0; }
.tutorial li { margin: 6px 0; font-size: 14px; line-height: 1.7; color: var(--text-color); }
.tutorial li strong { color: var(--text-color); }
.tut-note { background: var(--bg-color); opacity: 0.7; border: 1px solid #ffe58f; border-radius: 6px; padding: 8px 12px; font-size: 13px; color: #ad6800; margin-bottom: 12px; }
.tut-tip { background: #f6ffed; border: 1px solid #b7eb8f; border-radius: 6px; padding: 8px 12px; font-size: 13px; color: #389e0d; margin-top: 12px; }

@media (max-width: 767px) {
  .help-page { padding: 0 12px; }
  .client-card { padding: 12px 14px; }
}

/* Mobile tutorial cards */
.tut-card-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tut-card {
  border-radius: 10px;
  background: rgba(0,0,0,0.025);
  overflow: hidden;
  cursor: pointer;
  transition: background 0.2s;
}

.tut-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
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
  gap: 1px;
}

.tut-card-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color, #333);
}

.tut-card-platform {
  font-size: 11px;
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
  margin: 4px 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-color);
}

.tut-subtitle {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color);
  margin: 10px 0 4px;
}

.tut-card-body .tut-note,
.tut-card-body .tut-tip {
  font-size: 12px;
  padding: 6px 10px;
}
</style>