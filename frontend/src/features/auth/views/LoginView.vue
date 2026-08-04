<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AuthCard from '../components/AuthCard.vue'
import LoginForm from '../components/LoginForm.vue'
import { useAuthStore } from '../stores/auth.store'
import { getTeamsSSOToken, initTeams, isRunningInTeams } from '@/features/teams/teams.service'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const errorMessage = ref('')
const teamsLoading = ref(false)

async function handleSubmit(payload: { email: string; password: string }) {
  errorMessage.value = ''
  try {
    await authStore.login(payload.email, payload.password)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.push(redirect)
  } catch {
    errorMessage.value = authStore.error
  }
}

async function handleTeamsLogin() {
  errorMessage.value = ''
  teamsLoading.value = true
  try {
    const ssoToken = await getTeamsSSOToken()
    await authStore.loginWithTeams(ssoToken)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.push(redirect)
  } catch {
    errorMessage.value = authStore.error
  } finally {
    teamsLoading.value = false
  }
}

onMounted(async () => {
  const inside = await initTeams()
  if (inside) {
    handleTeamsLogin()
  }
})
</script>

<template>
  <AuthCard>
    <template #title>Đăng nhập</template>
    <LoginForm @submit="handleSubmit">
      <template #error>
        <p v-if="errorMessage" class="pixel-alert pixel-alert--error">{{ errorMessage }}</p>
      </template>
      <template #extra-buttons>
        <button
          v-if="!isRunningInTeams()"
          type="button"
          class="pixel-button pixel-button--teams"
          :disabled="teamsLoading"
          @click="handleTeamsLogin"
        >
          {{ teamsLoading ? 'Đang đăng nhập...' : 'Đăng nhập bằng Teams' }}
        </button>
      </template>
    </LoginForm>
    <p class="pixel-link-row">
      Chưa có tài khoản?
      <router-link :to="{ name: 'register' }">Đăng ký</router-link>
    </p>
  </AuthCard>
</template>
