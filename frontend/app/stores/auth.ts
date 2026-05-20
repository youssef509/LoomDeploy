import { defineStore } from 'pinia'
import type { AuthUser } from '~/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const token = useCookie<string | null>('ld_token', {
    maxAge: 60 * 60 * 24 * 7,
    sameSite: 'lax',
    secure: false
  })

  const isAuthenticated = computed(() => !!token.value)

  async function login(email: string, password: string) {
    const data = await $fetch<{ token: string, user: AuthUser }>('/api/auth/login', {
      method: 'POST',
      body: { email, password }
    })
    token.value = data.token
    user.value = data.user
  }

  async function fetchMe() {
    if (!token.value) return
    try {
      const data = await $fetch<AuthUser>('/api/auth/me', {
        headers: { Authorization: `Bearer ${token.value}` }
      })
      user.value = data
    }
    catch {
      token.value = null
      user.value = null
    }
  }

  async function logout() {
    token.value = null
    user.value = null
  }

  return { user, token, isAuthenticated, login, fetchMe, logout }
})
