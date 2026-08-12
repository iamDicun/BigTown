<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

import { useGameStore } from '../stores/game.store'
import { createGameSocket, getDefaultRealtimeUrl } from '../network/gameSocket'
import type { PlayerChatEvent } from '../network/gameEvents'
import * as chatService from '../services/chat.service'
import type { ChatMessageDto } from '../services/chat.service'
import * as realtimeService from '../services/realtime.service'
import PixelIcon from '@/shared/components/PixelIcon.vue'

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
const inputEl = ref<HTMLInputElement | null>(null)
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
let mapChangedHandler: ((e: Event) => void) | null = null
let focusChatInputHandler: ((e: Event) => void) | null = null
let blurChatInputHandler: ((e: Event) => void) | null = null

const canSend = computed(() => !sending.value && draft.value.trim().length > 0)

function toggleCollapsed() {
  collapsed.value = !collapsed.value
  if (!collapsed.value) scrollToBottom()
}

function onInputFocus() {
  window.dispatchEvent(new CustomEvent('game:chatFocus', { detail: { focused: true } }))
}

function onInputBlur() {
  window.dispatchEvent(new CustomEvent('game:chatFocus', { detail: { focused: false } }))
}

function onInputKeydown(e: KeyboardEvent) {
  // Bắt tất cả sự kiện gõ phím trong khung chat không cho lan ra window (disable hotkeys)
  e.stopPropagation()

  if (e.key === 'Enter') {
    e.preventDefault()
    if (canSend.value) {
      sendMessage()
    } else {
      inputEl.value?.blur()
    }
  }
  if (e.key === 'Escape') {
    e.preventDefault()
    inputEl.value?.blur()
    collapsed.value = true
  }
}

function disconnectSocket() {
  gameSocket?.close()
  gameSocket = null
}

async function connectToRoom(mapCode?: string) {
  disconnectSocket()
  status.value = 'connecting'
  error.value = ''
  messages.value = []

  let bootstrap: Awaited<ReturnType<typeof realtimeService.getBootstrap>>
  try {
    bootstrap = await realtimeService.getBootstrap(mapCode)
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
}

onMounted(async () => {
  try {
    if (!gameStore.characterId) {
      await gameStore.loadMyCharacter()
    }
  } catch {
    // Không chặn chat nếu chưa load được character — "mine" chỉ tạm sai, vẫn nhận/gửi được tin.
  }

  await connectToRoom()

  mapChangedHandler = (e: Event) => {
    const detail = (e as CustomEvent).detail as { mapCode: string }
    if (detail?.mapCode) {
      connectToRoom(detail.mapCode)
    }
  }
  window.addEventListener('game:mapChanged', mapChangedHandler)

  focusChatInputHandler = () => {
    if (collapsed.value) {
      collapsed.value = false
      nextTick(() => inputEl.value?.focus())
    } else {
      inputEl.value?.focus()
    }
  }
  window.addEventListener('game:focusChatInput', focusChatInputHandler)

  blurChatInputHandler = () => {
    inputEl.value?.blur()
  }
  window.addEventListener('game:blurChatInput', blurChatInputHandler)
})

onBeforeUnmount(() => {
  disconnectSocket()
  if (mapChangedHandler) {
    window.removeEventListener('game:mapChanged', mapChangedHandler)
    mapChangedHandler = null
  }
  if (focusChatInputHandler) {
    window.removeEventListener('game:focusChatInput', focusChatInputHandler)
    focusChatInputHandler = null
  }
  if (blurChatInputHandler) {
    window.removeEventListener('game:blurChatInput', blurChatInputHandler)
    blurChatInputHandler = null
  }
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

function formatTime(isoString?: string): string {
  if (!isoString) return ''
  const date = new Date(isoString)
  if (isNaN(date.getTime())) return ''
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
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
  <section class="chat-panel ui-panel" :class="{ collapsed }" aria-label="Game chat">
    <header>
      <span class="chat-panel__title">Chat</span>
      <div class="header-actions">
        <span class="pixel-badge" :class="statusBadgeClass">{{ statusLabel[status] }}</span>
        <button
          type="button"
          class="ui-btn ui-btn--ghost ui-btn--sm"
          :aria-expanded="!collapsed"
          :aria-label="collapsed ? 'Mở rộng khung chat' : 'Thu nhỏ khung chat'"
          @click="toggleCollapsed"
        >
          <PixelIcon :name="collapsed ? 'chevron-right' : 'chevron-down'" :size="16" />
        </button>
      </div>
    </header>
    <template v-if="!collapsed">
      <div ref="messagesEl" class="messages ui-scroll">
        <p v-if="error" class="pixel-alert pixel-alert--error">{{ error }}</p>
        <p v-if="messages.length === 0" class="empty">Mở thêm tab thứ hai rồi gửi thử một tin nhắn.</p>
        <article v-for="item in messages" :key="item.id" :class="['message', { mine: item.mine }]">
          <div class="message-header">
            <strong>{{ item.mine ? 'Bạn' : item.displayName }}</strong>
            <time v-if="item.sentAt" class="message-time">{{ formatTime(item.sentAt) }}</time>
          </div>
          <span>{{ item.message }}</span>
        </article>
      </div>
      <form class="chat-form" @submit.prevent="sendMessage">
        <input ref="inputEl" v-model="draft" class="ui-input chat-input" type="text" placeholder="Nhắn trong map..." @focus="onInputFocus" @blur="onInputBlur" @keydown="onInputKeydown">
        <button type="submit" class="ui-btn ui-btn--sm" :disabled="!canSend">Gửi</button>
      </form>
    </template>
  </section>
</template>

<style scoped>
.chat-panel {
  min-height: 0; flex: 1 1 auto; display: grid;
  grid-template-rows: auto 1fr auto;
  padding: 0;
}
.chat-panel.collapsed {
  flex: 0 0 auto;
  grid-template-rows: auto;
}
header {
  display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-3);
  background: linear-gradient(180deg, var(--pixel-wood) 0%, var(--pixel-wood-dark) 100%);
  border-bottom: var(--bw) solid var(--pixel-outline);
}
.chat-panel__title {
  font-family: var(--pixel-font); font-size: var(--fs-head); letter-spacing: var(--ls-pixel);
  color: var(--pixel-text-inverse); text-shadow: 1px 1px 0 var(--pixel-outline);
}
.header-actions { display: flex; align-items: center; gap: var(--sp-3); }
.messages {
  min-height: 260px; max-height: 320px; overflow-y: auto; padding: var(--sp-3);
  font-family: var(--pixel-font); font-size: var(--fs-body);
}
.empty { margin: 0; color: var(--pixel-text-muted); opacity: 0.75; }
.message {
  display: grid; gap: 2px; margin-bottom: var(--sp-2); padding: var(--sp-1) var(--sp-2);
  background: rgba(255,255,255,0.5); border: 2px solid var(--pixel-parchment-dark);
}
.message.mine {
  background: rgba(90,156,74,0.16); border-color: var(--pixel-green);
}
.message-header {
  display: flex; justify-content: space-between; align-items: center; gap: var(--sp-2);
}
.message-time {
  font-size: 11px; color: var(--pixel-text-muted); opacity: 0.8; font-family: var(--pixel-font);
}
.message strong { color: var(--pixel-accent-dark); font-size: var(--fs-label); letter-spacing: var(--ls-pixel); }
.message span { color: var(--pixel-ink); word-break: break-word; line-height: 1.2; }
.chat-form {
  display: grid; grid-template-columns: 1fr auto; gap: var(--sp-2);
  padding: var(--sp-3); background: var(--pixel-wood-dark);
}
.chat-input { min-width: 0; }
</style>
