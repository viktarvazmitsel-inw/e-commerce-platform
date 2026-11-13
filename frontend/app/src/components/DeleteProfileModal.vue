<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useProfileStore } from '@/stores/profile'

const props = defineProps({
  open: {
    type: Boolean,
    required: true,
  },
  targetId: {
    type: String,
    required: true,
  },
})

const emit = defineEmits(['close', 'success'])

const router = useRouter()
const session = useSessionStore()
const profileStore = useProfileStore()
const { userId } = storeToRefs(session)

const form = reactive({
  password: '',
})

const submitting = ref(false)
const errorMessage = ref('')
const errorDetails = ref([])

const canSubmit = computed(() => form.password !== '' && props.targetId !== '' && !submitting.value)

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

function resetForm() {
  form.password = ''
  errorMessage.value = ''
  errorDetails.value = []
}

async function leaveIfSignedOut(sessionCleared) {
  if (!sessionCleared && userId.value !== '') {
    return false
  }

  await router.push({ name: 'home' })
  return true
}

function requestClose() {
  if (submitting.value) {
    return
  }

  emit('close')
}

function onBackdrop(event) {
  if (event.target === event.currentTarget) {
    requestClose()
  }
}

function onKeydown(event) {
  if (!props.open || event.key !== 'Escape') {
    return
  }

  requestClose()
}

async function onSubmit() {
  if (!canSubmit.value) {
    return
  }

  errorMessage.value = ''
  errorDetails.value = []
  submitting.value = true

  try {
    const result = await profileStore.softDelete({
      targetId: props.targetId,
      password: form.password,
    })

    if (await leaveIfSignedOut(result.sessionCleared)) {
      return
    }

    if (result.ok) {
      emit('success')
      return
    }

    const error = readError(result.payload)
    errorMessage.value = error?.message || 'Deletion failed'
    errorDetails.value = error?.details ?? []
  } catch {
    errorMessage.value = 'Deletion failed'
  } finally {
    submitting.value = false
  }
}

watch(
  () => props.open,
  () => {
    resetForm()
  },
)

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="backdrop" @click="onBackdrop">
      <div
        class="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-action-title"
      >
        <form class="dialog-form" @submit.prevent="onSubmit">
          <h2 id="confirm-action-title">Confirm action</h2>

          <input type="hidden" name="target_id" :value="targetId" />

          <label for="delete-profile-password">Password</label>
          <input
            id="delete-profile-password"
            v-model="form.password"
            type="password"
            name="password"
            autocomplete="current-password"
          />

          <div v-if="errorMessage" class="form-errors">
            <p>{{ errorMessage }}</p>
            <p v-for="detail in errorDetails" :key="detail">{{ detail }}</p>
          </div>

          <button type="submit" :disabled="!canSubmit">Confirm</button>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: rgba(0, 0, 0, 0.45);
}

.dialog {
  width: 100%;
  max-width: 24rem;
  padding: 1.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-background);
}

.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

h2 {
  margin-bottom: 0.5rem;
  font-size: 1.5rem;
  color: var(--color-heading);
}

label {
  margin-top: 0.25rem;
  font-size: 0.9rem;
}

input[type='password'] {
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
