<script setup>
import { RouterView } from 'vue-router'
import AppHeader from './components/AppHeader.vue'
import AppFooter from './components/AppFooter.vue'

async function testConnect() {
  const url = '/api/auth/livez'
  try {
    const response = await fetch(url)

    if (!response.ok) {
      throw new Error('failed to make a request')
    }

    const result = await response.json()
    console.log(result)
  } catch (error) {
    console.log(error.message)
  }
}

testConnect()
</script>

<template>
  <AppHeader />
  <main>
    <RouterView />
  </main>
  <AppFooter />
</template>
