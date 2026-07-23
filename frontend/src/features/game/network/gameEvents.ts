export type Direction = 'up' | 'down' | 'left' | 'right'

// Payload client gửi qua Centrifuge RPC method "player_move". Không có characterId — server tự
// resolve từ token đã xác thực (xem docs/Realtime-Room-State-Decisions.md mục 6, Hướng B).
export type PlayerMoveCommand = {
  x: number
  y: number
  direction: Direction
  moving: boolean
}

// Event server broadcast vào room channel sau khi accept 1 movement.
export type PlayerMoveEvent = {
  type: 'player_move'
  characterId: string
  x: number
  y: number
  direction: Direction
  moving: boolean
}

export type RoomPlayerDto = {
  characterId: string
  name: string
  baseAssetKey: string
  x: number
  y: number
  direction: Direction
  moving: boolean
}

// Gửi riêng cho client vừa join qua Data của subscribe reply — xem
// backend/internal/module/realtime/transport/events.go roomSnapshotEvent.
export type RoomSnapshotEvent = {
  type: 'room_snapshot'
  roomId: string
  players: RoomPlayerDto[]
}

export type PlayerJoinedEvent = {
  type: 'player_joined'
  roomId: string
  player: RoomPlayerDto
}

export type PlayerLeftEvent = {
  type: 'player_left'
  roomId: string
  characterId: string
}

// Gửi riêng cho client bị reject qua personal channel (server-side subscription), không qua
// response của RPC player_move — xem docs/Realtime-Room-State-Decisions.md mục 6.
export type PlayerPositionCorrectionEvent = {
  type: 'player_position_correction'
  characterId: string
  reason: string
  x: number
  y: number
}

// player_chat giờ chỉ được server publish (xem docs/Realtime-Room-State-Decisions.md mục 9).
// Client không còn gửi event này qua socket — gửi chat qua chat.service.ts (HTTP POST).
export type PlayerChatEvent = {
  type: 'player_chat'
  roomId: string
  characterId: string
  displayName: string
  message: string
  sentAt: string
}

// enemy_hit chưa implement ở batch này (xem docs/Movement-Chat-Spawn-Plan.md mục I) — giữ type
// để khớp wire format đã chốt cho phase combat sau.
export type EnemyHitEvent = {
  type: 'enemy_hit'
  npcRuntimeId: string
}

export type RoomEvent = PlayerJoinedEvent | PlayerLeftEvent | PlayerMoveEvent | PlayerChatEvent
