<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { getBootstrap } from '../services/realtime.service'

interface MapItem {
  code: string
  name: string
  displayName: string
  emoji: string
  themeClass: string
  description: string
}

const maps = ref<MapItem[]>([
  {
    code: 'dark_village',
    name: 'Graveyard',
    displayName: 'Nghĩa Trang (Graveyard)',
    emoji: '🪦',
    themeClass: 'theme-graveyard',
    description: 'Một khu nghĩa trang cổ kính, u tối đầy rẫy bia mộ cổ và sương mù dày đặc bao phủ. Nơi các linh hồn lang thang tìm kiếm sự yên nghỉ và những điều bí ẩn đang chờ đợi kẻ thám hiểm gan dạ.'
  },
  {
    code: 'winter',
    name: 'Nivalis',
    displayName: 'Nivalis (Băng Giá)',
    emoji: '❄️',
    themeClass: 'theme-nivalis',
    description: 'Vùng cao nguyên tuyết trắng quanh năm lạnh giá. Gió bão buốt giá luôn rít qua các rặng thông phủ đầy tuyết trắng. Hãy sẵn sàng trang bị ấm áp trước khi bước vào nơi này!'
  },
  {
    code: 'village_adventure',
    name: 'Village',
    displayName: 'Thung Lũng Gỗ (Village)',
    emoji: '🏡',
    themeClass: 'theme-village',
    description: 'Ngôi làng trù phú, yên bình với những trảng cỏ xanh mướt, hàng rào gỗ quanh co và dòng suối trong vắt trôi lững lờ. Đây là điểm khởi đầu hoàn hảo cho mọi cuộc phiêu lưu.'
  }
])

const current = ref('')
const selectedCode = ref('')
const open = ref(false)
const imageErrors = ref<Record<string, boolean>>({})

onMounted(async () => {
  try {
    const data = await getBootstrap()
    current.value = data.map_code
  } catch { /* ignore */ }
})

const currentMapName = computed(() => {
  const m = maps.value.find(x => x.code === current.value)
  return m ? m.displayName : 'Chọn Bản Đồ'
})

const selectedMap = computed(() => {
  return maps.value.find(x => x.code === selectedCode.value) || maps.value[0]
})

function openModal() {
  selectedCode.value = current.value || maps.value[0].code
  open.value = true
}

function closeModal() {
  open.value = false
}

function selectMap(code: string) {
  selectedCode.value = code
}

function handleImageError(code: string) {
  imageErrors.value[code] = true
}

function handleConfirm() {
  const code = selectedCode.value
  if (!code) return
  
  if (code !== current.value) {
    window.dispatchEvent(new CustomEvent('game:switchMap', { detail: { mapCode: code } }))
    current.value = code
  }
  closeModal()
}
</script>

<template>
  <div class="map-switcher">
    <button class="pixel-button pixel-button--sm map-switcher__btn" @click="openModal">
      🗺️ {{ currentMapName }} ▾
    </button>

    <!-- Teleport modal to body to prevent rendering issues in game layout -->
    <Teleport to="body">
      <div v-if="open" class="modal-overlay" @click.self="closeModal">
        <div class="modal-box">
          <!-- Header -->
          <div class="modal-header">
            <h2 class="modal-title">CHỌN BẢN ĐỒ</h2>
            <button class="close-btn" @click="closeModal">×</button>
          </div>

          <!-- Body split layout -->
          <div class="modal-body">
            <!-- Left panel: Map list -->
            <div class="map-list">
              <button
                v-for="m in maps"
                :key="m.code"
                class="map-list-item"
                :class="{ 
                  active: m.code === selectedCode,
                  'current-active': m.code === current
                }"
                @click="selectMap(m.code)"
              >
                <span class="map-emoji">{{ m.emoji }}</span>
                <div class="map-item-info">
                  <span class="map-name">{{ m.displayName }}</span>
                  <span v-if="m.code === current" class="current-badge">Bản đồ hiện tại</span>
                </div>
              </button>
            </div>

            <!-- Right panel: Preview -->
            <div class="map-preview-panel">
              <div class="preview-card" :class="selectedMap.themeClass">
                <!-- Preview image or rich CSS/SVG fallback -->
                <div class="preview-media">
                  <img
                    v-if="!imageErrors[selectedMap.code]"
                    :src="`/assets/maps/${selectedMap.code}_preview.png`"
                    :alt="selectedMap.displayName"
                    class="preview-img"
                    @error="handleImageError(selectedMap.code)"
                  />
                  
                  <div v-else class="preview-fallback">
                    <div class="fallback-glow"></div>
                    <div class="fallback-art">{{ selectedMap.emoji }}</div>
                    <div class="fallback-title">{{ selectedMap.name }}</div>
                  </div>
                </div>
                
                <!-- Description -->
                <div class="preview-details">
                  <h3 class="preview-title">{{ selectedMap.displayName }}</h3>
                  <p class="preview-desc">{{ selectedMap.description }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Footer Actions -->
          <div class="modal-footer">
            <button class="pixel-button pixel-button--sm btn-cancel" @click="closeModal">Hủy</button>
            <button 
              class="pixel-button pixel-button--sm btn-confirm" 
              :disabled="selectedCode === current"
              @click="handleConfirm"
            >
              Đổi Bản Đồ
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.map-switcher {
  display: inline-block;
}

.map-switcher__btn {
  font-family: var(--pixel-font, inherit);
  display: flex;
  align-items: center;
  gap: 6px;
  background: linear-gradient(180deg, #d3c4a2 0%, #b8a67d 100%);
  border-color: #1a1c1e;
  color: #1a1c1e;
  cursor: pointer;
}

.map-switcher__btn:hover {
  background: linear-gradient(180deg, #e3d5b3 0%, #c9b78e 100%);
}

/* Modal Styling */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background-color: rgba(16, 22, 16, 0.75);
  backdrop-filter: blur(6px);
  z-index: 9999;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
}

.modal-box {
  background-color: var(--pixel-parchment, #f4ecd8);
  border: 4px solid var(--pixel-ink, #1a1c1e);
  box-shadow: 8px 8px 0 rgba(0, 0, 0, 0.4);
  border-radius: 4px;
  width: 100%;
  max-width: 780px;
  display: flex;
  flex-direction: column;
  max-height: 90vh;
  animation: modal-appear 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
  overflow: hidden;
}

@keyframes modal-appear {
  from {
    transform: scale(0.9) translateY(20px);
    opacity: 0;
  }
  to {
    transform: scale(1) translateY(0);
    opacity: 1;
  }
}

/* Header */
.modal-header {
  padding: 16px 20px;
  background: linear-gradient(180deg, var(--pixel-wood, #6d4c41) 0%, var(--pixel-wood-dark, #4e342e) 100%);
  border-bottom: 4px solid var(--pixel-ink, #1a1c1e);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-title {
  font-family: var(--pixel-font, inherit);
  color: var(--pixel-parchment, #f4ecd8);
  font-size: 24px;
  margin: 0;
  text-shadow: 2px 2px 0 var(--pixel-ink, #1a1c1e);
  letter-spacing: 1px;
}

.close-btn {
  background: none;
  border: none;
  font-family: var(--pixel-font, inherit);
  font-size: 28px;
  color: var(--pixel-parchment, #f4ecd8);
  cursor: pointer;
  line-height: 1;
  padding: 0;
  margin: 0;
  text-shadow: 2px 2px 0 var(--pixel-ink, #1a1c1e);
  transition: transform 0.1s;
}

.close-btn:hover {
  transform: scale(1.15);
  color: #ff8a80;
}

/* Body */
.modal-body {
  padding: 20px;
  display: flex;
  gap: 20px;
  flex: 1;
  overflow-y: auto;
  min-height: 320px;
}

/* Left panel */
.map-list {
  flex: 2;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.map-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #e6dbbf;
  border: 3px solid #8c7b64;
  border-radius: 4px;
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  transition: all 0.1s;
}

.map-list-item:hover {
  background: #ede3cb;
  border-color: var(--pixel-ink, #1a1c1e);
  transform: translateY(-2px);
  box-shadow: 0 4px 0 rgba(0, 0, 0, 0.1);
}

.map-list-item.active {
  background: #fff;
  border-color: #8bc34a;
  box-shadow: 0 0 0 2px #8bc34a;
}

.map-list-item.current-active {
  border-style: double;
}

.map-emoji {
  font-size: 24px;
}

.map-item-info {
  display: flex;
  flex-direction: column;
}

.map-name {
  font-weight: bold;
  font-size: 15px;
  color: var(--pixel-ink, #1a1c1e);
}

.current-badge {
  font-size: 11px;
  color: #27ae60;
  font-weight: bold;
  margin-top: 2px;
}

/* Right panel */
.map-preview-panel {
  flex: 3;
  display: flex;
}

.preview-card {
  width: 100%;
  border: 3px solid var(--pixel-ink, #1a1c1e);
  background: #ede3cb;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.preview-media {
  height: 180px;
  position: relative;
  overflow: hidden;
  background: #000;
  border-bottom: 3px solid var(--pixel-ink, #1a1c1e);
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* Fallback card styles matching map themes */
.preview-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  position: relative;
  overflow: hidden;
  color: #fff;
  text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
}

.theme-graveyard .preview-fallback {
  background: linear-gradient(135deg, #130f40 0%, #000000 100%);
}

.theme-nivalis .preview-fallback {
  background: linear-gradient(135deg, #74b9ff 0%, #2f3640 100%);
}

.theme-village .preview-fallback {
  background: linear-gradient(135deg, #2ecc71 0%, #1b3a1b 100%);
}

.fallback-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 120px;
  height: 120px;
  border-radius: 50%;
  filter: blur(24px);
  opacity: 0.6;
  animation: glow-pulse 2s infinite ease-in-out;
}

@keyframes glow-pulse {
  0%, 100% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 0.4;
  }
  50% {
    transform: translate(-50%, -50%) scale(1.2);
    opacity: 0.7;
  }
}

.theme-graveyard .fallback-glow {
  background: radial-gradient(circle, #9c27b0, transparent);
}

.theme-nivalis .fallback-glow {
  background: radial-gradient(circle, #00d2d3, transparent);
}

.theme-village .fallback-glow {
  background: radial-gradient(circle, #ffeb3b, transparent);
}

.fallback-art {
  font-size: 54px;
  margin-bottom: 8px;
  z-index: 1;
  animation: float-emoji 3s infinite ease-in-out;
}

@keyframes float-emoji {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-8px);
  }
}

.fallback-title {
  font-family: var(--pixel-font, inherit);
  font-size: 20px;
  font-weight: bold;
  letter-spacing: 1px;
  z-index: 1;
}

.preview-details {
  padding: 12px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.preview-title {
  font-family: var(--pixel-font, inherit);
  font-size: 18px;
  color: var(--pixel-ink, #1a1c1e);
  margin: 0;
}

.preview-desc {
  font-size: 12px;
  line-height: 1.5;
  color: #555;
  margin: 0;
}

/* Footer */
.modal-footer {
  padding: 16px 20px;
  border-top: 4px solid var(--pixel-ink, #1a1c1e);
  background: #ede3cb;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-cancel {
  background: linear-gradient(180deg, #e0e0e0 0%, #bdbdbd 100%);
  border-color: #1a1c1e;
  color: #1a1c1e;
}

.btn-confirm {
  background: linear-gradient(180deg, #8bc34a 0%, #689f38 100%);
  border-color: #1a1c1e;
  color: #fff;
  text-shadow: 1px 1px 0 #33691e;
}

.btn-confirm:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none !important;
  box-shadow: none !important;
}

/* Responsive adjustments */
@media (max-width: 680px) {
  .modal-body {
    flex-direction: column;
  }
  .map-list {
    flex: none;
  }
  .map-preview-panel {
    flex: none;
    height: 280px;
  }
}
</style>
