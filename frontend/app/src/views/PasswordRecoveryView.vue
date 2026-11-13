<script setup>
import { computed, reactive, ref } from 'vue'

const form = reactive({
  email: '',
})

const submitting = ref(false)
const sent = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

const canSubmit = computed(() => form.email.trim() !== '' && !submitting.value)

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

async function onSubmit() {
  if (!canSubmit.value) {
    return
  }

  errorMessage.value = ''
  errorDetails.value = []
  submitting.value = true

  try {
    const response = await fetch('/api/auth/initiate-password-recovery', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: form.email.trim(),
      }),
    })

    let payload = null
    try {
      payload = await response.json()
    } catch {
      payload = null
    }

    if (payload?.success === true) {
      sent.value = true
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
  <main class="password-recovery">
    <p v-if="sent" class="notice">Check your email</p>

    <form v-else class="password-recovery-form" @submit.prevent="onSubmit">
      <h1>Password recovery</h1>

      <label for="email">Email</label>
      <input id="email" v-model="form.email" type="email" name="email" autocomplete="email" required />

      <button type="submit" :disabled="!canSubmit">Recover</button>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.password-recovery {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.password-recovery-form {
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
