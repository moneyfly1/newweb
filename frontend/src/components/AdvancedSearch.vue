<template>
  <n-drawer v-model:show="visible" :width="400" placement="right">
    <n-drawer-content title="高级搜索">
      <n-form>
        <slot />
      </n-form>
      <template #footer>
        <n-space>
          <n-button @click="handleReset">重置</n-button>
          <n-button type="primary" @click="handleSearch">搜索</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const visible = ref(false)
const emit = defineEmits<{
  search: []
  reset: []
}>()

const open = () => {
  visible.value = true
}

const handleSearch = () => {
  emit('search')
  visible.value = false
}

const handleReset = () => {
  emit('reset')
}

defineExpose({ open })
</script>
