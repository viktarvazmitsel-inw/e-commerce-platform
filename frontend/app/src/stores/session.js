import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useSessionStore = defineStore('session', () => {
  const userId = ref('')
  const role = ref('')
  const accessToken = ref('')

  function setSession({ userId: nextUserId, role: nextRole, accessToken: nextAccessToken }) {
    userId.value = nextUserId
    role.value = nextRole
    accessToken.value = nextAccessToken
  }

  return { userId, role, accessToken, setSession }
})
