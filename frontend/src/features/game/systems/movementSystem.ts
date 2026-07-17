import type { Direction, PlayerMoveCommand } from '../network/gameEvents'

export type MovementInput = {
  up: boolean
  down: boolean
  left: boolean
  right: boolean
}

export function getDirectionFromInput(input: MovementInput): Direction | null {
  if (input.up) return 'up'
  if (input.down) return 'down'
  if (input.left) return 'left'
  if (input.right) return 'right'
  return null
}

// Throttled latest-event publishing — KHÔNG phải debounce (xem
// docs/Realtime-Room-State-Decisions.md mục 10). latestMovement luôn bị ghi đè bằng event mới
// nhất; network tick chỉ gửi khi đã qua movementThresholdMs kể từ lastSentAt.
export type MovementThrottle = {
  latestMovement: PlayerMoveCommand | null
  lastSentAt: number
}

export function createMovementThrottle(): MovementThrottle {
  return { latestMovement: null, lastSentAt: 0 }
}

export function recordMovement(throttle: MovementThrottle, movement: PlayerMoveCommand): void {
  throttle.latestMovement = movement
}

export function tickMovementThrottle(
  throttle: MovementThrottle,
  now: number,
  thresholdMs: number,
  send: (movement: PlayerMoveCommand) => void,
): void {
  if (!throttle.latestMovement) return
  if (now - throttle.lastSentAt < thresholdMs) return

  const movement = throttle.latestMovement
  throttle.latestMovement = null
  throttle.lastSentAt = now
  send(movement)
}

// Gửi ngay lập tức, bỏ qua threshold — dùng khi người chơi vừa dừng để remote clients dừng
// animation ngay, không đợi tick tiếp theo (xem docs/Phaser-Frontend-Guide.md mục 10).
export function flushMovementThrottle(throttle: MovementThrottle, now: number, send: (movement: PlayerMoveCommand) => void): void {
  if (!throttle.latestMovement) return

  const movement = throttle.latestMovement
  throttle.latestMovement = null
  throttle.lastSentAt = now
  send(movement)
}
