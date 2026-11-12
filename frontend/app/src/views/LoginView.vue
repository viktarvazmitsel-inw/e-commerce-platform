<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'

const router = useRouter()
const session = useSessionStore()

const form = reactive({
  email: '',
  password: '',
})

const submitting = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

const canSubmit = computed(() => {
  const filled = form.email.trim() !== '' && form.password.trim() !== ''
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

async function onSubmit() {
  if (!canSubmit.value) {
    return
  }

  errorMessage.value = ''
  errorDetails.value = []
  submitting.value = true

  try {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: form.email.trim(),
        password: form.password,
      }),
    })

    let payload = null
    try {
      payload = await response.json()
    } catch {
      payload = null
    }

    if (payload?.success === true) {
      const data = payload.data ?? {}
      session.setSession({
        userId: typeof data.user_id === 'string' ? data.user_id : '',
        role: typeof data.role === 'string' ? data.role : '',
        accessToken: typeof data.access_token === 'string' ? data.access_token : '',
      })
      await router.push({ name: 'account' })
      return
    }

    const error = readError(payload)
    if (error) {
      errorMessage.value = error.message
      errorDetails.value = error.details
      return
    }

    errorMessage.value = 'Login failed'
  } catch {
    errorMessage.value = 'Login failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="login">
    <form class="login-form" @submit.prevent="onSubmit">
      <h1>Login</h1>

      <label for="email">Email</label>
      <input id="email" v-model="form.email" type="text" name="email" autocomplete="email" required />

      <label for="password">Password</label>
      <input
        id="password"
        v-model="form.password"
        type="password"
        name="password"
        autocomplete="current-password"
        required
      />

      <button type="submit" :disabled="!canSubmit">Login</button>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.login {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.login-form {
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

.form-errors {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-top: 0.25rem;
  color: #c0392b;
  font-size: 0.9rem;
}
</style>
