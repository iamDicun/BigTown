<script setup lang="ts">
import { useAuthStore } from '@/features/auth/stores/auth.store'

const authStore = useAuthStore()
</script>

<template>
  <section>
    <h1>Dashboard</h1>
    <p>Bạn đã đăng nhập với role: <strong>{{ authStore.role }}</strong></p>
    <p>
      Đây là feature boilerplate để chứng minh route bảo vệ (<code>meta.requiresAuth</code>) và
      Pinia store hoạt động thật với backend. Copy cấu trúc <code>features/auth</code> sang các
      thư mục feature còn trống (<code>features/devices</code>, <code>features/employees</code>,
      ...) để implement nghiệp vụ thật, theo đúng thứ tự ở mục 23 của
      <code>frontend/frontend-architecture-it-asset-tracking.md</code>.
    </p>
  </section>
</template>
