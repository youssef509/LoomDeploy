<script setup lang="ts">
const auth = useAuthStore()
const colorMode = useColorMode()
</script>

<template>
  <div class="space-y-6">
    <UPageCard title="Account" description="Your LoomDeploy account details." variant="subtle">
      <dl class="space-y-4 text-sm">
        <div class="flex justify-between items-center">
          <dt class="text-muted font-medium">Email</dt>
          <dd class="font-mono text-highlighted">{{ auth.user?.email }}</dd>
        </div>
        <USeparator />
        <div class="flex justify-between items-center">
          <dt class="text-muted font-medium">Role</dt>
          <dd>
            <UBadge variant="subtle" size="sm">{{ auth.user?.role }}</UBadge>
          </dd>
        </div>
        <USeparator />
        <div class="flex justify-between items-center">
          <dt class="text-muted font-medium">Member since</dt>
          <dd class="text-muted">{{ auth.user?.created_at ? new Date(auth.user.created_at).toLocaleDateString() : '—' }}</dd>
        </div>
      </dl>
    </UPageCard>

    <UPageCard title="Appearance" description="Customize how LoomDeploy looks." variant="subtle">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-highlighted">Color mode</p>
          <p class="text-xs text-muted mt-0.5">Switch between light and dark</p>
        </div>
        <div class="flex items-center gap-2">
          <UButton
            :variant="colorMode.value === 'light' ? 'solid' : 'outline'"
            size="sm"
            icon="i-lucide-sun"
            color="neutral"
            @click="colorMode.preference = 'light'"
          >
            Light
          </UButton>
          <UButton
            :variant="colorMode.value === 'dark' ? 'solid' : 'outline'"
            size="sm"
            icon="i-lucide-moon"
            color="neutral"
            @click="colorMode.preference = 'dark'"
          >
            Dark
          </UButton>
        </div>
      </div>
    </UPageCard>

    <UPageCard title="Server Info" description="Your LoomDeploy server details." variant="subtle">
      <dl class="space-y-4 text-sm font-mono">
        <div class="flex justify-between items-center">
          <dt class="text-muted">Backend API</dt>
          <dd class="text-highlighted">localhost:8080</dd>
        </div>
        <USeparator />
        <div class="flex justify-between items-center">
          <dt class="text-muted">Docker network</dt>
          <dd class="text-highlighted">paas_network</dd>
        </div>
        <USeparator />
        <div class="flex justify-between items-center">
          <dt class="text-muted">Database</dt>
          <dd class="text-highlighted">/var/lib/loomdeploy/data.db</dd>
        </div>
      </dl>
    </UPageCard>
  </div>
</template>
