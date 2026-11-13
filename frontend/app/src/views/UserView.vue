<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'
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

const roleBits = [1, 2, 4]

const isAdmin = computed(() => (Number(role.value) & 4) === 4)

const loading = ref(true)
const errorMessage = ref('')
const profile = ref(null)
const deleteOpen = ref(false)
const draftRole = ref(0)
const roleUpdating = ref(false)
const roleUpdateError = ref('')
const roleUpdated = ref(false)

let roleOkTimer = 0

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

function clearRoleOkTimer() {
  if (roleOkTimer !== 0) {
    clearTimeout(roleOkTimer)
    roleOkTimer = 0
  }
}

function resetRoleEditor() {
  clearRoleOkTimer()
  roleUpdating.value = false
  roleUpdateError.value = ''
  roleUpdated.value = false
  draftRole.value = 0
}

function syncDraftRole(value) {
  const roleNumber = Number(value)
  draftRole.value = Number.isInteger(roleNumber) ? roleNumber : 0
}

function toggleRoleBit(bit, event) {
  if (bit === 4 || roleUpdating.value) {
    return
  }

  if (event.target.checked) {
    draftRole.value |= bit
    return
  }

  draftRole.value &= ~bit
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
  draftRole.value = 0

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
      syncDraftRole(result.profile.role)
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

async function updateUserRole() {
  const targetId = route.params.userId
  if (typeof targetId !== 'string' || targetId === '' || roleUpdating.value || roleUpdated.value) {
    return
  }

  const newRole = draftRole.value
  roleUpdating.value = true
  roleUpdateError.value = ''
  clearRoleOkTimer()

  try {
    const result = await profileStore.switchUserRole(targetId, newRole)
    if (route.params.userId !== targetId) {
      return
    }

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      if (profile.value) {
        profile.value = { ...profile.value, role: newRole }
      }
      draftRole.value = newRole
      roleUpdated.value = true
      roleOkTimer = window.setTimeout(() => {
        roleUpdated.value = false
        roleOkTimer = 0
      }, 2000)
      return
    }

    roleUpdateError.value = readError(result.payload) || 'Failed to update role'
  } catch {
    if (route.params.userId !== targetId) {
      return
    }
    roleUpdateError.value = 'Failed to update role'
  } finally {
    if (route.params.userId === targetId) {
      roleUpdating.value = false
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
    resetRoleEditor()
    if (typeof targetId === 'string' && targetId !== '') {
      loadUser(targetId)
    }
  },
  { immediate: true },
)

onUnmounted(() => {
  clearRoleOkTimer()
})
</script>

<template>
  <main class="user">
    <div v-if="!loading && profile" class="user-fields">
      <template v-for="line in lines" :key="line.label">
        <p v-if="!(isAdmin && line.label === 'Role')">{{ line.label }}: {{ line.value }}</p>
        <p v-else class="role-line">
          <span>Role:</span>
          <label v-for="bit in roleBits" :key="bit" class="role-bit">
            <input
              type="checkbox"
              :checked="(draftRole & bit) === bit"
              :disabled="bit === 4 || roleUpdating"
              @change="toggleRoleBit(bit, $event)"
            />
            {{ bit }}
          </label>
        </p>
      </template>
      <div v-if="isAdmin" class="role-actions">
        <p v-if="roleUpdated" class="role-ok">OK</p>
        <button
          v-else
          type="button"
          class="update-user"
          :disabled="roleUpdating"
          @click="updateUserRole"
        >
          Update user
        </button>
        <p v-if="roleUpdateError" class="user-error">{{ roleUpdateError }}</p>
      </div>
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

.role-line {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  margin: 0;
}

.role-bit {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.role-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.update-user {
  margin-top: 0.75rem;
  padding: 0.65rem 1rem;
  border: 0;
  border-radius: 4px;
  background: hsla(160, 100%, 37%, 1);
  color: #fff;
  cursor: pointer;
}

.update-user:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.role-ok {
  margin-top: 0.75rem;
  color: hsla(160, 100%, 37%, 1);
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
