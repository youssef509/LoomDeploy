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
      <div class="flex items-center justify-center mb-4">
        <img src="/logo.png" alt="LoomDeploy" class="h-16 w-auto" />
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
