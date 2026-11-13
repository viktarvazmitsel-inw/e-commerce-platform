<script setup>
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useUsersStore } from '@/stores/users'

const pageSize = 20

const router = useRouter()
const session = useSessionStore()
const usersStore = useUsersStore()
const { userId, role } = storeToRefs(session)

const isAdmin = computed(() => (Number(role.value) & 4) === 4)

const loading = ref(isAdmin.value)
const errorMessage = ref('')
const users = ref([])
const page = ref(1)
const quantity = ref(pageSize)
const totalCount = ref(0)

const showList = computed(() => !loading.value && errorMessage.value === '' && users.value.length > 0)
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

async function loadUsers(targetPage) {
  if (!isAdmin.value) {
    loading.value = false
    users.value = []
    errorMessage.value = ''
    return
  }

  const currentRequest = ++requestId
  loading.value = true
  errorMessage.value = ''

  try {
    const result = await usersStore.list({ page: targetPage, quantity: pageSize })
    if (currentRequest !== requestId) {
      return
    }

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      users.value = result.users
      page.value = result.page
      quantity.value = result.quantity
      totalCount.value = result.totalCount
      return
    }

    users.value = []
    errorMessage.value = readError(result.payload) || 'Failed to load users'
  } catch {
    if (currentRequest !== requestId) {
      return
    }
    users.value = []
    errorMessage.value = 'Failed to load users'
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
  loadUsers(targetPage)
}

loadUsers(1)
</script>

<template>
  <main class="users">
    <template v-if="showList">
      <div class="users-table">
        <div class="users-row users-head">
          <span>Email</span>
          <span>Name</span>
          <span>Surname</span>
          <span>Role</span>
          <span>Active</span>
          <span>Verified</span>
        </div>
        <RouterLink
          v-for="user in users"
          :key="user.id"
          class="users-row"
          :to="{ name: 'user', params: { userId: user.id } }"
        >
          <span>{{ user.email }}</span>
          <span>{{ user.name }}</span>
          <span>{{ user.surname }}</span>
          <span>{{ user.role }}</span>
          <span>{{ user.isActive }}</span>
          <span>{{ user.isVerified }}</span>
        </RouterLink>
      </div>
      <div class="users-pager">
        <button type="button" :disabled="!canGoPrevious" @click="goToPage(page - 1)">Previous</button>
        <button type="button" :disabled="!canGoNext" @click="goToPage(page + 1)">Next</button>
      </div>
    </template>
    <p v-else-if="!loading && errorMessage" class="users-error">{{ errorMessage }}</p>
    <p v-else-if="!loading" class="users-empty">No users found</p>
  </main>
</template>

<style scoped>
.users {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 2rem;
}

.users-table {
  width: min(100%, 56rem);
}

.users-row {
  display: grid;
  grid-template-columns: 1.6fr 1fr 1fr 0.5fr 0.6fr 0.6fr;
  gap: 0.75rem;
  align-items: center;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  text-decoration: none;
}

.users-head {
  color: var(--color-heading);
  font-weight: 600;
}

.users-pager {
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

.users-error {
  color: #c0392b;
  text-align: center;
}

.users-empty {
  text-align: center;
}
</style>
