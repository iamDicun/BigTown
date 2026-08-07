<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'

const open = ref(false)

function onKey(e: KeyboardEvent) {
  if ((e.target as HTMLElement)?.tagName === 'INPUT') return
  if (e.key === 'h' || e.key === 'H') {
    open.value = !open.value
  }
  if (e.key === 'Escape' && open.value) {
    open.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <div class="help-overlay-wrapper">
    <!-- Nút Hint luôn hiện dưới settings button -->
    <div class="help-hint-tag" @click="open = !open">
      Bấm H: {{ open ? 'Đóng trợ giúp' : 'Trợ giúp' }}
    </div>

    <!-- Bảng phím tắt chi tiết -->
    <div v-if="open" class="help-list-panel">
      <div class="help-item"><span>W/A/S/D / ↑←↓→</span><span>Di chuyển</span></div>
      <div class="help-item"><span>Shift (giữ)</span><span>Chạy nhanh</span></div>
      <div class="help-item"><span>Enter</span><span>Mở chat</span></div>
      <div class="help-item"><span>Esc</span><span>Hủy / Đóng</span></div>
      <div class="help-item"><span>Q</span><span>Xóa vật phẩm</span></div>
      <div class="help-item"><span>E</span><span>Mở túi đồ</span></div>
      <div class="help-item"><span>1-5</span><span>Chọn ô nhanh</span></div>
      <div class="help-item"><span>H</span><span>Đóng bảng này</span></div>
    </div>
  </div>
</template>

<style scoped>
.help-overlay-wrapper {
  position: fixed;
  top: 50%;
  left: 16px;
  transform: translateY(-50%);
  z-index: 1008;
  pointer-events: auto;
  font-family: var(--pixel-font), monospace;
  image-rendering: pixelated;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.help-hint-tag {
  color: #ffffff;
  font-size: 18px;
  text-shadow: 2px 2px 0 #000000;
  cursor: pointer;
  user-select: none;
  width: max-content;
}

.help-hint-tag:hover {
  color: #ffe066;
}

.help-list-panel {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: #ffffff;
  font-size: 18px;
  text-shadow: 2px 2px 0 #000000;
  padding: 4px 0;
  width: max-content;
}

.help-item {
  display: flex;
  gap: 16px;
  justify-content: space-between;
}

.help-item span:first-child {
  color: #ffe066;
  min-width: 140px;
}
</style>
