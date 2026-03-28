<template>
  <div class="pull-refresh" @touchstart="onTouchStart" @touchmove="onTouchMove" @touchend="onTouchEnd">
    <div class="pull-refresh-indicator" :style="{ transform: `translateY(${distance}px)` }">
      <n-spin v-if="loading" size="small" />
      <span v-else>{{ distance > 60 ? '释放刷新' : '下拉刷新' }}</span>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ refresh: [] }>()

const startY = ref(0)
const distance = ref(0)
const loading = ref(false)

const onTouchStart = (e: TouchEvent) => {
  if (window.scrollY === 0) {
    startY.value = e.touches[0].clientY
  }
}

const onTouchMove = (e: TouchEvent) => {
  if (startY.value && window.scrollY === 0) {
    const diff = e.touches[0].clientY - startY.value
    if (diff > 0) {
      distance.value = Math.min(diff * 0.5, 80)
    }
  }
}

const onTouchEnd = async () => {
  if (distance.value > 60) {
    loading.value = true
    emit('refresh')
    setTimeout(() => {
      loading.value = false
      distance.value = 0
    }, 1000)
  } else {
    distance.value = 0
  }
  startY.value = 0
}
</script>

<style scoped>
.pull-refresh {
  position: relative;
}
.pull-refresh-indicator {
  position: absolute;
  top: -40px;
  left: 50%;
  transform: translateX(-50%);
  transition: transform 0.3s;
  text-align: center;
}
</style>
