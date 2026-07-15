export type Direction = 'up' | 'down' | 'left' | 'right'

export type PlayerMoveEvent = {
  type: 'player_move'
  characterId: string
  x: number
  y: number
  direction: Direction
  moving: boolean
}

export type PlayerChatEvent = {
  type: 'player_chat'
  characterId: string
  message: string
  sentAt: string
}

export type EnemyHitEvent = {
  type: 'enemy_hit'
  npcRuntimeId: string
}

export type GameClientEvent = PlayerMoveEvent | PlayerChatEvent | EnemyHitEvent
