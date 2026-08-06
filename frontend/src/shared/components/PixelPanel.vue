<script setup lang="ts">
defineSlots<{ default(): any; title?(): any }>()
</script>

<template>
  <div class="ui-panel">
    <span v-if="$slots.title" class="ui-banner"><slot name="title" /></span>
    <slot />
  </div>
</template>
