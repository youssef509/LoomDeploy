<script setup lang="ts">
import * as z from 'zod'
import type { FormError, FormSubmitEvent } from '@nuxt/ui'

const api = useApi()
const toast = useToast()

const passwordSchema = z.object({
  current: z.string().min(8, 'Must be at least 8 characters'),
  new: z.string().min(8, 'Must be at least 8 characters')
})

type PasswordSchema = z.output<typeof passwordSchema>

const password = reactive<Partial<PasswordSchema>>({ current: '', new: '' })
const saving = ref(false)

const validate = (state: Partial<PasswordSchema>): FormError[] => {
  const errors: FormError[] = []
  if (state.current && state.new && state.current === state.new) {
    errors.push({ name: 'new', message: 'New password must be different' })
  }
  return errors
}

async function onSubmit(event: FormSubmitEvent<PasswordSchema>) {
  saving.value = true
  try {
    await api.changePassword(event.data.current, event.data.new)
    toast.add({ title: 'Password updated', color: 'success' })
    password.current = ''
    password.new = ''
  } catch (err: any) {
    toast.add({ title: 'Failed', description: err?.data?.message, color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UPageCard
    title="Password"
    description="Confirm your current password before setting a new one."
    variant="subtle"
  >
    <UForm
      :schema="passwordSchema"
      :state="password"
      :validate="validate"
      class="flex flex-col gap-4 max-w-xs"
      @submit="onSubmit"
    >
      <UFormField name="current">
        <UInput v-model="password.current" type="password" placeholder="Current password" class="w-full" />
      </UFormField>
      <UFormField name="new">
        <UInput v-model="password.new" type="password" placeholder="New password" class="w-full" />
      </UFormField>
      <UButton label="Update" class="w-fit" type="submit" :loading="saving" />
    </UForm>
  </UPageCard>

  <UPageCard
    title="Account"
    description="No longer want to use our service? You can delete your account here. This action is not reversible. All information related to this account will be deleted permanently."
    class="bg-linear-to-tl from-error/10 from-5% to-default"
  >
    <template #footer>
      <UButton label="Delete account" color="error" />
    </template>
  </UPageCard>
</template>
