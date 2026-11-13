<script setup>
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const form = reactive({
  email: '',
  password: '',
  confirmPassword: '',
  name: '',
  surname: '',
  phone: '',
})

const submitting = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

const passwordsMismatch = computed(
  () =>
    form.password.length > 0 &&
    form.confirmPassword.length > 0 &&
    form.password !== form.confirmPassword,
)

const canSubmit = computed(() => {
  const filled = [form.email, form.password, form.confirmPassword, form.name, form.surname, form.phone].every(
    (value) => value.trim() !== '',
  )

  return filled && form.password === form.confirmPassword && !submitting.value
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
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: form.email.trim(),
        password: form.password.trim(),
        name: form.name.trim(),
        surname: form.surname.trim(),
        phone: form.phone.trim(),
      }),
    })

    let payload = null
    try {
      payload = await response.json()
    } catch {
      payload = null
    }

    if (payload?.success === true) {
      await router.push({ name: 'check-email' })
      return
    }

    const error = readError(payload)
    if (error) {
      errorMessage.value = error.message
      errorDetails.value = error.details
      return
    }

    errorMessage.value = 'Registration failed'
  } catch {
    errorMessage.value = 'Registration failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="registration">
    <form class="registration-form" @submit.prevent="onSubmit">
      <h1>Registration</h1>

      <label for="email">Email</label>
      <input id="email" v-model="form.email" type="text" name="email" autocomplete="email" required />

      <label for="password">Password</label>
      <input
        id="password"
        v-model="form.password"
        type="password"
        name="password"
        autocomplete="new-password"
        required
      />

      <label for="confirm-password">Confirm password</label>
      <input
        id="confirm-password"
        v-model="form.confirmPassword"
        type="password"
        name="confirmPassword"
        autocomplete="new-password"
        required
      />
      <p v-if="passwordsMismatch" class="field-error">Passwords do not match</p>

      <label for="name">Name</label>
      <input id="name" v-model="form.name" type="text" name="name" autocomplete="given-name" required />

      <label for="surname">Surname</label>
      <input id="surname" v-model="form.surname" type="text" name="surname" autocomplete="family-name" required />

      <label for="phone">Phone</label>
      <input id="phone" v-model="form.phone" type="tel" name="phone" autocomplete="tel" required />

      <button type="submit" :disabled="!canSubmit">Register</button>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>
  </main>
</template>

<style scoped>
.registration {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.registration-form {
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
