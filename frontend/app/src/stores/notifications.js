import { defineStore } from 'pinia'
import { useSessionStore } from '@/stores/session'

function notificationFromPayload(data) {
  return {
    id: typeof data?.id === 'string' ? data.id : '',
    event: typeof data?.event === 'string' ? data.event : '',
    receiver: typeof data?.receiver === 'string' ? data.receiver : '',
    status: typeof data?.status === 'string' ? data.status : '',
    resendAttempts: typeof data?.resend_attempts === 'number' ? data.resend_attempts : 0,
  }
}

function notificationDetailFromPayload(data) {
  const payload = data?.payload
  return {
    ...notificationFromPayload(data),
    payload:
      payload !== null && typeof payload === 'object' && !Array.isArray(payload) ? payload : {},
    createdAt: typeof data?.created_at === 'string' ? data.created_at : '',
    updatedAt: typeof data?.updated_at === 'string' ? data.updated_at : '',
  }
}

async function readPayload(response) {
  try {
    return await response.json()
  } catch {
    return null
  }
}

export const useNotificationsStore = defineStore('notifications', () => {
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
      `/api/note/notifications?${params}`,
      { method: 'GET' },
    )
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      const data = payload.data ?? {}
      const notifications = Array.isArray(data.notifications)
        ? data.notifications.map(notificationFromPayload)
        : []

      return {
        ok: true,
        sessionCleared: false,
        payload,
        notifications,
        page: typeof data.page === 'number' ? data.page : page,
        quantity: typeof data.quantity === 'number' ? data.quantity : quantity,
        totalCount: typeof data.total_count === 'number' ? data.total_count : 0,
      }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function fetchById(notificationId) {
    const { response, sessionCleared } = await authorizedFetch(
      `/api/note/notifications/${encodeURIComponent(notificationId)}`,
      { method: 'GET' },
    )
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return {
        ok: true,
        sessionCleared: false,
        payload,
        notification: notificationDetailFromPayload(payload.data),
      }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  return { list, fetchById }
})
