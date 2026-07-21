import type Phaser from 'phaser'

import type { Direction, PlayerMoveCommand } from '../network/gameEvents'
import {
  facingForDirection,
  idleAnimForFacing,
  walkAnimForFacing,
  type Facing,
} from '../phaser/playerAnimations'
import {
  createMovementThrottle,
  flushMovementThrottle,
  getDirectionFromInput,
  recordMovement,
  tickMovementThrottle,
  type MovementInput,
  type MovementThrottle,
} from './movementSystem'

const PLAYER_SPEED = 120
const MOVEMENT_THRESHOLD_MS = 100
const PLAYER_BODY_SIZE = { width: 16, height: 12 }
const PLAYER_BODY_OFFSET = { x: 8, y: 18 }

// Gói toàn bộ local player: sprite, input -> velocity/animation, throttle + gửi RPC player_move,
// và snap về vị trí authoritative (join snapshot / correction). Tách khỏi GameScene để scene
// không phình to khi thêm tính năng mới (combat, HP...) — xem docs/Phaser-Frontend-Guide.md mục 19.
export class LocalPlayerController {
  readonly sprite: Phaser.Physics.Arcade.Sprite

  private facing: Facing = 'down'
  private lastDirection: Direction = 'down'
  private wasMoving = false
  private movementThrottle: MovementThrottle = createMovementThrottle()
  private readonly sendMove: (command: PlayerMoveCommand) => void

  constructor(scene: Phaser.Scene, x: number, y: number, sendMove: (command: PlayerMoveCommand) => void) {
    this.sendMove = sendMove
    this.sprite = scene.physics.add.sprite(x, y, 'player', 0)

    const body = this.sprite.body as Phaser.Physics.Arcade.Body
    body.setSize(PLAYER_BODY_SIZE.width, PLAYER_BODY_SIZE.height)
    body.setOffset(PLAYER_BODY_OFFSET.x, PLAYER_BODY_OFFSET.y)

    this.sprite.anims.play(idleAnimForFacing(this.facing))
  }

  update(time: number, cursors: MovementKeys): void {
    const input: MovementInput = {
      up: cursors.up?.isDown ?? false,
      down: cursors.down?.isDown ?? false,
      left: cursors.left?.isDown ?? false,
      right: cursors.right?.isDown ?? false,
    }
    const direction = getDirectionFromInput(input)
    const moving = direction !== null

    this.sprite.setVelocity(0)
    if (direction) {
      this.applyMovement(direction)
    }
    this.sprite.anims.play(moving ? walkAnimForFacing(this.facing) : idleAnimForFacing(this.facing), true)

    recordMovement(this.movementThrottle, {
      x: Math.round(this.sprite.x),
      y: Math.round(this.sprite.y),
      direction: this.lastDirection,
      moving,
    })

    if (this.wasMoving && !moving) {
      // Người chơi vừa dừng — gửi ngay, không đợi network tick, để remote clients dừng animation
      // ngay lập tức (xem docs/Phaser-Frontend-Guide.md mục 10).
      flushMovementThrottle(this.movementThrottle, time, this.sendMove)
    } else {
      tickMovementThrottle(this.movementThrottle, time, MOVEMENT_THRESHOLD_MS, this.sendMove)
    }
    this.wasMoving = moving
  }

  // Correction đến qua personal channel khi 1 movement bị reject — xem
  // docs/Realtime-Room-State-Decisions.md mục 6. Snap thẳng, không tween, để tránh cảm giác
  // trôi lệch tiếp khỏi vị trí server chấp nhận.
  applyCorrection(x: number, y: number): void {
    this.sprite.setPosition(x, y)
    this.movementThrottle.latestMovement = null
  }

  // Dùng khi nhận room_snapshot lúc join — server có thể đã dịch spawn do anti-overlap
  // (xem docs/Movement-Chat-Spawn-Plan.md mục 0.1), nên vị trí thật khác bootstrap.spawn_x/y tĩnh.
  applyServerPosition(x: number, y: number, direction: Direction): void {
    this.sprite.setPosition(x, y)
    this.facing = facingForDirection(direction)
    this.lastDirection = direction
  }

  private applyMovement(direction: Direction): void {
    if (direction === 'left') {
      this.sprite.setVelocityX(-PLAYER_SPEED)
      this.sprite.setFlipX(true)
    } else if (direction === 'right') {
      this.sprite.setVelocityX(PLAYER_SPEED)
      this.sprite.setFlipX(false)
    } else if (direction === 'up') {
      this.sprite.setVelocityY(-PLAYER_SPEED)
    } else if (direction === 'down') {
      this.sprite.setVelocityY(PLAYER_SPEED)
    }

    this.facing = facingForDirection(direction)
    this.lastDirection = direction
  }
}

export type MovementKeys = {
  up?: Phaser.Input.Keyboard.Key
  down?: Phaser.Input.Keyboard.Key
  left?: Phaser.Input.Keyboard.Key
  right?: Phaser.Input.Keyboard.Key
}
