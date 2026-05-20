<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
await navigateTo('/projects', { replace: true })
</script>
<template><div /></template>
