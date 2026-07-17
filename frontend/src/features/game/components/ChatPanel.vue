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
  <section class="panel chat-panel" :class="{ collapsed }" aria-label="Game chat">
    <header>
      <span>Chat</span>
      <div class="header-actions">
        <small :class="['status', status]">{{ status }}</small>
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
        <p v-if="error" class="error">{{ error }}</p>
        <p v-if="messages.length === 0" class="empty">Mở thêm tab thứ hai rồi gửi thử một tin nhắn.</p>
        <article v-for="item in messages" :key="item.id" :class="['message', { mine: item.mine }]">
          <strong>{{ item.mine ? 'Bạn' : item.displayName }}</strong>
          <span>{{ item.message }}</span>
        </article>
      </div>
      <form class="chat-form" @submit.prevent="sendMessage">
        <input v-model="draft" type="text" placeholder="Nhắn trong map...">
        <button type="submit" :disabled="!canSend">Gửi</button>
      </form>
    </template>
  </section>
</template>

<style scoped>
.panel {
  border: 1px solid rgba(159, 212, 127, 0.24);
  border-radius: 14px;
  background: rgba(16, 22, 16, 0.86);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.28);
  backdrop-filter: blur(12px);
}

.chat-panel {
  min-height: 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
}

.chat-panel.collapsed {
  grid-template-rows: auto;
  /* GameView.vue đặt ChatPanel trong grid row 1fr — mặc định grid item stretch full chiều cao
     track đó. Khi collapsed, panel chỉ nên cao bằng header, không kéo dài khung bọc xuống hết
     phần trống của track. */
  align-self: start;
}

header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid rgba(159, 212, 127, 0.18);
  font-weight: 700;
}

.chat-panel.collapsed header {
  border-bottom: none;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toggle-btn {
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #c9c1aa;
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
}

.toggle-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #f3ead7;
}

.status {
  color: #c9c1aa;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}

.status.connected {
  color: #9fd47f;
}

.status.error {
  color: #ffb4a8;
}

.messages {
  min-height: 120px;
  overflow: auto;
  padding: 14px;
  color: #c9c1aa;
}

.empty,
.error {
  margin: 0;
  font-size: 13px;
}

.error {
  color: #ffb4a8;
}

.message {
  display: grid;
  gap: 4px;
  margin-bottom: 10px;
  padding: 9px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.06);
}

.message.mine {
  background: rgba(159, 212, 127, 0.14);
}

.message strong {
  color: #9fd47f;
  font-size: 12px;
}

.message span {
  color: #f3ead7;
  word-break: break-word;
}

.chat-form {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  padding: 12px;
}

input,
button {
  min-width: 0;
  border: 1px solid #566744;
  border-radius: 8px;
  padding: 9px 10px;
  background: #172017;
  color: #f3ead7;
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
