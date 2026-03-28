<template>
  <div class="table-skeleton">
    <n-skeleton v-for="i in rows" :key="i" text :repeat="1" style="margin-bottom: 12px" />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  rows?: number
}>()
</script>

<style scoped>
.table-skeleton {
  padding: 16px;
}
</style>
