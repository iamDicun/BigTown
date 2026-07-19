<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

import { useGameStore } from '../stores/game.store'
import { createGameSocket, getDefaultRealtimeUrl } from '../network/gameSocket'
import type { PlayerChatEvent } from '../network/gameEvents'
import * as chatService from '../services/chat.service'
import type { ChatMessageDto } from '../services/chat.service'
import * as realtimeService from '../services/realtime.service'

type ChatMessage = {
  id: string
  characterId: string
  displayName: string
  message: string
  sentAt: string
  mine: boolean
}

const gameStore = useGameStore()
const messages = ref<ChatMessage[]>([])
const draft = ref('')
const sending = ref(false)
const status = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')
const error = ref('')
const messagesEl = ref<HTMLElement | null>(null)
const collapsed = ref(false)

const statusLabel: Record<typeof status.value, string> = {
  connecting: 'Đang kết nối',
  connected: 'Đã kết nối',
  disconnected: 'Mất kết nối',
  error: 'Lỗi',
}

const statusBadgeClass = computed(() => {
  if (status.value === 'connected') return 'pixel-badge--ok'
  if (status.value === 'connecting') return 'pixel-badge--warn'
  return 'pixel-badge--error'
})

let gameSocket: ReturnType<typeof createGameSocket> | null = null
let roomId = ''

const canSend = computed(() => !sending.value && draft.value.trim().length > 0)

function toggleCollapsed() {
  collapsed.value = !collapsed.value
  if (!collapsed.value) scrollToBottom()
}

onMounted(async () => {
  try {
    if (!gameStore.characterId) {
      await gameStore.loadMyCharacter()
    }
  } catch {
    // Không chặn chat nếu chưa load được character — "mine" chỉ tạm sai, vẫn nhận/gửi được tin.
  }

  let bootstrap: Awaited<ReturnType<typeof realtimeService.getBootstrap>>
  try {
    bootstrap = await realtimeService.getBootstrap()
    roomId = bootstrap.default_room_id
  } catch (err) {
    status.value = 'error'
    error.value = err instanceof Error ? err.message : 'Không thể lấy cấu hình realtime'
    return
  }

  try {
    const history = await chatService.getMessages(roomId)
    messages.value = history.map(toChatMessage)
    scrollToBottom()
  } catch {
    // Lỗi load lịch sử không nên chặn realtime connect ở dưới.
  }

  try {
    gameSocket = createGameSocket(getDefaultRealtimeUrl(), {
      channel: bootstrap.default_channel,
      onPlayerChat(event) {
        handlePlayerChat(event)
      },
    })

    gameSocket.centrifuge.on('connected', () => {
      status.value = 'connected'
      error.value = ''
    })
    gameSocket.centrifuge.on('disconnected', () => {
      status.value = 'disconnected'
    })
    gameSocket.centrifuge.on('error', (ctx) => {
      status.value = 'error'
      error.value = ctx.error.message
    })
  } catch (err) {
    status.value = 'error'
    error.value = err instanceof Error ? err.message : 'Không thể kết nối realtime'
  }
})

onBeforeUnmount(() => {
  gameSocket?.close()
  gameSocket = null
})

async function sendMessage() {
  const message = draft.value.trim()
  if (!message || sending.value) return

  sending.value = true
  try {
    await chatService.sendMessage(roomId, message)
    draft.value = ''
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Không thể gửi tin nhắn'
  } finally {
    sending.value = false
  }
}

function handlePlayerChat(event: PlayerChatEvent) {
  messages.value.push({
    id: `${event.characterId}-${event.sentAt}-${messages.value.length}`,
    characterId: event.characterId,
    displayName: event.displayName,
    message: event.message,
    sentAt: event.sentAt,
    mine: event.characterId === gameStore.characterId,
  })

  scrollToBottom()
}

function toChatMessage(dto: ChatMessageDto): ChatMessage {
  return {
    id: dto.id,
    characterId: dto.character_id,
    displayName: dto.character_name,
    message: dto.message,
    sentAt: dto.created_at,
    mine: dto.character_id === gameStore.characterId,
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}
</script>

<template>
  <section class="chat-panel" :class="{ collapsed }" aria-label="Game chat">
    <header>
      <span class="chat-panel__title">Chat</span>
      <div class="header-actions">
        <span class="pixel-badge" :class="statusBadgeClass">{{ statusLabel[status] }}</span>
        <button
          type="button"
          class="toggle-btn"
          :aria-expanded="!collapsed"
          :aria-label="collapsed ? 'Mở rộng khung chat' : 'Thu nhỏ khung chat'"
          @click="toggleCollapsed"
        >
          {{ collapsed ? '▸' : '▾' }}
        </button>
      </div>
    </header>
    <template v-if="!collapsed">
      <div ref="messagesEl" class="messages">
        <p v-if="error" class="pixel-alert pixel-alert--error">{{ error }}</p>
        <p v-if="messages.length === 0" class="empty">Mở thêm tab thứ hai rồi gửi thử một tin nhắn.</p>
        <article v-for="item in messages" :key="item.id" :class="['message', { mine: item.mine }]">
          <strong>{{ item.mine ? 'Bạn' : item.displayName }}</strong>
          <span>{{ item.message }}</span>
        </article>
      </div>
      <form class="chat-form pixel-field pixel-field--sm" @submit.prevent="sendMessage">
        <input v-model="draft" type="text" placeholder="Nhắn trong map...">
        <button type="submit" class="pixel-button pixel-button--sm" :disabled="!canSend">Gửi</button>
      </form>
    </template>
  </section>
</template>

<style scoped>
.chat-panel {
  min-height: 0;
  flex: 1 1 auto;
  display: grid;
  grid-template-rows: auto 1fr auto;
  background: var(--pixel-parchment);
  box-shadow:
    0 0 0 3px var(--pixel-wood-dark),
    0 0 0 6px var(--pixel-wood),
    0 0 0 8px var(--pixel-wood-dark),
    0 10px 20px rgba(0, 0, 0, 0.4);
}

.chat-panel.collapsed {
  flex: 0 0 auto;
  grid-template-rows: auto;
}

header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  background: linear-gradient(180deg, var(--pixel-wood) 0%, var(--pixel-wood-dark) 100%);
  border-bottom: 3px solid var(--pixel-ink);
}

.chat-panel__title {
  font-family: var(--pixel-font);
  font-size: 22px;
  letter-spacing: 1px;
  color: var(--pixel-parchment);
  text-shadow: 1px 1px 0 var(--pixel-ink);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.toggle-btn {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border: 2px solid var(--pixel-ink);
  background: var(--pixel-parchment);
  color: var(--pixel-wood-dark);
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
}

.toggle-btn:hover {
  background: var(--pixel-accent);
  color: #fff8ec;
}

.messages {
  min-height: 120px;
  overflow: auto;
  padding: 12px;
  font-family: var(--pixel-font);
  font-size: 18px;
}

.empty {
  margin: 0;
  color: var(--pixel-wood-dark);
  opacity: 0.75;
}

.message {
  display: grid;
  gap: 2px;
  margin-bottom: 8px;
  padding: 7px 9px;
  background: rgba(255, 255, 255, 0.5);
  border: 2px solid var(--pixel-parchment-dark);
}

.message.mine {
  background: rgba(90, 156, 74, 0.16);
  border-color: var(--pixel-green);
}

.message strong {
  color: var(--pixel-accent-dark);
  font-size: 15px;
  letter-spacing: 0.5px;
}

.message span {
  color: var(--pixel-ink);
  word-break: break-word;
  line-height: 1.2;
}

.chat-form {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  padding: 10px;
  background: var(--pixel-wood-dark);
}

.chat-form input {
  min-width: 0;
}
</style>
