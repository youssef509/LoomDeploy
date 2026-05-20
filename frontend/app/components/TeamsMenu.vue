<script setup lang="ts">
defineProps<{
  collapsed?: boolean
}>()
</script>

<template>
  <NuxtLink to="/" class="flex items-center gap-2.5 px-2 py-1.5 min-w-0">
    <img src="/logo.png" alt="LoomDeploy" class="size-7 shrink-0 rounded-md" />
    <span
      v-if="!collapsed"
      class="font-semibold text-sm text-highlighted truncate"
    >
      LoomDeploy
    </span>
  </NuxtLink>
</template>
