import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { login as loginApi } from '@/api'
import { TOKEN_KEY, USER_KEY } from '@/constants'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) ?? '')
  const username = ref(localStorage.getItem(USER_KEY) ?? '')

  const isAuthed = computed(() => !!token.value)

  function persist() {
    if (token.value) localStorage.setItem(TOKEN_KEY, token.value)
    else localStorage.removeItem(TOKEN_KEY)
    if (username.value) localStorage.setItem(USER_KEY, username.value)
    else localStorage.removeItem(USER_KEY)
  }

  function clearSession() {
    token.value = ''
    username.value = ''
    persist()
  }

  async function login(user: string, password: string) {
    const data = await loginApi(user, password)
    token.value = data.token
    username.value = data.username || user
    persist()
  }

  function logout() {
    clearSession()
  }

  return { token, username, isAuthed, login, logout, clearSession }
})
