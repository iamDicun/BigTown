<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import AppLayout from '@/app/layouts/AppLayout.vue'
import AuthLayout from '@/app/layouts/AuthLayout.vue'
import LoadingSplash from '@/shared/components/LoadingSplash.vue'
import { useAuthStore } from '@/features/auth/stores/auth.store'

const route = useRoute()
const authStore = useAuthStore()
const layout = computed(() => (route.meta.layout === 'auth' ? AuthLayout : AppLayout))
</script>

<template>
  <LoadingSplash v-if="!authStore.sessionReady" />
  <component :is="layout" v-else>
    <router-view />
  </component>
</template>
