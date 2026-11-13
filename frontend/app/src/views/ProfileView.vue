<script setup>
import { computed, nextTick, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useProfileStore } from '@/stores/profile'
import DeleteProfileModal from '@/components/DeleteProfileModal.vue'

const router = useRouter()
const session = useSessionStore()
const profileStore = useProfileStore()
const { userId } = storeToRefs(session)
const { profile } = storeToRefs(profileStore)

const form = reactive({
  name: '',
  surname: '',
  phone: '',
})

const editing = reactive({
  name: false,
  surname: false,
  phone: false,
})

const nameInput = ref(null)
const surnameInput = ref(null)
const phoneInput = ref(null)

const inputs = {
  name: nameInput,
  surname: surnameInput,
  phone: phoneInput,
}

const loading = ref(true)
const submitting = ref(false)
const loaded = ref(false)
const deleteOpen = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])
const successMessage = ref('')

const canSubmit = computed(() => {
  const filled = form.name.trim() !== '' && form.phone.trim() !== ''
  return loaded.value && filled && !submitting.value && !loading.value
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

function applyProfileToForm() {
  form.name = profile.value.name
  form.surname = profile.value.surname
  form.phone = profile.value.phone
}

function lockFields() {
  editing.name = false
  editing.surname = false
  editing.phone = false
}

async function activate(field) {
  if (!loaded.value || loading.value || submitting.value || editing[field]) {
    return
  }

  editing[field] = true
  await nextTick()
  inputs[field].value?.focus()
}

async function leaveIfSignedOut(sessionCleared) {
  if (!sessionCleared && userId.value !== '') {
    return false
  }

  await router.push({ name: 'home' })
  return true
}

async function loadProfile() {
  loading.value = true
  errorMessage.value = ''
  errorDetails.value = []
  successMessage.value = ''

  try {
    const result = await profileStore.load(userId.value)
    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      applyProfileToForm()
      loaded.value = true
      return
    }

    const error = readError(result.payload)
    errorMessage.value = error?.message || 'Failed to load profile'
    errorDetails.value = error?.details ?? []
  } catch {
    errorMessage.value = 'Failed to load profile'
  } finally {
    loading.value = false
  }
}

async function onSubmit() {
  if (!canSubmit.value) {
    return
  }

  errorMessage.value = ''
  errorDetails.value = []
  successMessage.value = ''
  submitting.value = true

  try {
    const result = await profileStore.update({
      name: form.name,
      surname: form.surname,
      phone: form.phone,
    })

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      applyProfileToForm()
      lockFields()
      const message = result.payload?.data?.message
      successMessage.value = typeof message === 'string' && message !== '' ? message : 'Profile updated'
      return
    }

    const error = readError(result.payload)
    errorMessage.value = error?.message || 'Profile update failed'
    errorDetails.value = error?.details ?? []
  } catch {
    errorMessage.value = 'Profile update failed'
  } finally {
    submitting.value = false
  }
}

async function onProfileDeleted() {
  deleteOpen.value = false
  session.clearSession()
  await router.push({ name: 'home' })
}

loadProfile()
</script>

<template>
  <main class="profile">
    <form class="profile-form" @submit.prevent="onSubmit">
      <h1>Profile</h1>

      <div v-if="loaded" class="email-row">
        <div class="email-field">
          <span class="email-label">Email</span>
          <span class="email-value">{{ profile.email }}</span>
        </div>
        <div class="email-actions">
          <RouterLink class="change-email" :to="{ name: 'update-email' }">Change email</RouterLink>
          <RouterLink class="change-email" :to="{ name: 'update-password' }">Change password</RouterLink>
        </div>
      </div>

      <label for="name">Name</label>
      <input
        id="name"
        ref="nameInput"
        v-model="form.name"
        type="text"
        name="name"
        autocomplete="given-name"
        :readonly="!editing.name"
        @click="activate('name')"
      />

      <label for="surname">Surname</label>
      <input
        id="surname"
        ref="surnameInput"
        v-model="form.surname"
        type="text"
        name="surname"
        autocomplete="family-name"
        :readonly="!editing.surname"
        @click="activate('surname')"
      />

      <label for="phone">Phone</label>
      <input
        id="phone"
        ref="phoneInput"
        v-model="form.phone"
        type="tel"
        name="phone"
        autocomplete="tel"
        :readonly="!editing.phone"
        @click="activate('phone')"
      />

      <button type="submit" :disabled="!canSubmit">Update</button>

      <p v-if="successMessage" class="form-success">{{ successMessage }}</p>

      <div v-if="errorMessage" class="form-errors">
        <p>{{ errorMessage }}</p>
        <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
      </div>
    </form>

    <button v-if="loaded" type="button" class="delete-profile" @click="deleteOpen = true">
      Delete profile
    </button>

    <DeleteProfileModal
      :open="deleteOpen"
      :target-id="userId"
      @close="deleteOpen = false"
      @success="onProfileDeleted"
    />
  </main>
</template>

<style scoped>
.profile {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.profile-form {
  width: 100%;
  max-width: 24rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.delete-profile {
  width: 100%;
  max-width: 24rem;
  margin-top: 1.5rem;
  background: #c0392b;
}

h1 {
  margin-bottom: 0.5rem;
  font-size: 1.5rem;
  color: var(--color-heading);
}

.email-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.5rem;
}

.email-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.email-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
  flex-shrink: 0;
}

.email-label {
  font-size: 0.9rem;
}

.email-value {
  overflow-wrap: anywhere;
}

.change-email {
  padding: 0.45rem 0.7rem;
  border-radius: 4px;
  background: hsla(160, 100%, 37%, 1);
  color: #fff;
  font-size: 0.9rem;
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

input[readonly] {
  cursor: pointer;
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

.form-success {
  margin-top: 0.25rem;
  color: hsla(160, 100%, 37%, 1);
  font-size: 0.9rem;
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
