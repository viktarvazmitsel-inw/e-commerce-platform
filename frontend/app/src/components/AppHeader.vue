<script setup>
import { RouterLink } from 'vue-router'
</script>

<template>
  <header class="site-header">
    <nav>
      <RouterLink to="/registration">Register</RouterLink>
      <RouterLink to="/login">Login</RouterLink>
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
</style>
