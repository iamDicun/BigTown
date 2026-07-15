export type InterpolationState = {
  fromX: number
  fromY: number
  toX: number
  toY: number
  elapsedMs: number
  durationMs: number
}

export function interpolatePosition(state: InterpolationState) {
  const progress = Math.min(state.elapsedMs / state.durationMs, 1)

  return {
    x: state.fromX + (state.toX - state.fromX) * progress,
    y: state.fromY + (state.toY - state.fromY) * progress,
  }
}
