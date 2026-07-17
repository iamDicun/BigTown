import type { Direction } from '../network/gameEvents'

// Player.png là spritesheet 6 cột x 10 hàng, khung 32x32px (xem docs/Movement-Chat-Spawn-Plan.md
// mục D — soi pixel thật, không phải 12x20 @16px như nhận định ban đầu). Frame index = row*6 + col.
//
// Hàng 0: idle-down · Hàng 1: walk-down · Hàng 2: idle-up · Hàng 3: walk-up · Hàng 4: walk-left.
// Không có hàng walk-right riêng biệt rõ ràng — dùng chung hàng 4 + setFlipX(true) khi đi sang phải.
// Hàng 5-9 là attack/hurt, dành cho combat NPC ở phase sau, chưa dùng ở đây.

const COLS_PER_ROW = 6

const ROW = {
  idleDown: 0,
  walkDown: 1,
  idleUp: 2,
  walkUp: 3,
  walkSide: 4,
}

function rowFrames(row: number): number[] {
  return Array.from({ length: COLS_PER_ROW }, (_, i) => row * COLS_PER_ROW + i)
}

export const playerAnimKey = {
  idleDown: 'player-idle-down',
  walkDown: 'player-walk-down',
  idleUp: 'player-idle-up',
  walkUp: 'player-walk-up',
  idleSide: 'player-idle-side',
  walkSide: 'player-walk-side',
} as const

export function createPlayerAnimations(scene: Phaser.Scene): void {
  const frames = (row: number) => scene.anims.generateFrameNumbers('player', { frames: rowFrames(row) })

  scene.anims.create({ key: playerAnimKey.idleDown, frames: frames(ROW.idleDown), frameRate: 4, repeat: -1 })
  scene.anims.create({ key: playerAnimKey.walkDown, frames: frames(ROW.walkDown), frameRate: 8, repeat: -1 })
  scene.anims.create({ key: playerAnimKey.idleUp, frames: frames(ROW.idleUp), frameRate: 4, repeat: -1 })
  scene.anims.create({ key: playerAnimKey.walkUp, frames: frames(ROW.walkUp), frameRate: 8, repeat: -1 })
  scene.anims.create({ key: playerAnimKey.walkSide, frames: frames(ROW.walkSide), frameRate: 8, repeat: -1 })
  scene.anims.create({
    key: playerAnimKey.idleSide,
    frames: [{ key: 'player', frame: rowFrames(ROW.walkSide)[0] }],
  })
}

// Facing gộp left/right thành 'side' (khác biệt chỉ ở setFlipX) — dùng chung cho local player
// (GameScene.update) và remote player (applyRemoteMove) để không lặp logic chọn animation.
export type Facing = 'down' | 'up' | 'side'

export function facingForDirection(direction: Direction): Facing {
  if (direction === 'up') return 'up'
  if (direction === 'left' || direction === 'right') return 'side'
  return 'down'
}

export function walkAnimForFacing(facing: Facing): string {
  if (facing === 'up') return playerAnimKey.walkUp
  if (facing === 'side') return playerAnimKey.walkSide
  return playerAnimKey.walkDown
}

export function idleAnimForFacing(facing: Facing): string {
  if (facing === 'up') return playerAnimKey.idleUp
  if (facing === 'side') return playerAnimKey.idleSide
  return playerAnimKey.idleDown
}
