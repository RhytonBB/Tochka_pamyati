<script setup lang="ts">
import { onMounted } from 'vue';
import { useAuthStore } from './store/auth';
import ToastProvider from './components/ToastProvider.vue';

const auth = useAuthStore();

onMounted(async () => {
  if (!auth.initialized) {
    await auth.fetchMe();
  }
});
</script>

<template>
  <router-view />
  <ToastProvider />
</template>

<style>
/* Global styles are in style.css */
</style>
