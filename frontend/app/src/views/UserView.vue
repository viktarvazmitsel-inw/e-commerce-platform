<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useProfileStore } from '@/stores/profile'
import DeleteProfileModal from '@/components/DeleteProfileModal.vue'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const profileStore = useProfileStore()
const { userId, role } = storeToRefs(session)

const isAdmin = computed(() => (Number(role.value) & 4) === 4)

const loading = ref(true)
const errorMessage = ref('')
const profile = ref(null)
const deleteOpen = ref(false)

const lines = computed(() => {
  const data = profile.value
  if (!data) {
    return []
  }

  const fields = [
    { label: 'ID', value: data.id },
    { label: 'Email', value: data.email },
    { label: 'Name', value: data.name },
    { label: 'Surname', value: data.surname },
    { label: 'Phone', value: data.phone },
    { label: 'Role', value: data.role },
    { label: 'Active', value: data.isActive },
    { label: 'Verified', value: data.isVerified },
  ]

  return fields.filter((field) => hasValue(field.value))
})

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

async function loadUser(targetId) {
  loading.value = true
  errorMessage.value = ''
  profile.value = null

  try {
    const result = await profileStore.fetchById(targetId)
    if (route.params.userId !== targetId) {
      return
    }

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      profile.value = result.profile
      return
    }

    errorMessage.value = readError(result.payload) || 'Failed to load profile'
  } catch {
    if (route.params.userId !== targetId) {
      return
    }
    errorMessage.value = 'Failed to load profile'
  } finally {
    if (route.params.userId === targetId) {
      loading.value = false
    }
  }
}

async function onUserDeleted() {
  deleteOpen.value = false

  if (route.params.userId === userId.value) {
    session.clearSession()
    await router.push({ name: 'home' })
    return
  }

  if (profile.value) {
    profile.value = { ...profile.value, isActive: false }
  }
}

watch(
  () => route.params.userId,
  (targetId) => {
    deleteOpen.value = false
    if (typeof targetId === 'string' && targetId !== '') {
      loadUser(targetId)
    }
  },
  { immediate: true },
)
</script>

<template>
  <main class="user">
    <div v-if="!loading && profile" class="user-fields">
      <p v-for="line in lines" :key="line.label">{{ line.label }}: {{ line.value }}</p>
      <button
        v-if="isAdmin"
        type="button"
        class="delete-user"
        :disabled="profile.isActive !== true"
        @click="deleteOpen = true"
      >
        Delete User
      </button>
    </div>
    <p v-else-if="!loading && errorMessage" class="user-error">{{ errorMessage }}</p>

    <DeleteProfileModal
      :open="deleteOpen"
      :target-id="typeof route.params.userId === 'string' ? route.params.userId : ''"
      @close="deleteOpen = false"
      @success="onUserDeleted"
    />
  </main>
</template>

<style scoped>
.user {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  text-align: center;
}

.user-fields {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.delete-user {
  margin-top: 0.75rem;
  padding: 0.65rem 1rem;
  border: 0;
  border-radius: 4px;
  background: #c0392b;
  color: #fff;
  cursor: pointer;
}

.delete-user:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.user-error {
  color: #c0392b;
}
</style>
