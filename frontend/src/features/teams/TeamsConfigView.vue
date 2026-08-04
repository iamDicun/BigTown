<script setup lang="ts">
import { onMounted } from 'vue'
import { pages } from '@microsoft/teams-js'

onMounted(() => {
  const url = new URL(window.location.href)
  url.search = ''
  pages.config.setConfig({
    contentUrl: url.toString(),
    suggestedDisplayName: 'BigTown',
    websiteUrl: url.toString(),
  })
})
</script>

<template>
  <div class="pixel-alert">Đang cấu hình tab BigTown...</div>
</template>
