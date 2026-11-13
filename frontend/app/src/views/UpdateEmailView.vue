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
  newEmail: '',
  password: '',
})

const submitting = ref(false)
const sent = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

const canSubmit = computed(() => {
  const filled = form.newEmail.trim() !== '' && form.password !== ''
  return filled && !submitting.value
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
    const result = await profileStore.initiateEmailUpdate({
      newEmail: form.newEmail,
      password: form.password,
    })

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      sent.value = true
      return
    }

    const error = readError(result.payload)
    errorMessage.value = error?.message || 'Email update failed'
    errorDetails.value = error?.details ?? []
  } catch {
    errorMessage.value = 'Email update failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="update-email">
    <p v-if="sent" class="notice">Check your email</p>

    <form v-else class="update-email-form" @submit.prevent="onSubmit">
      <h1>Change email</h1>

      <label for="new-email">New email</label>
      <input
        id="new-email"
        v-model="form.newEmail"
        type="email"
        name="new_email"
        autocomplete="email"
        required
      />

      <label for="password">Password</label>
      <input
        id="password"
        v-model="form.password"
        type="password"
        name="password"
        autocomplete="current-password"
        required
      />

      <button type="submit" :disabled="!canSubmit">Update</button>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.update-email {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.update-email-form {
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

.notice {
  font-size: 1.5rem;
  color: var(--color-heading);
}

.form-errors {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-top: 0.25rem;
  color: #c0392b;
  font-size: 0.9rem;
}
</style>
