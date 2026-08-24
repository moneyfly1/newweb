<template>
  <div class="auth-layout">
    <!-- 左栏：品牌面板（桌面端展示，移动端隐藏） -->
    <aside class="auth-brand-panel" aria-label="CBoard 品牌信息">
      <div class="brand-glow brand-glow--1" aria-hidden="true" />
      <div class="brand-glow brand-glow--2" aria-hidden="true" />
      <div class="brand-panel-inner">
        <div class="brand-head">
          <div class="brand-logo" aria-hidden="true">C</div>
          <div class="brand-name">CBoard</div>
        </div>
        <p class="brand-tagline">{{ title }}</p>
        <ul class="brand-features">
          <li v-for="feature in features" :key="feature" class="brand-feature">
            <span class="brand-feature-dot" aria-hidden="true" />
            <span>{{ feature }}</span>
          </li>
        </ul>
      </div>
    </aside>

    <!-- 右栏：表单区（移动端只保留此栏 + 顶部品牌条） -->
    <main class="auth-main">
      <div class="auth-mobile-brand">
        <div class="auth-mobile-logo" aria-hidden="true">C</div>
        <div class="auth-mobile-name">CBoard</div>
      </div>
      <div class="auth-form-area">
        <slot />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  /** 品牌标语，如「高速稳定 · 全球节点」 */
  title: string
  /** 卖点列表（左栏品牌面板展示），缺省使用默认三条 */
  subtitle?: string[]
}>(), {
  subtitle: () => [
    '多格式订阅聚合，一键导入',
    '智能设备管理，安全可控',
    '实时节点监控，稳定高速',
  ],
})

const features = computed<string[]>(() => props.subtitle ?? [])
</script>

<style scoped>
.auth-layout {
  height: 100vh;
  min-height: 100vh;
  display: flex;
  overflow: hidden;
}

/* ===== 左栏：品牌面板 ===== */
.auth-brand-panel {
  position: relative;
  flex: 1 1 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 64px;
  background: var(--brand-gradient, linear-gradient(135deg, #667eea, #764ba2));
  color: #fff;
  overflow: hidden;
  animation: auth-fade-up 380ms cubic-bezier(0.22, 0.61, 0.36, 1) both;
}

/* 光晕装饰 */
.brand-glow {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
  filter: blur(64px);
}
.brand-glow--1 {
  width: 380px;
  height: 380px;
  top: -130px;
  right: -120px;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.22) 0%, rgba(255, 255, 255, 0) 70%);
}
.brand-glow--2 {
  width: 320px;
  height: 320px;
  bottom: -110px;
  left: -110px;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.16) 0%, rgba(255, 255, 255, 0) 70%);
}

.brand-panel-inner {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 420px;
}

.brand-head {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 36px;
  animation: auth-fade-up 380ms cubic-bezier(0.22, 0.61, 0.36, 1) 120ms both;
}
.brand-logo {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 26px;
  font-weight: 800;
  background: rgba(255, 255, 255, 0.16);
  border: 1px solid rgba(255, 255, 255, 0.32);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}
.brand-name {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.brand-tagline {
  font-size: 30px;
  line-height: 1.35;
  font-weight: 700;
  margin-bottom: 40px;
  animation: auth-fade-up 380ms cubic-bezier(0.22, 0.61, 0.36, 1) 180ms both;
}

.brand-features {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
  animation: auth-fade-up 380ms cubic-bezier(0.22, 0.61, 0.36, 1) 240ms both;
}
.brand-feature {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 15px;
  opacity: 0.92;
}
.brand-feature-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.75);
  box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.12);
  flex-shrink: 0;
}

/* ===== 右栏：表单区 ===== */
.auth-main {
  flex: 1 1 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 32px;
  background: var(--bg-color, #fff);
  overflow-y: auto;
  animation: auth-fade-up 380ms cubic-bezier(0.22, 0.61, 0.36, 1) 90ms both;
}
.auth-form-area {
  width: 100%;
  max-width: 400px;
}

/* 移动端品牌条：默认隐藏，仅移动端展示 */
.auth-mobile-brand {
  display: none;
}

/* ===== 入场动效（fade-up，380ms） ===== */
@keyframes auth-fade-up {
  from {
    opacity: 0;
    transform: translateY(24px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-brand-panel,
  .auth-main,
  .brand-head,
  .brand-tagline,
  .brand-features {
    animation: none;
  }
}

/* ===== 移动端适配 ===== */
@media (max-width: 768px) {
  .auth-brand-panel {
    display: none;
  }
  .auth-main {
    padding: 28px 20px;
    justify-content: flex-start;
  }
  .auth-mobile-brand {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    margin-bottom: 28px;
  }
  .auth-mobile-logo {
    width: 56px;
    height: 56px;
    border-radius: 16px;
    background: var(--brand-gradient, linear-gradient(135deg, #667eea, #764ba2));
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 26px;
    font-weight: 800;
    box-shadow: 0 8px 24px rgba(102, 126, 234, 0.35);
  }
  .auth-mobile-name {
    font-size: 20px;
    font-weight: 700;
    color: var(--text-color, #1f2937);
  }
}
</style>
