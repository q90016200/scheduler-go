import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { getMe, login as apiLogin, logout as apiLogout } from '../api/auth'
import type { User } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isLoggedIn = computed(() => user.value !== null)

  async function fetchMe() {
    try {
      user.value = await getMe()
    } catch {
      user.value = null
    }
  }

  async function login(payload: { user_name: string; password: string }) {
    await apiLogin(payload)
    await fetchMe()
  }

  async function logout() {
    await apiLogout()
    user.value = null
  }

  return { user, isLoggedIn, fetchMe, login, logout }
})
