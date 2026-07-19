<script setup lang="ts">
import { reactive } from 'vue'

const emit = defineEmits<{ submit: [payload: { email: string; password: string }] }>()

const form = reactive({ email: '', password: '' })

function handleSubmit() {
  emit('submit', { ...form })
}
</script>

<template>
  <form class="auth-form" @submit.prevent="handleSubmit">
    <label class="pixel-field">
      <span class="pixel-field__label">Email</span>
      <input v-model="form.email" type="email" required autocomplete="email" />
    </label>
    <label class="pixel-field">
      <span class="pixel-field__label">Mật khẩu</span>
      <input v-model="form.password" type="password" required autocomplete="current-password" />
    </label>
    <slot name="error" />
    <button type="submit" class="pixel-button">Đăng nhập</button>
  </form>
</template>

<style scoped>
.auth-form {
  display: grid;
  gap: 14px;
}
</style>
