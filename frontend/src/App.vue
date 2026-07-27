<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppLayout from '@/app/layouts/AppLayout.vue'
import AuthLayout from '@/app/layouts/AuthLayout.vue'
import LoadingSplash from '@/shared/components/LoadingSplash.vue'
import { useAuthStore } from '@/features/auth/stores/auth.store'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const layout = computed(() => (route.meta.layout === 'auth' ? AuthLayout : AppLayout))

watch(() => authStore.sessionReady, (ready) => {
  if (ready) {
    router.replace(router.currentRoute.value.fullPath)
  }
})
</script>

<template>
  <LoadingSplash v-if="!authStore.sessionReady" />
  <component :is="layout" v-else>
    <router-view />
  </component>
</template>
