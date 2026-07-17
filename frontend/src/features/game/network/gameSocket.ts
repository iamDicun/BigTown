import { Centrifuge, type Subscription } from 'centrifuge'

import { getAccessToken } from '@/shared/api/tokenStorage'

import type {
  PlayerChatEvent,
  PlayerJoinedEvent,
  PlayerLeftEvent,
  PlayerMoveCommand,
  PlayerMoveEvent,
  PlayerPositionCorrectionEvent,
  RoomSnapshotEvent,
} from './gameEvents'

type GameSocketOptions = {
  // Channel room:<mapCode> — luôn lấy từ bootstrap.default_channel, không hardcode tên map
  // (xem docs/Architecture.md mục 9.1).
  channel: string
  onRoomSnapshot?: (event: RoomSnapshotEvent) => void
  onPlayerJoined?: (event: PlayerJoinedEvent) => void
  onPlayerLeft?: (event: PlayerLeftEvent) => void
  onPlayerMove?: (event: PlayerMoveEvent) => void
  onPlayerChat?: (event: PlayerChatEvent) => void
  // Personal channel (server-side subscription) — dùng cho player_position_correction, xem
  // docs/Realtime-Room-State-Decisions.md mục 6.
  onCorrection?: (event: PlayerPositionCorrectionEvent) => void
}

export function getDefaultRealtimeUrl() {
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL
  if (!apiBaseUrl) {
    throw new Error('Thiếu biến môi trường VITE_API_BASE_URL — kiểm tra file .env (xem .env.example).')
  }

  const baseUrl = new URL(apiBaseUrl)
  baseUrl.pathname = '/connection/websocket'
  baseUrl.search = ''
  baseUrl.protocol = baseUrl.protocol === 'https:' ? 'wss:' : 'ws:'

  return baseUrl.toString()
}

// Toàn bộ việc "raw JSON không rõ kiểu -> event đã gõ kiểu" nằm ở đây (network layer), không
// để scene/component tự đoán — xem docs/Phaser-Frontend-Guide.md mục 3 ("network: Centrifuge
// connection và event types").
export function createGameSocket(url: string, options: GameSocketOptions) {
  const token = getAccessToken()
  if (!token) {
    throw new Error('Missing access token for realtime connection')
  }

  const centrifuge = new Centrifuge(url, { token })
  const subscription: Subscription = centrifuge.newSubscription(options.channel)

  subscription.on('subscribed', (ctx) => {
    if (isRoomSnapshotEvent(ctx.data)) options.onRoomSnapshot?.(ctx.data)
  })

  subscription.on('publication', (ctx) => {
    const data: unknown = ctx.data
    if (isPlayerJoinedEvent(data)) options.onPlayerJoined?.(data)
    else if (isPlayerLeftEvent(data)) options.onPlayerLeft?.(data)
    else if (isPlayerMoveEvent(data)) options.onPlayerMove?.(data)
    else if (isPlayerChatEvent(data)) options.onPlayerChat?.(data)
  })

  // Personal channel là server-side subscription (ConnectReply.Subscriptions ở backend) —
  // publication của nó nổi lên top-level Centrifuge, không qua Subscription của room channel.
  centrifuge.on('publication', (ctx) => {
    if (isPositionCorrectionEvent(ctx.data)) options.onCorrection?.(ctx.data)
  })

  subscription.subscribe()
  centrifuge.connect()

  return {
    centrifuge,
    subscription,
    sendMove(command: PlayerMoveCommand) {
      return centrifuge.rpc('player_move', command)
    },
    close() {
      subscription.unsubscribe()
      centrifuge.disconnect()
    },
  }
}

export type GameSocket = ReturnType<typeof createGameSocket>

function hasType(event: unknown, type: string): boolean {
  return !!event && typeof event === 'object' && (event as { type?: unknown }).type === type
}

function isRoomSnapshotEvent(event: unknown): event is RoomSnapshotEvent {
  return hasType(event, 'room_snapshot')
}

function isPlayerJoinedEvent(event: unknown): event is PlayerJoinedEvent {
  return hasType(event, 'player_joined')
}

function isPlayerLeftEvent(event: unknown): event is PlayerLeftEvent {
  return hasType(event, 'player_left')
}

function isPlayerMoveEvent(event: unknown): event is PlayerMoveEvent {
  return hasType(event, 'player_move')
}

function isPlayerChatEvent(event: unknown): event is PlayerChatEvent {
  return hasType(event, 'player_chat')
}

function isPositionCorrectionEvent(event: unknown): event is PlayerPositionCorrectionEvent {
  return hasType(event, 'player_position_correction')
}
