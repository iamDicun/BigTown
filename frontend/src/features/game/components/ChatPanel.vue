<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

import { useAuthStore } from '@/features/auth/stores/auth.store'

import { createGameSocket, defaultGameChannel, getDefaultRealtimeUrl } from '../network/gameSocket'
import type { GameClientEvent, PlayerChatEvent } from '../network/gameEvents'

type ChatMessage = {
  id: string
  characterId: string
  message: string
  sentAt: string
  mine: boolean
}

const authStore = useAuthStore()
const messages = ref<ChatMessage[]>([])
const draft = ref('')
const status = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')
const error = ref('')
const messagesEl = ref<HTMLElement | null>(null)

let gameSocket: ReturnType<typeof createGameSocket> | null = null

const canSend = computed(() => status.value === 'connected' && draft.value.trim().length > 0)

onMounted(() => {
  try {
    gameSocket = createGameSocket(getDefaultRealtimeUrl(), {
      channel: defaultGameChannel,
      onEvent(event) {
        handleRealtimeEvent(event)
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
  if (!message || !gameSocket) return

  const event: PlayerChatEvent = {
    type: 'player_chat',
    characterId: authStore.userId ?? 'unknown',
    message,
    sentAt: new Date().toISOString(),
  }

  await gameSocket.send(event satisfies GameClientEvent)
  draft.value = ''
}

function handleRealtimeEvent(event: unknown) {
  if (!isPlayerChatEvent(event)) return

  messages.value.push({
    id: `${event.characterId}-${event.sentAt}-${messages.value.length}`,
    characterId: event.characterId,
    message: event.message,
    sentAt: event.sentAt,
    mine: event.characterId === authStore.userId,
  })

  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}

function isPlayerChatEvent(event: unknown): event is PlayerChatEvent {
  if (!event || typeof event !== 'object') return false

  const candidate = event as Partial<PlayerChatEvent>
  return candidate.type === 'player_chat' && typeof candidate.message === 'string'
}
</script>

<template>
  <section class="panel chat-panel" aria-label="Game chat">
    <header>
      <span>Chat</span>
      <small :class="['status', status]">{{ status }}</small>
    </header>
    <div ref="messagesEl" class="messages">
      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="messages.length === 0" class="empty">Mở thêm tab thứ hai rồi gửi thử một tin nhắn.</p>
      <article v-for="item in messages" :key="item.id" :class="['message', { mine: item.mine }]">
        <strong>{{ item.mine ? 'Bạn' : item.characterId }}</strong>
        <span>{{ item.message }}</span>
      </article>
    </div>
    <form class="chat-form" @submit.prevent="sendMessage">
      <input v-model="draft" type="text" placeholder="Nhắn trong map...">
      <button type="submit" :disabled="!canSend">Gửi</button>
    </form>
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

header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid rgba(159, 212, 127, 0.18);
  font-weight: 700;
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
