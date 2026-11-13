<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const token = ref(typeof route.query.token === 'string' ? route.query.token : '')

const form = reactive({
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
  const filled = form.newPassword !== '' && form.confirmNewPassword !== ''
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

async function onSubmit(event) {
  if (!canSubmit.value) {
    return
  }

  const formToken = String(new FormData(event.target).get('token') ?? '').trim()

  errorMessage.value = ''
  errorDetails.value = []
  submitting.value = true

  try {
    const response = await fetch('/api/auth/confirm-password-recovery', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        new_password: form.newPassword,
        token: formToken,
      }),
    })

    let payload = null
    try {
      payload = await response.json()
    } catch {
      payload = null
    }

    if (payload?.success === true) {
      await router.push({ name: 'login' })
      return
    }

    const error = readError(payload)
    errorMessage.value = error?.message || 'Password recovery failed'
    errorDetails.value = error?.details ?? []
  } catch {
    errorMessage.value = 'Password recovery failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="password-recovery-verify">
    <form class="password-recovery-verify-form" @submit.prevent="onSubmit">
      <h1>Set a new password</h1>

      <input type="hidden" name="token" :value="token" />

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

      <button type="submit" :disabled="!canSubmit">Save</button>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.password-recovery-verify {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.password-recovery-verify-form {
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
