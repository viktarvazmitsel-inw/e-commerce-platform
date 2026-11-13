<script setup>
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useNotificationsStore } from '@/stores/notifications'

const pageSize = 20

const router = useRouter()
const session = useSessionStore()
const notificationsStore = useNotificationsStore()
const { userId } = storeToRefs(session)

const loading = ref(true)
const errorMessage = ref('')
const notifications = ref([])
const page = ref(1)
const quantity = ref(pageSize)
const totalCount = ref(0)

const showList = computed(
  () => !loading.value && errorMessage.value === '' && notifications.value.length > 0,
)
const canGoPrevious = computed(() => page.value > 1)
const canGoNext = computed(() => page.value * quantity.value < totalCount.value)

let requestId = 0

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

async function loadNotifications(targetPage) {
  const currentRequest = ++requestId
  loading.value = true
  errorMessage.value = ''

  try {
    const result = await notificationsStore.list({ page: targetPage, quantity: pageSize })
    if (currentRequest !== requestId) {
      return
    }

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      notifications.value = result.notifications
      page.value = result.page
      quantity.value = result.quantity
      totalCount.value = result.totalCount
      return
    }

    notifications.value = []
    errorMessage.value = readError(result.payload) || 'Failed to load notifications'
  } catch {
    if (currentRequest !== requestId) {
      return
    }
    notifications.value = []
    errorMessage.value = 'Failed to load notifications'
  } finally {
    if (currentRequest === requestId) {
      loading.value = false
    }
  }
}

function goToPage(targetPage) {
  if (loading.value || targetPage < 1) {
    return
  }
  if (targetPage === page.value) {
    return
  }
  loadNotifications(targetPage)
}

loadNotifications(1)
</script>

<template>
  <main class="notifications">
    <template v-if="showList">
      <div class="notifications-table">
        <div class="notifications-row notifications-head">
          <span>Event</span>
          <span>Receiver</span>
          <span>Status</span>
          <span>Resend attempts</span>
        </div>
        <RouterLink
          v-for="notification in notifications"
          :key="notification.id"
          class="notifications-row"
          :to="{ name: 'notification', params: { notificationId: notification.id } }"
        >
          <span>{{ notification.event }}</span>
          <span>{{ notification.receiver }}</span>
          <span>{{ notification.status }}</span>
          <span>{{ notification.resendAttempts }}</span>
        </RouterLink>
      </div>
      <div class="notifications-pager">
        <button type="button" :disabled="!canGoPrevious" @click="goToPage(page - 1)">Previous</button>
        <button type="button" :disabled="!canGoNext" @click="goToPage(page + 1)">Next</button>
      </div>
    </template>
    <p v-else-if="!loading && errorMessage" class="notifications-error">{{ errorMessage }}</p>
    <p v-else-if="!loading" class="notifications-empty">No notifications found</p>
  </main>
</template>

<style scoped>
.notifications {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 2rem;
}

.notifications-table {
  width: min(100%, 56rem);
}

.notifications-row {
  display: grid;
  grid-template-columns: 1.4fr 1.6fr 0.7fr 0.8fr;
  gap: 0.75rem;
  align-items: center;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  text-decoration: none;
}

.notifications-head {
  color: var(--color-heading);
  font-weight: 600;
}

.notifications-pager {
  display: flex;
  gap: 0.5rem;
}

button {
  padding: 3px;
  border: 0;
  background: transparent;
  color: hsla(160, 100%, 37%, 1);
  font: inherit;
  cursor: pointer;
  transition: 0.4s;
}

button:disabled {
  color: var(--color-border);
  cursor: default;
}

@media (hover: hover) {
  button:not(:disabled):hover {
    background-color: hsla(160, 100%, 37%, 0.2);
  }
}

.notifications-error {
  color: #c0392b;
  text-align: center;
}

.notifications-empty {
  text-align: center;
}
</style>
