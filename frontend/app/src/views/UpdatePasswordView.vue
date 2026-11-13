<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useProfileStore } from '@/stores/profile'

const router = useRouter()
const session = useSessionStore()
const profileStore = useProfileStore()
const { userId } = storeToRefs(session)

const form = reactive({
  currentPassword: '',
  newPassword: '',
  confirmNewPassword: '',
})

const submitting = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

const passwordsMismatch = computed(
  () =>
    form.newPassword.length > 0 &&
    form.confirmNewPassword.length > 0 &&
    form.newPassword !== form.confirmNewPassword,
)

const canSubmit = computed(() => {
  const filled =
    form.currentPassword !== '' && form.newPassword !== '' && form.confirmNewPassword !== ''
  return filled && form.newPassword === form.confirmNewPassword && !submitting.value
})

function readError(payload) {
  const errors = payload?.errors
  if (!errors || typeof errors.message !== 'string' || errors.message === '') {
    return null
  }

  const details =
    errors.details && typeof errors.details === 'object'
      ? Object.values(errors.details).filter((item) => typeof item === 'string' && item !== '')
      : []

  return { message: errors.message, details }
}

async function leaveIfSignedOut(sessionCleared) {
  if (!sessionCleared && userId.value !== '') {
    return false
  }

  await router.push({ name: 'home' })
  return true
}

async function onSubmit() {
  if (!canSubmit.value) {
    return
  }

  errorMessage.value = ''
  errorDetails.value = []
  submitting.value = true

  try {
    const result = await profileStore.updatePassword({
      oldPassword: form.currentPassword,
      newPassword: form.newPassword,
    })

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      session.clearSession()
      await router.push({ name: 'login' })
      return
    }

    const error = readError(result.payload)
    errorMessage.value = error?.message || 'Password update failed'
    errorDetails.value = error?.details ?? []
  } catch {
    errorMessage.value = 'Password update failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="update-password">
    <form class="update-password-form" @submit.prevent="onSubmit">
      <h1>Change password</h1>

      <label for="current-password">Current password</label>
      <input
        id="current-password"
        v-model="form.currentPassword"
        type="password"
        name="current_password"
        autocomplete="current-password"
        required
      />

      <label for="new-password">New password</label>
      <input
        id="new-password"
        v-model="form.newPassword"
        type="password"
        name="new_password"
        autocomplete="new-password"
        required
      />

      <label for="confirm-new-password">Confirm new password</label>
      <input
        id="confirm-new-password"
        v-model="form.confirmNewPassword"
        type="password"
        name="confirm_new_password"
        autocomplete="new-password"
        required
      />
      <p v-if="passwordsMismatch" class="field-error">Passwords do not match</p>

      <button type="submit" :disabled="!canSubmit">Update</button>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.update-password {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.update-password-form {
  width: 100%;
  max-width: 24rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

h1 {
  margin-bottom: 0.5rem;
  font-size: 1.5rem;
  color: var(--color-heading);
}

label {
  margin-top: 0.25rem;
  font-size: 0.9rem;
}

input {
  padding: 0.55rem 0.7rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-background);
  color: var(--color-text);
}

button {
  margin-top: 0.75rem;
  padding: 0.65rem 1rem;
  border: 0;
  border-radius: 4px;
  background: hsla(160, 100%, 37%, 1);
  color: #fff;
  cursor: pointer;
}

button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.field-error,
.form-errors {
  color: #c0392b;
  font-size: 0.9rem;
}

.form-errors {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-top: 0.25rem;
}
</style>
