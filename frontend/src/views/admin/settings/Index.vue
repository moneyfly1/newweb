<template>
  <div class="settings-container">
    <n-card title="系统设置" :bordered="false">
      <n-spin :show="loading">
        <n-tabs type="line" animated>
          <!-- Tab 1: 基本设置 -->
          <n-tab-pane name="basic" tab="基本设置">
            <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '140'" :model="form">
              <n-form-item label="站点名称">
                <n-input v-model:value="form.site_name" placeholder="请输入站点名称" />
              </n-form-item>
              <n-form-item label="站点描述">
                <n-input v-model:value="form.site_description" type="textarea" placeholder="请输入站点描述" :rows="3" />
              </n-form-item>
              <n-form-item label="站点地址">
                <n-input v-model:value="form.site_url" placeholder="https://example.com" />
              </n-form-item>
              <n-form-item label="客服邮箱">
                <n-input v-model:value="form.support_email" placeholder="support@example.com" />
              </n-form-item>
              <n-form-item label="客服 QQ">
                <n-input v-model:value="form.support_qq" placeholder="请输入客服 QQ" />
              </n-form-item>
              <n-form-item label="客服 Telegram">
                <n-input v-model:value="form.support_telegram" placeholder="请输入 Telegram 用户名" />
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 2: 注册设置 -->
          <n-tab-pane name="register" tab="注册设置">
            <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '200'" :model="form">
              <n-form-item label="开放注册">
                <n-switch v-model:value="form.register_enabled" />
              </n-form-item>
              <n-form-item label="注册邮箱验证">
                <n-switch v-model:value="form.register_email_verify" />
              </n-form-item>
              <n-form-item label="注册需要邀请码">
                <n-switch v-model:value="form.register_invite_required" />
              </n-form-item>
              <n-form-item label="新用户默认设备数限制">
                <n-input-number v-model:value="form.default_device_limit" :min="1" :max="100" />
              </n-form-item>
              <n-form-item label="新用户默认订阅天数">
                <n-input-number v-model:value="form.default_subscribe_days" :min="0" :max="3650" />
              </n-form-item>
              <n-form-item label="最小密码长度">
                <n-input-number v-model:value="form.min_password_length" :min="6" :max="32" />
              </n-form-item>
              <n-divider />
              <n-form-item label="启用 Telegram 登录">
                <n-switch v-model:value="form.telegram_login_enabled" />
              </n-form-item>
              <n-form-item label="Telegram Bot Username">
                <n-input v-model:value="form.telegram_bot_username" placeholder="不含 @ 的 Bot 用户名" />
                <template #feedback>
                  <span style="font-size: 12px; color: #999">用于 Telegram Login Widget，需与通知设置中的 Bot Token 对应</span>
                </template>
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 3: 邮件设置 -->
          <n-tab-pane name="email" tab="邮件设置">
            <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '140'" :model="form">
              <n-form-item label="SMTP 主机">
                <n-input v-model:value="form.smtp_host" placeholder="smtp.example.com" />
              </n-form-item>
              <n-form-item label="SMTP 端口">
                <n-input-number v-model:value="form.smtp_port" :min="1" :max="65535" />
              </n-form-item>
              <n-form-item label="SMTP 用户名">
                <n-input v-model:value="form.smtp_username" placeholder="请输入 SMTP 用户名" />
              </n-form-item>
              <n-form-item label="SMTP 密码">
                <n-input v-model:value="form.smtp_password" type="password" show-password-on="click" placeholder="请输入 SMTP 密码" />
              </n-form-item>
              <n-form-item label="加密方式">
                <n-select v-model:value="form.smtp_encryption" :options="encryptionOptions" placeholder="请选择加密方式" />
              </n-form-item>
              <n-form-item label="发件人邮箱">
                <n-input v-model:value="form.smtp_from_email" placeholder="noreply@example.com" />
              </n-form-item>
              <n-form-item label="发件人名称">
                <n-input v-model:value="form.smtp_from_name" placeholder="请输入发件人名称" />
              </n-form-item>
              <n-divider />
              <n-form-item label="测试邮件">
                <n-space :vertical="appStore.isMobile">
                  <n-input v-model:value="testEmail" placeholder="输入测试邮箱地址" :style="{ width: appStore.isMobile ? '100%' : '280px' }" />
                  <n-button type="info" :loading="sendingTest" @click="handleSendTestEmail" :block="appStore.isMobile">发送测试邮件</n-button>
                </n-space>
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 4: 支付设置 -->
          <n-tab-pane name="payment" tab="支付设置">
            <n-space vertical :size="16">
              <n-card title="余额支付" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '140'" :model="form">
                  <n-form-item label="启用余额支付">
                    <n-switch v-model:value="form.pay_balance_enabled" />
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="支付宝" size="small" :bordered="true" :collapsible="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '140'" :model="form">
                  <n-form-item label="启用支付宝">
                    <n-switch v-model:value="form.pay_alipay_enabled" />
                  </n-form-item>
                  <n-form-item label="App ID">
                    <n-input v-model:value="form.pay_alipay_app_id" placeholder="请输入支付宝 App ID" />
                  </n-form-item>
                  <n-form-item label="应用私钥">
                    <n-input v-model:value="form.pay_alipay_private_key" type="textarea" placeholder="请输入应用私钥（PKCS1 或 PKCS8 格式）" :rows="3" />
                  </n-form-item>
                  <n-form-item label="支付宝公钥">
                    <n-input v-model:value="form.pay_alipay_public_key" type="textarea" placeholder="请输入支付宝公钥（用于验证回调签名）" :rows="3" />
                  </n-form-item>
                  <n-form-item label="异步回调地址">
                    <n-input v-model:value="form.pay_alipay_notify_url" :placeholder="alipayNotifyUrlHint" />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">支付宝服务器通知地址，留空则自动使用: {{ alipayNotifyUrlHint }}</span>
                    </template>
                  </n-form-item>
                  <n-form-item label="同步回调地址">
                    <n-input v-model:value="form.pay_alipay_return_url" :placeholder="alipayReturnUrlHint" />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">支付完成后跳转地址，留空则自动使用: {{ alipayReturnUrlHint }}</span>
                    </template>
                  </n-form-item>
                  <n-form-item label="沙箱模式">
                    <n-switch v-model:value="form.pay_alipay_sandbox" />
                    <span style="margin-left: 8px; font-size: 12px; color: #999">开启后使用支付宝沙箱环境测试</span>
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="微信支付" size="small" :bordered="true" :collapsible="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '140'" :model="form">
                  <n-form-item label="启用微信支付">
                    <n-switch v-model:value="form.pay_wechat_enabled" />
                  </n-form-item>
                  <n-form-item label="App ID">
                    <n-input v-model:value="form.pay_wechat_app_id" placeholder="请输入微信 App ID" />
                  </n-form-item>
                  <n-form-item label="MCH ID">
                    <n-input v-model:value="form.pay_wechat_mch_id" placeholder="请输入商户号" />
                  </n-form-item>
                  <n-form-item label="API Key">
                    <n-input v-model:value="form.pay_wechat_api_key" type="password" show-password-on="click" placeholder="请输入 API 密钥" />
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="易支付" size="small" :bordered="true" :collapsible="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '140'" :model="form">
                  <n-form-item label="启用易支付">
                    <n-switch v-model:value="form.pay_epay_enabled" />
                  </n-form-item>
                  <n-form-item label="网关地址">
                    <n-input v-model:value="form.pay_epay_gateway" placeholder="https://pay.example.com" />
                  </n-form-item>
                  <n-form-item label="商户 ID">
                    <n-input v-model:value="form.pay_epay_merchant_id" placeholder="请输入商户 ID" />
                  </n-form-item>
                  <n-form-item label="商户密钥">
                    <n-input v-model:value="form.pay_epay_secret_key" type="password" show-password-on="click" placeholder="请输入商户密钥" />
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="Stripe" size="small" :bordered="true" :collapsible="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '160'" :model="form">
                  <n-form-item label="启用 Stripe">
                    <n-switch v-model:value="form.pay_stripe_enabled" />
                  </n-form-item>
                  <n-form-item label="Secret Key">
                    <n-input v-model:value="form.pay_stripe_secret_key" type="password" show-password-on="click" placeholder="sk_live_..." />
                  </n-form-item>
                  <n-form-item label="Publishable Key">
                    <n-input v-model:value="form.pay_stripe_publishable_key" placeholder="pk_live_..." />
                  </n-form-item>
                  <n-form-item label="Webhook Secret">
                    <n-input v-model:value="form.pay_stripe_webhook_secret" type="password" show-password-on="click" placeholder="whsec_..." />
                  </n-form-item>
                  <n-form-item label="汇率 (CNY→USD)">
                    <n-input-number v-model:value="form.pay_stripe_exchange_rate" :min="0.01" :max="100" :precision="2" placeholder="7.2" />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">人民币兑美元汇率，用于自动换算支付金额</span>
                    </template>
                  </n-form-item>
                  <n-form-item label="Webhook 地址">
                    <n-input :value="stripeWebhookUrlHint" readonly />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">请在 Stripe Dashboard 中配置此 Webhook 地址</span>
                    </template>
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="加密货币 (USDT)" size="small" :bordered="true" :collapsible="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '160'" :model="form">
                  <n-form-item label="启用加密货币">
                    <n-switch v-model:value="form.pay_crypto_enabled" />
                  </n-form-item>
                  <n-form-item label="钱包地址">
                    <n-input v-model:value="form.pay_crypto_wallet_address" placeholder="请输入收款钱包地址" />
                  </n-form-item>
                  <n-form-item label="网络">
                    <n-select v-model:value="form.pay_crypto_network" :options="cryptoNetworkOptions" placeholder="请选择网络" />
                  </n-form-item>
                  <n-form-item label="币种">
                    <n-select v-model:value="form.pay_crypto_currency" :options="cryptoCurrencyOptions" placeholder="请选择币种" />
                  </n-form-item>
                  <n-form-item label="汇率 (CNY→USDT)">
                    <n-input-number v-model:value="form.pay_crypto_exchange_rate" :min="0.01" :max="100" :precision="2" placeholder="7.2" />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">人民币兑 USDT 汇率，用于自动换算支付金额</span>
                    </template>
                  </n-form-item>
                </n-form>
              </n-card>

              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-space>
          </n-tab-pane>

          <!-- Tab 5: 通知设置 -->
          <n-tab-pane name="notification" tab="通知设置">
            <n-space vertical :size="16">
              <!-- 通知渠道配置 -->
              <n-card title="通知渠道" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-h4 prefix="bar">邮件通知</n-h4>
                  <n-form-item label="启用邮件通知">
                    <n-switch v-model:value="form.notify_email_enabled" />
                  </n-form-item>
                  <n-form-item label="管理员通知邮箱">
                    <n-input v-model:value="form.notify_admin_email" placeholder="admin@example.com" />
                  </n-form-item>
                  <n-divider style="margin: 12px 0" />
                  <n-h4 prefix="bar">Telegram 通知</n-h4>
                  <n-form-item label="启用 Telegram">
                    <n-switch v-model:value="form.notify_telegram_enabled" />
                  </n-form-item>
                  <n-form-item label="Bot Token">
                    <n-input v-model:value="form.notify_telegram_bot_token" placeholder="请输入 Bot Token" />
                  </n-form-item>
                  <n-form-item label="Chat ID">
                    <n-input v-model:value="form.notify_telegram_chat_id" placeholder="请输入 Chat ID" />
                  </n-form-item>
                  <n-form-item label="测试">
                    <n-button type="info" :loading="testingTelegram" @click="handleTestTelegram" :disabled="!form.notify_telegram_enabled">发送测试消息</n-button>
                  </n-form-item>
                  <n-divider style="margin: 12px 0" />
                  <n-h4 prefix="bar">Bark 通知</n-h4>
                  <n-form-item label="启用 Bark">
                    <n-switch v-model:value="form.notify_bark_enabled" />
                  </n-form-item>
                  <n-form-item label="服务器地址">
                    <n-input v-model:value="form.notify_bark_server" placeholder="https://api.day.app" />
                  </n-form-item>
                  <n-form-item label="Device Key">
                    <n-input v-model:value="form.notify_bark_device_key" placeholder="请输入 Device Key" />
                  </n-form-item>
                </n-form>
              </n-card>

              <!-- 管理员事件通知开关 -->
              <n-card title="管理员事件通知" size="small" :bordered="true">
                <template #header-extra><n-text depth="3" style="font-size: 12px">触发时通知管理员</n-text></template>
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="新用户注册"><n-switch v-model:value="form.notify_new_user" /></n-form-item>
                  <n-form-item label="新订单创建"><n-switch v-model:value="form.notify_new_order" /></n-form-item>
                  <n-form-item label="支付成功"><n-switch v-model:value="form.notify_payment_success" /></n-form-item>
                  <n-form-item label="充值成功"><n-switch v-model:value="form.notify_recharge_success" /></n-form-item>
                  <n-form-item label="新工单"><n-switch v-model:value="form.notify_new_ticket" /></n-form-item>
                  <n-form-item label="订阅到期提醒"><n-switch v-model:value="form.notify_expiry_reminder" /></n-form-item>
                  <n-form-item label="订阅重置"><n-switch v-model:value="form.notify_subscription_reset" /></n-form-item>
                  <n-form-item label="异常登录"><n-switch v-model:value="form.notify_abnormal_login" /></n-form-item>
                  <n-form-item label="未支付订单提醒"><n-switch v-model:value="form.notify_unpaid_order" /></n-form-item>
                </n-form>
              </n-card>

              <!-- 用户邮件通知开关 -->
              <n-card title="用户邮件通知" size="small" :bordered="true">
                <template #header-extra><n-text depth="3" style="font-size: 12px">系统级控制发给用户的邮件</n-text></template>
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="注册欢迎邮件"><n-switch v-model:value="form.user_notify_welcome" /></n-form-item>
                  <n-form-item label="支付成功通知"><n-switch v-model:value="form.user_notify_payment" /></n-form-item>
                  <n-form-item label="订阅到期提醒"><n-switch v-model:value="form.user_notify_expiry" /></n-form-item>
                  <n-form-item label="订阅过期通知"><n-switch v-model:value="form.user_notify_expired" /></n-form-item>
                  <n-form-item label="订阅重置通知"><n-switch v-model:value="form.user_notify_reset" /></n-form-item>
                  <n-form-item label="账户状态变更"><n-switch v-model:value="form.user_notify_account_status" /></n-form-item>
                  <n-form-item label="未支付订单提醒"><n-switch v-model:value="form.user_notify_unpaid_order" /></n-form-item>
                </n-form>
              </n-card>

              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-space>
          </n-tab-pane>

          <!-- Tab 6: 安全设置 -->
          <n-tab-pane name="security" tab="安全设置">
            <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '200'" :model="form">
              <n-form-item label="最大登录尝试次数">
                <n-input-number v-model:value="form.max_login_attempts" :min="1" :max="100" />
              </n-form-item>
              <n-form-item label="登录锁定时长 (分钟)">
                <n-input-number v-model:value="form.login_lockout_minutes" :min="1" :max="1440" />
              </n-form-item>
              <n-form-item label="会话超时 (分钟)">
                <n-input-number v-model:value="form.session_timeout_minutes" :min="5" :max="10080" />
              </n-form-item>
              <n-form-item label="异常登录提醒">
                <n-switch v-model:value="form.abnormal_login_alert" />
              </n-form-item>
              <n-form-item label="IP 白名单">
                <n-input v-model:value="form.ip_whitelist" type="textarea" placeholder="每行一个 IP 地址" :rows="5" />
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>
          <!-- Tab 7: 签到设置 -->
          <n-tab-pane name="checkin" tab="签到设置">
            <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '200'" :model="form">
              <n-form-item label="启用签到功能">
                <n-switch v-model:value="form.checkin_enabled" />
              </n-form-item>
              <n-form-item label="最小奖励金额 (分)">
                <n-input-number v-model:value="form.checkin_min_reward" :min="1" :max="10000" />
                <template #feedback>
                  <span style="font-size: 12px; color: #999">签到随机奖励的最小值，单位为分（如 10 = 0.10 元）</span>
                </template>
              </n-form-item>
              <n-form-item label="最大奖励金额 (分)">
                <n-input-number v-model:value="form.checkin_max_reward" :min="1" :max="10000" />
                <template #feedback>
                  <span style="font-size: 12px; color: #999">签到随机奖励的最大值，单位为分（如 50 = 0.50 元）</span>
                </template>
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>
          <!-- Tab 8: 软件下载配置 -->
          <n-tab-pane name="client" tab="软件下载">
            <n-space vertical :size="16">
              <n-card title="Windows 客户端" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="Clash for Windows"><n-input v-model:value="form.client_clash_windows_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="V2rayN"><n-input v-model:value="form.client_v2rayn_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="Mihomo Party"><n-input v-model:value="form.client_mihomo_windows_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="Hiddify"><n-input v-model:value="form.client_hiddify_windows_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="FlClash"><n-input v-model:value="form.client_flclash_windows_url" placeholder="下载链接" /></n-form-item>
                </n-form>
              </n-card>
              <n-card title="Android 客户端" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="Clash Meta"><n-input v-model:value="form.client_clash_android_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="V2rayNG"><n-input v-model:value="form.client_v2rayng_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="Hiddify"><n-input v-model:value="form.client_hiddify_android_url" placeholder="下载链接" /></n-form-item>
                </n-form>
              </n-card>
              <n-card title="macOS 客户端" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="FlClash"><n-input v-model:value="form.client_flclash_macos_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="Mihomo Party"><n-input v-model:value="form.client_mihomo_macos_url" placeholder="下载链接" /></n-form-item>
                </n-form>
              </n-card>
              <n-card title="iOS 客户端" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="Shadowrocket"><n-input v-model:value="form.client_shadowrocket_url" placeholder="App Store 链接" /></n-form-item>
                  <n-form-item label="Stash"><n-input v-model:value="form.client_stash_url" placeholder="App Store 链接" /></n-form-item>
                </n-form>
              </n-card>
              <n-card title="通用客户端" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '180'" :model="form">
                  <n-form-item label="Sing-box"><n-input v-model:value="form.client_singbox_url" placeholder="下载链接" /></n-form-item>
                  <n-form-item label="Clash (Linux)"><n-input v-model:value="form.client_clash_linux_url" placeholder="下载链接" /></n-form-item>
                </n-form>
              </n-card>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-space>
          </n-tab-pane>

          <!-- Tab 8: 备份与恢复 -->
          <n-tab-pane name="backup" tab="备份与恢复">
            <n-space vertical :size="16">
              <n-card title="数据库备份" size="small" :bordered="true">
                <n-space vertical :size="12">
                  <n-text>创建当前数据库的完整备份。备份文件将保存在服务器上。</n-text>
                  <n-space>
                    <n-button type="primary" :loading="backupCreating" @click="handleCreateBackup">创建备份</n-button>
                    <n-button @click="loadBackups" :loading="backupLoading">刷新列表</n-button>
                  </n-space>
                </n-space>
              </n-card>

              <n-card title="GitHub 备份设置" size="small" :bordered="true">
                <n-form :label-placement="appStore.isMobile ? 'top' : 'left'" :label-width="appStore.isMobile ? 'auto' : '160'" :model="form">
                  <n-form-item label="启用 GitHub 备份">
                    <n-switch v-model:value="form.backup_github_enabled" />
                  </n-form-item>
                  <n-form-item label="GitHub Token">
                    <n-input v-model:value="form.backup_github_token" type="password" show-password-on="click" placeholder="ghp_xxxxxxxxxxxx" />
                  </n-form-item>
                  <n-form-item label="仓库地址">
                    <n-input v-model:value="form.backup_github_repo" placeholder="username/repo-name" />
                  </n-form-item>
                  <n-form-item label="自动备份间隔 (小时)">
                    <n-input-number v-model:value="form.backup_interval_hours" :min="0" :max="720" />
                  </n-form-item>
                  <n-space justify="center" style="margin-top: 16px">
                    <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
                  </n-space>
                </n-form>
              </n-card>

              <n-card title="备份记录" size="small" :bordered="true">
                <template v-if="!appStore.isMobile">
                  <n-data-table
                    :columns="backupColumns"
                    :data="backups"
                    :loading="backupLoading"
                    :bordered="false"
                    size="small"
                  />
                </template>
                <template v-else>
                  <div class="mobile-card-list">
                    <div v-for="item in backups" :key="item.filename" class="mobile-card">
                      <div class="card-header">
                        <span class="card-title">{{ item.filename }}</span>
                      </div>
                      <div class="card-body">
                        <div class="card-row"><span class="card-label">大小:</span><span>{{ item.size }}</span></div>
                        <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                      </div>
                    </div>
                  </div>
                </template>
                <n-empty v-if="!backupLoading && backups.length === 0" description="暂无备份记录" style="padding: 40px 0" />
              </n-card>
            </n-space>
          </n-tab-pane>
        </n-tabs>
      </n-spin>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage, NH4 } from 'naive-ui'
import { getSettings, updateSettings, sendTestEmail, testTelegram, createBackup, listBackups } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const sendingTest = ref(false)
const testEmail = ref('')
const backupLoading = ref(false)
const backupCreating = ref(false)
const testingTelegram = ref(false)
const backups = ref<any[]>([])

// Computed URL hints based on site_url
const siteBase = computed(() => {
  const url = form.value.site_url || ''
  if (!url) return window.location.origin
  return url.startsWith('http') ? url.replace(/\/+$/, '') : 'https://' + url.replace(/\/+$/, '')
})
const alipayNotifyUrlHint = computed(() => siteBase.value + '/api/v1/payment/notify/alipay')
const alipayReturnUrlHint = computed(() => siteBase.value + '/payment/return')
const stripeWebhookUrlHint = computed(() => siteBase.value + '/api/v1/payment/notify/stripe')

const encryptionOptions = [
  { label: '无', value: 'none' },
  { label: 'TLS', value: 'tls' },
  { label: 'SSL', value: 'ssl' }
]

const cryptoNetworkOptions = [
  { label: 'TRC20 (Tron)', value: 'TRC20' },
  { label: 'ERC20 (Ethereum)', value: 'ERC20' }
]

const cryptoCurrencyOptions = [
  { label: 'USDT', value: 'USDT' },
  { label: 'USDC', value: 'USDC' }
]

// Boolean keys that should be stored as switches
const booleanKeys = [
  'register_enabled', 'register_email_verify', 'register_invite_required',
  'pay_balance_enabled', 'pay_alipay_enabled', 'pay_alipay_sandbox', 'pay_wechat_enabled', 'pay_epay_enabled',
  'pay_stripe_enabled', 'pay_crypto_enabled',
  'notify_email_enabled', 'notify_telegram_enabled', 'notify_bark_enabled',
  'notify_new_order', 'notify_new_ticket', 'notify_new_user', 'notify_expiry_reminder',
  'notify_payment_success', 'notify_recharge_success', 'notify_subscription_reset',
  'notify_abnormal_login', 'notify_unpaid_order',
  'user_notify_welcome', 'user_notify_payment', 'user_notify_expiry', 'user_notify_expired',
  'user_notify_reset', 'user_notify_account_status', 'user_notify_unpaid_order',
  'abnormal_login_alert', 'backup_github_enabled',
  'checkin_enabled', 'telegram_login_enabled'
]

// Number keys that should be stored as numbers
const numberKeys = [
  'default_device_limit', 'default_subscribe_days', 'min_password_length',
  'smtp_port', 'max_login_attempts', 'login_lockout_minutes', 'session_timeout_minutes',
  'backup_interval_hours', 'pay_stripe_exchange_rate', 'pay_crypto_exchange_rate',
  'checkin_min_reward', 'checkin_max_reward'
]

const form = ref<Record<string, any>>({
  // Basic
  site_name: '',
  site_description: '',
  site_url: '',
  support_email: '',
  support_qq: '',
  support_telegram: '',
  // Registration
  register_enabled: true,
  register_email_verify: false,
  register_invite_required: false,
  default_device_limit: 3,
  default_subscribe_days: 0,
  min_password_length: 8,
  telegram_login_enabled: false,
  telegram_bot_username: '',
  // Email
  smtp_host: '',
  smtp_port: 465,
  smtp_username: '',
  smtp_password: '',
  smtp_encryption: 'ssl',
  smtp_from_email: '',
  smtp_from_name: '',
  // Payment
  pay_balance_enabled: true,
  pay_alipay_enabled: false,
  pay_alipay_app_id: '',
  pay_alipay_private_key: '',
  pay_alipay_public_key: '',
  pay_alipay_notify_url: '',
  pay_alipay_return_url: '',
  pay_alipay_sandbox: false,
  pay_wechat_enabled: false,
  pay_wechat_app_id: '',
  pay_wechat_mch_id: '',
  pay_wechat_api_key: '',
  pay_epay_enabled: false,
  pay_epay_gateway: '',
  pay_epay_merchant_id: '',
  pay_epay_secret_key: '',
  // Stripe
  pay_stripe_enabled: false,
  pay_stripe_secret_key: '',
  pay_stripe_publishable_key: '',
  pay_stripe_webhook_secret: '',
  pay_stripe_exchange_rate: 7.2,
  // Crypto
  pay_crypto_enabled: false,
  pay_crypto_wallet_address: '',
  pay_crypto_network: 'TRC20',
  pay_crypto_currency: 'USDT',
  pay_crypto_exchange_rate: 7.2,
  // Notification - channels
  notify_email_enabled: true,
  notify_admin_email: '',
  notify_telegram_enabled: false,
  notify_telegram_bot_token: '',
  notify_telegram_chat_id: '',
  notify_bark_enabled: false,
  notify_bark_server: '',
  notify_bark_device_key: '',
  // Notification - admin events
  notify_new_order: false,
  notify_new_ticket: false,
  notify_new_user: false,
  notify_expiry_reminder: false,
  notify_payment_success: false,
  notify_recharge_success: false,
  notify_subscription_reset: false,
  notify_abnormal_login: false,
  notify_unpaid_order: false,
  // Notification - user email
  user_notify_welcome: true,
  user_notify_payment: true,
  user_notify_expiry: true,
  user_notify_expired: true,
  user_notify_reset: true,
  user_notify_account_status: true,
  user_notify_unpaid_order: true,
  // Security
  max_login_attempts: 5,
  login_lockout_minutes: 30,
  session_timeout_minutes: 120,
  abnormal_login_alert: true,
  ip_whitelist: '',
  // Backup
  backup_github_enabled: false,
  backup_github_token: '',
  backup_github_repo: '',
  backup_interval_hours: 24,
  // Client download URLs
  client_clash_windows_url: '',
  client_v2rayn_url: '',
  client_mihomo_windows_url: '',
  client_hiddify_windows_url: '',
  client_flclash_windows_url: '',
  client_clash_android_url: '',
  client_v2rayng_url: '',
  client_hiddify_android_url: '',
  client_flclash_macos_url: '',
  client_mihomo_macos_url: '',
  client_shadowrocket_url: '',
  client_stash_url: '',
  client_singbox_url: '',
  client_clash_linux_url: '',
  // Check-in
  checkin_enabled: true,
  checkin_min_reward: 10,
  checkin_max_reward: 50,
})

const loadSettings = async () => {
  loading.value = true
  try {
    const res = await getSettings()
    if (res.code === 0 && res.data) {
      const data = res.data as Record<string, any>
      for (const key of Object.keys(form.value)) {
        if (key in data) {
          if (booleanKeys.includes(key)) {
            form.value[key] = data[key] === true || data[key] === 'true' || data[key] === '1'
          } else if (numberKeys.includes(key)) {
            form.value[key] = Number(data[key]) || form.value[key]
          } else {
            form.value[key] = data[key]
          }
        }
      }
    } else {
      message.error(res.message || '加载设置失败')
    }
  } catch (error: any) {
    message.error(error.message || '加载设置失败')
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    const res = await updateSettings(form.value)
    if (res.code === 0) {
      message.success('保存成功')
    } else {
      message.error(res.message || '保存失败')
    }
  } catch (error: any) {
    message.error(error.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const handleSendTestEmail = async () => {
  if (!testEmail.value) {
    message.warning('请输入测试邮箱地址')
    return
  }
  sendingTest.value = true
  try {
    const res = await sendTestEmail({ email: testEmail.value })
    if (res.code === 0) {
      message.success('测试邮件已发送')
    } else {
      message.error(res.message || '发送失败')
    }
  } catch (error: any) {
    message.error(error.message || '发送失败')
  } finally {
    sendingTest.value = false
  }
}

const handleTestTelegram = async () => {
  testingTelegram.value = true
  try {
    const res = await testTelegram()
    if (res.code === 0) {
      message.success('Telegram 测试消息已发送')
    } else {
      message.error(res.message || '发送失败')
    }
  } catch (error: any) {
    message.error(error.message || '发送失败')
  } finally {
    testingTelegram.value = false
  }
}

const backupColumns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true } },
  { title: '大小', key: 'size' },
  { title: '创建时间', key: 'created_at', width: 180 },
]

const loadBackups = async () => {
  backupLoading.value = true
  try {
    const res = await listBackups()
    backups.value = res.data || []
  } catch (error: any) {
    message.error(error.message || '加载备份列表失败')
  } finally {
    backupLoading.value = false
  }
}

const handleCreateBackup = async () => {
  backupCreating.value = true
  try {
    const res = await createBackup()
    message.success(res.message || '备份已创建')
    loadBackups()
  } catch (error: any) {
    message.error(error.message || '创建备份失败')
  } finally {
    backupCreating.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-container {
  padding: 20px;
}

:deep(.n-tabs-pane-wrapper) {
  padding-top: 20px;
}

:deep(.n-card.n-card--bordered) {
  margin-bottom: 0;
}

.mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
.mobile-card { background: #fff; border-radius: 10px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); overflow: hidden; }
.card-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid #f0f0f0; }
.card-title { font-weight: 600; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.card-body { padding: 10px 14px; }
.card-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; font-size: 13px; }
.card-label { color: #999; }

@media (max-width: 767px) {
  .settings-container { padding: 8px; }
  :deep(.n-tabs-pane-wrapper) { padding-top: 12px; }
  :deep(.n-form-item .n-form-item-label) { min-width: 80px; }
}
</style>
