<script setup lang="ts">
import type { FormSubmitEvent } from '@nuxt/ui'
import { z } from 'zod'

definePageMeta({ layout: 'auth', middleware: 'auth' })

const auth = useAuthStore()
const router = useRouter()
const toast = useToast()
const api = useApi()

onMounted(async () => {
  try {
    const { needs_setup } = await api.getSetupStatus()
    if (needs_setup) router.replace('/setup')
  } catch { /* backend unreachable, stay on login */ }
})

const schema = z.object({
  email: z.string().email('Invalid email'),
  password: z.string().min(1, 'Password is required')
})

type Schema = z.output<typeof schema>

const state = reactive<Schema>({
  email: '',
  password: ''
})

const loading = ref(false)

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    await auth.login(event.data.email, event.data.password)
    router.push('/')
  }
  catch (err: any) {
    toast.add({
      title: 'Login failed',
      description: err?.data?.message ?? 'Invalid credentials',
      color: 'error'
    })
  }
  finally {
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
      <h1 class="text-2xl font-bold text-highlighted">Welcome back</h1>
      <p class="text-sm text-muted mt-1">Sign in to your LoomDeploy instance</p>
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
          />
        </UFormField>

        <UFormField label="Password" name="password">
          <UInput
            v-model="state.password"
            type="password"
            placeholder="••••••••"
            icon="i-lucide-lock"
            class="w-full"
          />
        </UFormField>

        <UButton
          type="submit"
          :loading="loading"
          block
          class="mt-2"
        >
          Sign in
        </UButton>
      </UForm>
    </UCard>

    <p class="text-center text-xs text-muted mt-6">
      LoomDeploy &mdash; Self-hosted deployment platform
    </p>
  </div>
</template>
