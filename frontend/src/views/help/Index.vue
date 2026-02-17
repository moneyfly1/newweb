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
                  <a v-for="c in windowsClients" :key="c.key" class="client-card" :href="c.url" target="_blank" rel="noopener">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">{{ c.name }}</span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-icon :component="DownloadOutline" size="18" color="#667eea" />
                  </a>
                </div>
              </n-tab-pane>
              <n-tab-pane name="android" tab="Android" v-if="androidClients.length">
                <div class="client-grid">
                  <a v-for="c in androidClients" :key="c.key" class="client-card" :href="c.url" target="_blank" rel="noopener">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">{{ c.name }}</span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-icon :component="DownloadOutline" size="18" color="#667eea" />
                  </a>
                </div>
              </n-tab-pane>
              <n-tab-pane name="macos" tab="macOS" v-if="macClients.length">
                <div class="client-grid">
                  <a v-for="c in macClients" :key="c.key" class="client-card" :href="c.url" target="_blank" rel="noopener">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">{{ c.name }}</span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-icon :component="DownloadOutline" size="18" color="#667eea" />
                  </a>
                </div>
              </n-tab-pane>
              <n-tab-pane name="ios" tab="iOS" v-if="iosClients.length">
                <div class="client-grid">
                  <a v-for="c in iosClients" :key="c.key" class="client-card" :href="c.url" target="_blank" rel="noopener">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">{{ c.name }}</span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-icon :component="DownloadOutline" size="18" color="#667eea" />
                  </a>
                </div>
              </n-tab-pane>
              <n-tab-pane name="linux" tab="Linux" v-if="linuxClients.length">
                <div class="client-grid">
                  <a v-for="c in linuxClients" :key="c.key" class="client-card" :href="c.url" target="_blank" rel="noopener">
                    <span class="client-icon">{{ c.icon }}</span>
                    <div class="client-info">
                      <span class="client-name">{{ c.name }}</span>
                      <span class="client-desc">{{ c.desc }}</span>
                    </div>
                    <n-icon :component="DownloadOutline" size="18" color="#667eea" />
                  </a>
                </div>
              </n-tab-pane>
            </n-tabs>
          </div>
          <n-empty v-else-if="!loadingConfig" description="管理员暂未配置下载链接" />
        </n-spin>
      </n-card>

      <n-card title="联系我们" :bordered="false">
        <n-space vertical :size="8">
          <n-text>如果您在使用过程中遇到任何问题，可以通过以下方式联系我们：</n-text>
          <n-text>1. 提交工单：前往「工单」页面创建新工单</n-text>
          <n-text>2. 邮件联系：发送邮件至客服邮箱</n-text>
          <n-text depth="3">工作时间：周一至周五 9:00 - 18:00，工单通常在 24 小时内回复。</n-text>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { DownloadOutline, ChevronDownOutline, ChevronUpOutline } from '@vicons/ionicons5'
import { getPublicConfig } from '@/api/common'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const loadingConfig = ref(false)
const config = ref<Record<string, string>>({})
const expandedTut = ref('')

const toggleTut = (key: string) => {
  expandedTut.value = expandedTut.value === key ? '' : key
}

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
    { key: 'client_v2rayn_url', name: 'V2rayN', icon: '🟢', desc: 'V2Ray 图形化客户端' },
    { key: 'client_clashparty_windows_url', name: 'Clash Party', icon: '🟣', desc: 'Clash Party GUI 客户端' },
    { key: 'client_hiddify_windows_url', name: 'Hiddify', icon: '🟠', desc: '多协议代理客户端' },
    { key: 'client_flclash_windows_url', name: 'FlClash', icon: '⚡', desc: 'Flutter 跨平台 Clash 客户端' },
  ],
  android: [
    { key: 'client_clash_android_url', name: 'Clash Meta', icon: '🔵', desc: 'Android Clash 客户端' },
    { key: 'client_v2rayng_url', name: 'V2rayNG', icon: '🟢', desc: 'Android V2Ray 客户端' },
    { key: 'client_hiddify_android_url', name: 'Hiddify', icon: '🟠', desc: 'Android 多协议客户端' },
  ],
  macos: [
    { key: 'client_flclash_macos_url', name: 'FlClash', icon: '⚡', desc: 'macOS Clash 客户端' },
    { key: 'client_clashparty_macos_url', name: 'Clash Party', icon: '🟣', desc: 'macOS Clash Party 客户端' },
  ],
  ios: [
    { key: 'client_shadowrocket_url', name: 'Shadowrocket', icon: '🚀', desc: '需外区 Apple ID 购买' },
    { key: 'client_stash_url', name: 'Stash', icon: '🟡', desc: '基于规则的代理客户端' },
  ],
  linux: [
    { key: 'client_clash_linux_url', name: 'Clash', icon: '🐧', desc: 'Linux Clash 客户端' },
    { key: 'client_singbox_url', name: 'Sing-box', icon: '📦', desc: '通用代理平台' },
  ],
}

const filterClients = (list: typeof allClients.windows) =>
  list.filter(c => config.value[c.key]).map(c => ({ ...c, url: config.value[c.key] }))

const windowsClients = computed(() => filterClients(allClients.windows))
const androidClients = computed(() => filterClients(allClients.android))
const macClients = computed(() => filterClients(allClients.macos))
const iosClients = computed(() => filterClients(allClients.ios))
const linuxClients = computed(() => filterClients(allClients.linux))
const hasAnyClient = computed(() =>
  Object.values(allClients).flat().some(c => config.value[c.key])
)

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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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
}
.client-card:hover {
  background: rgba(0,0,0,0.04);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.client-icon { font-size: 24px; flex-shrink: 0; }
.client-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.client-name { font-size: 15px; font-weight: 600; }
.client-desc { font-size: 12px; color: #999; }

.tutorial { padding: 12px 0; }
.tutorial h4 { margin: 0 0 12px; font-size: 16px; font-weight: 600; }
.tutorial h5 { margin: 16px 0 8px; font-size: 14px; font-weight: 600; color: #555; }
.tutorial ol { padding-left: 20px; margin: 0; }
.tutorial li { margin: 6px 0; font-size: 14px; line-height: 1.7; color: #444; }
.tutorial li strong { color: #333; }
.tut-note { background: #fff7e6; border: 1px solid #ffe58f; border-radius: 6px; padding: 8px 12px; font-size: 13px; color: #ad6800; margin-bottom: 12px; }
.tut-tip { background: #f6ffed; border: 1px solid #b7eb8f; border-radius: 6px; padding: 8px 12px; font-size: 13px; color: #389e0d; margin-top: 12px; }

@media (max-width: 767px) {
  .help-page { padding: 0; }
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
  color: #999;
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
  color: #555;
}

.tut-subtitle {
  font-size: 13px;
  font-weight: 600;
  color: #555;
  margin: 10px 0 4px;
}

.tut-card-body .tut-note,
.tut-card-body .tut-tip {
  font-size: 12px;
  padding: 6px 10px;
}
</style>