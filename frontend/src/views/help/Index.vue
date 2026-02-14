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
            <p>我们支持主流的代理客户端，包括 Clash for Windows、ClashX、Clash Verge、V2rayN、Shadowrocket、Quantumult X、Stash、Surfboard 等。请参考下方客户端下载区域获取对应平台的客户端。</p>
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

      <n-card title="客户端下载" :bordered="false">
        <n-grid :x-gap="16" :y-gap="16" cols="1 s:2 m:3 l:4" responsive="screen">
          <n-gi v-for="client in clients" :key="client.name">
            <n-card size="small" :bordered="true" hoverable>
              <div class="client-card">
                <div class="client-name">{{ client.name }}</div>
                <n-tag size="small" :type="client.tagType" :bordered="false">{{ client.platform }}</n-tag>
                <div class="client-desc">{{ client.desc }}</div>
              </div>
            </n-card>
          </n-gi>
        </n-grid>
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
const clients = [
  { name: 'Clash for Windows', platform: 'Windows', tagType: 'info' as const, desc: 'Windows 平台主流代理客户端，支持多种协议' },
  { name: 'ClashX', platform: 'Mac', tagType: 'success' as const, desc: 'macOS 平台轻量级 Clash 客户端' },
  { name: 'Clash Verge', platform: '跨平台', tagType: 'warning' as const, desc: '基于 Tauri 的跨平台 Clash GUI 客户端' },
  { name: 'V2rayN', platform: 'Windows', tagType: 'info' as const, desc: 'Windows 平台 V2Ray 图形化客户端' },
  { name: 'Shadowrocket', platform: 'iOS', tagType: 'error' as const, desc: 'iOS 平台热门代理工具，需外区 Apple ID 购买' },
  { name: 'Quantumult X', platform: 'iOS', tagType: 'error' as const, desc: 'iOS 平台功能强大的网络工具' },
  { name: 'Stash', platform: 'iOS', tagType: 'error' as const, desc: 'iOS 平台基于规则的代理客户端' },
  { name: 'Surfboard', platform: 'Android', tagType: 'success' as const, desc: 'Android 平台代理客户端，兼容 Surge 配置' },
]
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

.client-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.client-name {
  font-size: 15px;
  font-weight: 600;
}

.client-desc {
  font-size: 13px;
  color: #999;
  line-height: 1.5;
}

@media (max-width: 767px) {
  .help-page { padding: 0; }
}
</style>
