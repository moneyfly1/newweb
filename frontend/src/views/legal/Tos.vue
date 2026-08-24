<template>
  <div class="legal-page">
    <div class="legal-header">
      <h1 class="title">服务条款</h1>
      <p class="subtitle">最后更新：2026 年 8 月</p>
    </div>

    <n-card :bordered="false" class="legal-card">
      <p class="lead">
        欢迎使用本平台提供的订阅服务。请您在使用本服务前仔细阅读本《服务条款》（以下简称「本条款」）。
        您通过注册、登录或使用本服务的任何行为，即表示您已阅读、理解并同意受本条款约束。
        如您不同意本条款的任何内容，请停止注册或使用本服务。
      </p>

      <section class="legal-section">
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

      <section class="legal-section">
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

      <section class="legal-section">
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

      <section class="legal-section">
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

      <section class="legal-section">
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

      <section class="legal-section">
        <h2 class="section-title">六、隐私简述</h2>
        <p class="section-para">
          我们仅收集为您提供服务所必需的信息（如账户信息、订单信息、设备信息等），
          并采取合理的加密与安全措施加以保护。我们不会向任何第三方出售您的个人信息。
          完整的个人信息处理规则，请查阅《隐私政策》。
        </p>
      </section>

      <section class="legal-section">
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
</script>

<style scoped>
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

.legal-card {
  border-radius: 10px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
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
  .legal-page {
    padding: 8px;
  }
  .title {
    font-size: 24px;
  }
  .legal-card {
    border-radius: 14px;
  }
}
</style>
