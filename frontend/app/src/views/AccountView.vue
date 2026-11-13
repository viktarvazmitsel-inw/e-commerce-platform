<script setup>
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
const { userId, role } = storeToRefs(session)
</script>

<template>
  <main class="account">
    <h1>Account</h1>
    <p>User ID: {{ userId || '—' }}</p>
    <p>Role: {{ role || '—' }}</p>
  </main>
</template>

<style scoped>
.account {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
}

h1 {
  margin-bottom: 0.5rem;
  font-size: 1.5rem;
  color: var(--color-heading);
}
</style>
