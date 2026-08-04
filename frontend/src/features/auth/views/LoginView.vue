<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AuthCard from '../components/AuthCard.vue'
import LoginForm from '../components/LoginForm.vue'
import { useAuthStore } from '../stores/auth.store'
import LoadingSplash from '@/shared/components/LoadingSplash.vue'
import { getTeamsSSOToken, initTeams } from '@/features/teams/teams.service'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const errorMessage = ref('')

// Sau khi người dùng bấm "Đăng xuất" (Navbar đẩy ?loggedout=1), KHÔNG auto-login lại,
// nếu không logout trong Teams sẽ bị đá vào đăng nhập lại ngay lập tức.
const skipAutoLogin = route.query.loggedout === '1'

// Đoán sớm app có đang bị nhúng iframe (mở trong Teams) không, để:
//  - hiện overlay ngay khi sắp auto-login (tránh chớp form)
//  - biết có nên hiện nút "Đăng nhập bằng Teams" hay không (nút chỉ chạy được trong Teams)
const inTeams = ref(window.self !== window.top)
const teamsConnecting = ref(inTeams.value && !skipAutoLogin)

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
  teamsConnecting.value = true
  try {
    const ssoToken = await getTeamsSSOToken()
    await authStore.loginWithTeams(ssoToken)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.push(redirect)
    // Thành công thì điều hướng đi (query loggedout được bỏ), giữ overlay tới khi unmount.
  } catch {
    errorMessage.value = authStore.error
    teamsConnecting.value = false
  }
}

onMounted(async () => {
  const inside = await initTeams()
  inTeams.value = inside
  if (inside && !skipAutoLogin) {
    handleTeamsLogin()
  } else {
    teamsConnecting.value = false
  }
})
</script>

<template>
  <LoadingSplash v-if="teamsConnecting" label="Đang liên kết với tài khoản Teams…" />
  <AuthCard v-else>
    <template #title>Đăng nhập</template>
    <LoginForm @submit="handleSubmit">
      <template #error>
        <p v-if="errorMessage" class="pixel-alert pixel-alert--error">{{ errorMessage }}</p>
      </template>
    </LoginForm>
    <!-- Chỉ hiện khi thực sự chạy trong Teams (vd sau khi đăng xuất): nút này lấy lại SSO. -->
    <button
      v-if="inTeams"
      type="button"
      class="pixel-button pixel-button--teams"
      @click="handleTeamsLogin"
    >
      Đăng nhập bằng Teams
    </button>
    <p class="pixel-link-row">
      Chưa có tài khoản?
      <router-link :to="{ name: 'register' }">Đăng ký</router-link>
    </p>
  </AuthCard>
</template>