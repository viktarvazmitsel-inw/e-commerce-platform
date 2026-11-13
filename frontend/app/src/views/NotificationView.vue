<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useNotificationsStore } from '@/stores/notifications'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const notificationsStore = useNotificationsStore()
const { userId } = storeToRefs(session)

const loading = ref(true)
const errorMessage = ref('')
const notification = ref(null)

const lines = computed(() => {
  const data = notification.value
  if (!data) {
    return []
  }

  const fields = [
    { label: 'ID', value: data.id },
    { label: 'Event', value: data.event },
    { label: 'Receiver', value: data.receiver },
    ...payloadLines(data.payload),
    { label: 'Status', value: data.status },
    { label: 'Resend attempts', value: data.resendAttempts },
    { label: 'Created at', value: data.createdAt },
    { label: 'Updated at', value: data.updatedAt },
  ]

  return fields.filter((field) => hasValue(field.value))
})

function payloadLines(payload) {
  if (payload === null || typeof payload !== 'object' || Array.isArray(payload)) {
    return []
  }

  return Object.entries(payload).map(([key, value]) => ({
    label: `Payload.${key}`,
    value: displayPayloadValue(value),
  }))
}

function displayPayloadValue(value) {
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  if (value === null || value === undefined) {
    return ''
  }
  try {
    return JSON.stringify(value)
  } catch {
    return ''
  }
}

function hasValue(value) {
  if (typeof value === 'string') {
    return value !== ''
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return true
  }
  return false
}

function readError(payload) {
  const message = payload?.errors?.message
  if (typeof message === 'string' && message !== '') {
    return message
  }
  return null
}

async function leaveIfSignedOut(sessionCleared) {
  if (!sessionCleared && userId.value !== '') {
    return false
  }

  await router.push({ name: 'home' })
  return true
}

async function loadNotification(targetId) {
  loading.value = true
  errorMessage.value = ''
  notification.value = null

  try {
    const result = await notificationsStore.fetchById(targetId)
    if (route.params.notificationId !== targetId) {
      return
    }

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      notification.value = result.notification
      return
    }

    errorMessage.value = readError(result.payload) || 'Failed to load notification'
  } catch {
    if (route.params.notificationId !== targetId) {
      return
    }
    errorMessage.value = 'Failed to load notification'
  } finally {
    if (route.params.notificationId === targetId) {
      loading.value = false
    }
  }
}

watch(
  () => route.params.notificationId,
  (targetId) => {
    if (typeof targetId === 'string' && targetId !== '') {
      loadNotification(targetId)
    }
  },
  { immediate: true },
)
</script>

<template>
  <main class="notification">
    <div v-if="!loading && notification" class="notification-fields">
      <p v-for="line in lines" :key="line.label">{{ line.label }}: {{ line.value }}</p>
    </div>
    <p v-else-if="!loading && errorMessage" class="notification-error">{{ errorMessage }}</p>
  </main>
</template>

<style scoped>
.notification {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  text-align: center;
}

.notification-fields {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.notification-error {
  color: #c0392b;
}
</style>
