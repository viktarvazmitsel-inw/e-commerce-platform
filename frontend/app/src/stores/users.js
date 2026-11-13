import { defineStore } from 'pinia'
import { useSessionStore } from '@/stores/session'

function userFromPayload(data) {
  return {
    id: typeof data?.id === 'string' ? data.id : '',
    email: typeof data?.email === 'string' ? data.email : '',
    name: typeof data?.name === 'string' ? data.name : '',
    surname: typeof data?.surname === 'string' ? data.surname : '',
    role: data?.role ?? '',
    isActive: typeof data?.is_active === 'boolean' ? data.is_active : null,
    isVerified: typeof data?.is_verified === 'boolean' ? data.is_verified : null,
  }
}

async function readPayload(response) {
  try {
    return await response.json()
  } catch {
    return null
  }
}

export const useUsersStore = defineStore('users', () => {
  const session = useSessionStore()

  async function authorizedFetch(url, options, retried = false) {
    const response = await fetch(url, {
      ...options,
      credentials: 'include',
      headers: {
        ...options.headers,
        Authorization: `Bearer ${session.accessToken}`,
      },
    })

    if (response.status === 401 && !retried) {
      await session.restoreAccess()
      if (session.userId === '') {
        return { response, sessionCleared: true }
      }
      return authorizedFetch(url, options, true)
    }

    return { response, sessionCleared: session.userId === '' }
  }

  async function list({ page, quantity }) {
    const params = new URLSearchParams({
      page: String(page),
      quantity: String(quantity),
    })
    const { response, sessionCleared } = await authorizedFetch(
      `/api/auth/list-users?${params}`,
      { method: 'GET' },
    )
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      const data = payload.data ?? {}
      const users = Array.isArray(data.users) ? data.users.map(userFromPayload) : []

      return {
        ok: true,
        sessionCleared: false,
        payload,
        users,
        page: typeof data.page === 'number' ? data.page : page,
        quantity: typeof data.quantity === 'number' ? data.quantity : quantity,
        totalCount: typeof data.total_count === 'number' ? data.total_count : 0,
      }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  return { list }
})
