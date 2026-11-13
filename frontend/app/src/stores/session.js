import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useSessionStore = defineStore(
  'session',
  () => {
    const userId = ref('')
    const role = ref('')
    const accessToken = ref('')

    function setSession({ userId: nextUserId, role: nextRole, accessToken: nextAccessToken }) {
      userId.value = nextUserId
      role.value = nextRole
      accessToken.value = nextAccessToken
    }

    function clearSession() {
      userId.value = ''
      role.value = ''
      accessToken.value = ''
    }

    async function restoreAccess() {
      if (userId.value === '') {
        return
      }

      try {
        const response = await fetch('/api/auth/refresh-access', {
          method: 'POST',
          credentials: 'include',
        })

        let payload = null
        try {
          payload = await response.json()
        } catch {
          payload = null
        }

        const token = payload?.data?.access_token
        if (response.ok && payload?.success === true && typeof token === 'string' && token !== '') {
          accessToken.value = token
          return
        }

        clearSession()
      } catch {
        clearSession()
      }
    }

    return { userId, role, accessToken, setSession, clearSession, restoreAccess }
  },
  {
    persist: {
      storage: localStorage,
      pick: ['userId', 'role'],
    },
  },
)
