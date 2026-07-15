import type { Direction } from '../network/gameEvents'

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
