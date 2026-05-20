<script setup lang="ts">
import { z } from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const toast = useToast()

const { data: users, refresh } = useAsyncData('users', () =>
  $fetch<{ id: string, email: string, role: string, created_at: string }[]>('/api/auth/users', {
    headers: { Authorization: `Bearer ${useAuthStore().token}` }
  }), { default: () => [] }
)

const isInviteOpen = ref(false)
const schema = z.object({
  email: z.string().email('Invalid email'),
  password: z.string().min(8, 'Min 8 characters'),
  role: z.enum(['admin', 'developer', 'viewer'] as const)
})
type Schema = z.output<typeof schema>
const form = reactive<Partial<Schema>>({ email: '', password: '', role: 'developer' })
const creating = ref(false)

async function onInvite(event: FormSubmitEvent<Schema>) {
  creating.value = true
  try {
    await $fetch('/api/auth/register', {
      method: 'POST',
      body: event.data,
      headers: { Authorization: `Bearer ${useAuthStore().token}` }
    })
    toast.add({ title: 'User created', color: 'success' })
    isInviteOpen.value = false
    refresh()
  }
  catch (err: any) {
    toast.add({ title: 'Failed', description: err?.data?.message, color: 'error' })
  }
  finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <UPageCard
      title="Members"
      description="Manage who has access to LoomDeploy."
      variant="naked"
      orientation="horizontal"
      class="mb-4"
    >
      <UButton label="Add user" icon="i-lucide-user-plus" color="neutral" class="w-fit lg:ms-auto" @click="isInviteOpen = true" />
    </UPageCard>

    <UPageCard variant="subtle">
      <div class="divide-y divide-default">
        <div
          v-for="user in users"
          :key="user.id"
          class="flex items-center justify-between py-3"
        >
          <div class="flex items-center gap-3">
            <UAvatar :alt="user.email" icon="i-lucide-user" size="sm" />
            <div>
              <p class="text-sm font-medium text-highlighted">{{ user.email }}</p>
              <p class="text-xs text-muted">Since {{ new Date(user.created_at).toLocaleDateString() }}</p>
            </div>
          </div>
          <UBadge variant="subtle" size="sm">{{ user.role }}</UBadge>
        </div>
        <div v-if="!users?.length" class="py-8 text-center text-sm text-muted">
          No users found.
        </div>
      </div>
    </UPageCard>
  </div>

  <UModal v-model:open="isInviteOpen" title="Add User">
    <template #body>
      <UForm :schema="schema" :state="form" class="p-4 space-y-4" @submit="onInvite">
        <UFormField label="Email" name="email">
          <UInput v-model="form.email" type="email" placeholder="user@example.com" class="w-full" />
        </UFormField>
        <UFormField label="Password" name="password">
          <UInput v-model="form.password" type="password" placeholder="••••••••" class="w-full" />
        </UFormField>
        <UFormField label="Role" name="role">
          <USelect v-model="form.role" :items="['admin','developer','viewer']" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton variant="ghost" color="neutral" @click="isInviteOpen = false">Cancel</UButton>
          <UButton type="submit" :loading="creating">Create User</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
