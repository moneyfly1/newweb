<template>
  <div>
    <slot v-if="!hasError" />
    <div v-else class="error-boundary">
      <n-result status="error" title="出错了" description="页面遇到了一些问题">
        <template #footer>
          <n-button @click="handleReset">刷新页面</n-button>
        </template>
      </n-result>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'

const hasError = ref(false)

onErrorCaptured((err) => {
  console.error('Error caught:', err)
  hasError.value = true
  return false
})

const handleReset = () => {
  window.location.reload()
}
</script>

<style scoped>
.error-boundary {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}
</style>
