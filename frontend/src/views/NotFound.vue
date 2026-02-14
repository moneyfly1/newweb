<template>
  <div class="not-found">
    <div class="not-found-content">
      <div class="error-code">404</div>
      <div class="error-message">页面未找到</div>
      <n-text depth="3" style="font-size: 15px; margin-bottom: 32px; display: block;">
        你访问的页面不存在或已被移除
      </n-text>
      <n-button type="primary" size="large" @click="$router.push('/')">
        返回首页
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  text-align: center;
}

.not-found-content {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.error-code {
  font-size: 120px;
  font-weight: 800;
  line-height: 1;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 16px;
}

.error-message {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;
  color: #333;
}
</style>
