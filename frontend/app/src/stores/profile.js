import { ref } from 'vue'
import { defineStore } from 'pinia'
import { useSessionStore } from '@/stores/session'

function emptyProfile() {
  return {
    id: '',
    email: '',
    name: '',
    surname: '',
    phone: '',
    role: '',
    isActive: null,
    isVerified: null,
  }
}

function profileFromPayload(data) {
  return {
    id: typeof data?.id === 'string' ? data.id : '',
    email: typeof data?.email === 'string' ? data.email : '',
    name: typeof data?.name === 'string' ? data.name : '',
    surname: typeof data?.surname === 'string' ? data.surname : '',
    phone: typeof data?.phone === 'string' ? data.phone : '',
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

export const useProfileStore = defineStore('profile', () => {
  const profile = ref(emptyProfile())
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

  async function load(userId) {
    const { response, sessionCleared } = await authorizedFetch(
      `/api/auth/profile/${encodeURIComponent(userId)}`,
      { method: 'GET' },
    )
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      profile.value = profileFromPayload(payload.data)
      return { ok: true, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function fetchById(userId) {
    const { response, sessionCleared } = await authorizedFetch(
      `/api/auth/profile/${encodeURIComponent(userId)}`,
      { method: 'GET' },
    )
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return { ok: true, sessionCleared: false, payload, profile: profileFromPayload(payload.data) }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function update({ name, surname, phone }) {
    const next = {
      name: name.trim(),
      surname: surname.trim(),
      phone: phone.trim(),
    }

    const { response, sessionCleared } = await authorizedFetch('/api/auth/update-user-profile', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next),
    })
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      profile.value = {
        ...profile.value,
        name: next.name,
        surname: next.surname,
        phone: next.phone,
      }
      return { ok: true, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function initiateEmailUpdate({ newEmail, password }) {
    const { response, sessionCleared } = await authorizedFetch('/api/auth/initiate-email-update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        new_email: newEmail.trim(),
        password,
      }),
    })
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return { ok: true, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function updatePassword({ oldPassword, newPassword }) {
    const { response, sessionCleared } = await authorizedFetch('/api/auth/password-update', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        old_password: oldPassword,
        new_password: newPassword,
      }),
    })
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return { ok: true, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function softDelete({ targetId, password }, retried = false) {
    const response = await fetch('/api/auth/profile-soft-delete', {
      method: 'PATCH',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${session.accessToken}`,
      },
      body: JSON.stringify({
        target_id: targetId,
        password,
      }),
    })
    const payload = await readPayload(response)

    if (response.status === 401 && payload?.errors?.code === 'INVALID_CREDENTIALS') {
      return { ok: false, sessionCleared: false, payload }
    }

    if (response.status === 401 && !retried) {
      await session.restoreAccess()
      if (session.userId === '') {
        return { ok: false, sessionCleared: true, payload }
      }
      return softDelete({ targetId, password }, true)
    }

    if (session.userId === '') {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return { ok: true, sessionCleared: false, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function switchUserRole(userId, newRole) {
    const { response, sessionCleared } = await authorizedFetch(
      `/api/auth/switch-user-role/${encodeURIComponent(userId)}`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_role: newRole }),
      },
    )
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return { ok: true, sessionCleared: false, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  async function confirmEmailUpdate({ token }) {
    const { response, sessionCleared } = await authorizedFetch('/api/auth/confirm-email-update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token.trim() }),
    })
    const payload = await readPayload(response)

    if (sessionCleared) {
      return { ok: false, sessionCleared: true, payload }
    }

    if (response.ok && payload?.success === true) {
      return { ok: true, payload }
    }

    return { ok: false, sessionCleared: false, payload }
  }

  return {
    profile,
    load,
    fetchById,
    update,
    initiateEmailUpdate,
    updatePassword,
    softDelete,
    switchUserRole,
    confirmEmailUpdate,
  }
})
