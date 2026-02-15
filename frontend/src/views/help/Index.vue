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
            <p>我们支持主流的代理客户端，包括 Clash for Windows、V2rayN、Mihomo Party、Hiddify、FlClash、Shadowrocket、Stash 等。请参考下方客户端下载区域获取对应平台的客户端。</p>
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
import { DownloadOutline } from '@vicons/ionicons5'
import { getPublicConfig } from '@/api/common'

const loadingConfig = ref(false)
const config = ref<Record<string, string>>({})

const allClients = {
  windows: [
    { key: 'client_clash_windows_url', name: 'Clash for Windows', icon: '🔵', desc: 'Clash 内核，支持多种协议' },
    { key: 'client_v2rayn_url', name: 'V2rayN', icon: '🟢', desc: 'V2Ray 图形化客户端' },
    { key: 'client_mihomo_windows_url', name: 'Mihomo Party', icon: '🟣', desc: 'Mihomo 内核 GUI 客户端' },
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
    { key: 'client_mihomo_macos_url', name: 'Mihomo Party', icon: '🟣', desc: 'macOS Mihomo 客户端' },
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
  max-width: 1200px;
  margin: 0 auto;
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
  background: var(--n-color-embedded, #f5f5f5);
  cursor: pointer;
  transition: all 0.2s;
  text-decoration: none;
  color: inherit;
}
.client-card:hover {
  background: var(--n-color-hover, #eee);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.client-icon { font-size: 24px; flex-shrink: 0; }
.client-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.client-name { font-size: 15px; font-weight: 600; }
.client-desc { font-size: 12px; color: #999; }

@media (max-width: 767px) {
  .help-page { padding: 0; }
  .client-card { padding: 12px 14px; }
}
</style>