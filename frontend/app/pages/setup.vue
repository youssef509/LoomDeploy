<script setup lang="ts">
import { z } from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

definePageMeta({ layout: 'auth' })

const api = useApi()
const auth = useAuthStore()
const router = useRouter()
const toast = useToast()

// If already set up, bounce to login
onMounted(async () => {
  try {
    const { needs_setup } = await api.getSetupStatus()
    if (!needs_setup) router.replace('/login')
  } catch { /* stay */ }
})

const schema = z.object({
  email: z.string().email('Invalid email'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  confirm: z.string()
}).refine(d => d.password === d.confirm, {
  message: 'Passwords do not match',
  path: ['confirm']
})

type Schema = z.output<typeof schema>

const state = reactive({ email: '', password: '', confirm: '' })
const loading = ref(false)

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    await api.setupAdmin(event.data.email, event.data.password)
    await auth.login(event.data.email, event.data.password)
    toast.add({ title: 'Welcome to LoomDeploy!', description: 'Your admin account is ready.', color: 'success' })
    router.push('/')
  } catch (err: any) {
    toast.add({ title: 'Setup failed', description: err?.data?.message ?? 'Something went wrong', color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full max-w-sm px-4">
    <div class="text-center mb-8">
      <div class="flex items-center justify-center gap-2.5 mb-6">
        <div class="size-9 rounded-xl bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center shadow-md shadow-blue-500/30">
          <svg width="19" height="19" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="1.5" y="1.5" width="5.3" height="5.3" rx="1.3" fill="white"/>
            <rect x="8.2" y="1.5" width="5.3" height="5.3" rx="1.3" fill="white" fill-opacity="0.65"/>
            <rect x="1.5" y="8.2" width="5.3" height="5.3" rx="1.3" fill="white" fill-opacity="0.65"/>
            <rect x="8.2" y="8.2" width="5.3" height="5.3" rx="1.3" fill="white" fill-opacity="0.35"/>
          </svg>
        </div>
        <span class="text-xl font-bold text-highlighted tracking-tight">LoomDeploy</span>
      </div>

      <div class="inline-flex items-center gap-1.5 bg-violet-500/10 text-violet-400 border border-violet-500/20 rounded-full px-3 py-1 text-xs font-medium mb-4">
        <UIcon name="i-lucide-sparkles" class="size-3.5" />
        First-time setup
      </div>

      <h1 class="text-2xl font-bold text-highlighted">Create your admin account</h1>
      <p class="text-sm text-muted mt-1">This account will have full control over your LoomDeploy instance.</p>
    </div>

    <UCard>
      <UForm :schema="schema" :state="state" class="space-y-4" @submit="onSubmit">
        <UFormField label="Email" name="email">
          <UInput
            v-model="state.email"
            type="email"
            placeholder="admin@example.com"
            icon="i-lucide-mail"
            class="w-full"
            autofocus
          />
        </UFormField>

        <UFormField label="Password" name="password" hint="Minimum 8 characters">
          <UInput
            v-model="state.password"
            type="password"
            placeholder="••••••••"
            icon="i-lucide-lock"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Confirm Password" name="confirm">
          <UInput
            v-model="state.confirm"
            type="password"
            placeholder="••••••••"
            icon="i-lucide-lock-keyhole"
            class="w-full"
          />
        </UFormField>

        <UButton type="submit" :loading="loading" block class="mt-2" icon="i-lucide-rocket">
          Create Admin & Get Started
        </UButton>
      </UForm>
    </UCard>

    <p class="text-center text-xs text-muted mt-6">
      After setup, registration is closed — new users must be invited by an admin.
    </p>
  </div>
</template>
