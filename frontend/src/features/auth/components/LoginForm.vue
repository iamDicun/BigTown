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
    <label>
      Email
      <input v-model="form.email" type="email" required autocomplete="email" />
    </label>
    <label>
      Mật khẩu
      <input v-model="form.password" type="password" required autocomplete="current-password" />
    </label>
    <slot name="error" />
    <button type="submit">Đăng nhập</button>
  </form>
</template>

<style scoped>
.auth-form {
  display: grid;
  gap: 12px;
  min-width: 320px;
}

label {
  display: grid;
  gap: 4px;
}

input,
button {
  font: inherit;
  padding: 8px;
}
</style>
