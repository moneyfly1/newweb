<template>
  <div class="not-found">
    <div class="not-found-code" aria-hidden="true">404</div>
    <EmptyState
      :icon="PlanetOutline"
      description="你访问的页面不存在或已被移除"
      action-text="返回首页"
      @action="goHome"
    />
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { PlanetOutline } from '@vicons/ionicons5'
import EmptyState from '@/components/EmptyState.vue'

const router = useRouter()

function goHome() {
  router.push('/')
}
</script>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 0 24px;
}

.not-found-code {
  font-size: 88px;
  font-weight: 800;
  line-height: 1;
  background: var(--brand-gradient, linear-gradient(135deg, #667eea 0%, #764ba2 100%));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 8px;
}

@media (max-width: 767px) {
  .not-found {
    padding: 0 16px;
  }

  .not-found-code {
    font-size: 64px;
  }
}
</style>
