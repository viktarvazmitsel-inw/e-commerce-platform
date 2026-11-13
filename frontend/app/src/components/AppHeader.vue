<script setup>
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const { userId, accessToken } = storeToRefs(session)

const isAuthenticated = computed(() => userId.value !== '')

const navLinks = computed(() => {
  const links = isAuthenticated.value
    ? [{ name: 'profile', label: 'Profile' }]
    : [
        { name: 'registration', label: 'Register' },
        { name: 'login', label: 'Login' },
      ]

  return links.filter((link) => link.name !== route.name)
})

async function logout() {
  try {
    await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
      headers: {
        Authorization: `Bearer ${accessToken.value}`,
      },
    })
  } catch {
    // A failed logout request still ends the local session.
  } finally {
    session.clearSession()
    await router.push({ name: 'home' })
  }
}
</script>

<template>
  <header class="site-header">
    <nav>
      <RouterLink
        v-for="link in navLinks"
        :key="link.name"
        :to="{ name: link.name }"
      >
        {{ link.label }}
      </RouterLink>
      <button v-if="isAuthenticated" type="button" @click="logout">Logout</button>
    </nav>
  </header>
</template>

<style scoped>
.site-header {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding: 0.75rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

nav {
  display: flex;
  gap: 0.5rem;
}

button {
  padding: 3px;
  border: 0;
  background: transparent;
  color: hsla(160, 100%, 37%, 1);
  font: inherit;
  cursor: pointer;
  transition: 0.4s;
}

@media (hover: hover) {
  button:hover {
    background-color: hsla(160, 100%, 37%, 0.2);
  }
}
</style>
