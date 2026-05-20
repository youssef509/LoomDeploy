<script setup lang="ts">
defineProps<{
  collapsed?: boolean
}>()
</script>

<template>
  <NuxtLink
    to="/"
    :class="collapsed
      ? 'flex items-center justify-center w-full py-2 min-w-0 group'
      : 'flex items-center px-2 py-2 min-w-0 group'"
  >
    <!-- Collapsed: fill the full header width -->
    <img v-if="collapsed" src="/logo-tg.png" alt="LoomDeploy" class="w-full h-auto max-h-30 object-contain" />
    <!-- Expanded: logo.png -->
    <img v-else src="/logo.png" alt="LoomDeploy" class="h-25 w-auto flex-none" />
  </NuxtLink>
</template>
