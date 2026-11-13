<script setup>
import { ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()

const token = ref(typeof route.query.token === 'string' ? route.query.token : '')
const submitting = ref(false)
const verified = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

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
  if (submitting.value || verified.value) {
    return
  }

  const formToken = String(new FormData(event.target).get('token') ?? '').trim()

  errorMessage.value = ''
  errorDetails.value = []
  submitting.value = true

  try {
    const response = await fetch('/api/auth/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: formToken }),
    })

    let payload = null
    try {
      payload = await response.json()
    } catch {
      payload = null
    }

    if (payload?.success === true) {
      verified.value = true
      return
    }

    const error = readError(payload)
    if (error) {
      errorMessage.value = error.message
      errorDetails.value = error.details
      return
    }

    errorMessage.value = 'Verification failed'
  } catch {
    errorMessage.value = 'Verification failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="verify">
    <form class="verify-form" @submit.prevent="onSubmit">
      <input type="hidden" name="token" :value="token" />

      <button v-if="!verified" type="submit" :disabled="submitting">Verify</button>
      <template v-else>
        <p class="success">Verification successful</p>
        <RouterLink class="login-link" :to="{ name: 'login' }">Login</RouterLink>
      </template>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.verify {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.verify-form {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

button {
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

.success {
  margin: 0;
  font-size: 1.25rem;
  color: hsla(160, 100%, 37%, 1);
}

.login-link {
  padding: 0.65rem 1rem;
  border-radius: 4px;
  background: hsla(160, 100%, 37%, 1);
  color: #fff;
}

.form-errors {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  color: #c0392b;
  font-size: 0.9rem;
}
</style>
