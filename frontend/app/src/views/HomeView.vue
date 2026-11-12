<script setup>
import { RouterLink } from 'vue-router'
</script>

<template>
  <section class="home">
    <h1>E-COMMERCE PLATFORM</h1>
    <div class="actions">
      <RouterLink class="action" to="/login">Login</RouterLink>
      <RouterLink class="action" to="/registration">Register</RouterLink>
    </div>
  </section>
</template>

<style scoped>
.home {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1.5rem;
  padding: 2rem;
}

h1 {
  font-size: 2rem;
  letter-spacing: 0.04em;
  color: var(--color-heading);
  text-align: center;
}

.actions {
  display: flex;
  gap: 1rem;
}

.action {
  padding: 0.65rem 1.25rem;
  border-radius: 4px;
  background: hsla(160, 100%, 37%, 1);
  color: #fff;
}

.action:hover {
  background: hsla(160, 100%, 30%, 1);
  color: #fff;
}
</style>
