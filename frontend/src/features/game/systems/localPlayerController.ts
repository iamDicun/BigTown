import type Phaser from 'phaser'

import { playRandomSfx } from '@/shared/audio/audio.service'

import type { Direction, PlayerMoveCommand } from '../network/gameEvents'
import {
  facingForDirection,
  idleAnimKey,
  walkAnimKey,
  type Facing,
} from '../phaser/playerAnimations'
import type { SpritesheetConfigDto } from '../services/character.service'
import {
  createMovementThrottle,
  flushMovementThrottle,
  getDirectionFromInput,
  recordMovement,
  tickMovementThrottle,
  type MovementInput,
  type MovementThrottle,
} from './movementSystem'
import { createNameTag, updateNameTagPosition } from './nameTagSystem'

const PLAYER_SPEED = 120
const MOVEMENT_THRESHOLD_MS = 100
const PLAYER_BODY_SIZE = { width: 16, height: 12 }
const FOOTSTEP_INTERVAL_MS = 460
const FOOTSTEP_VOLUME = 0.38
const FOOTSTEP_SOUNDS = [
  '/assets/sounds/f1.mp3',
  '/assets/sounds/f2.mp3',
  '/assets/sounds/f3.mp3',
  '/assets/sounds/f4.mp3',
  '/assets/sounds/f5.mp3',
  '/assets/sounds/f6.mp3',
  '/assets/sounds/f7.mp3',
]

// Gói toàn bộ local player: sprite, input -> velocity/animation, throttle + gửi RPC player_move,
// và snap về vị trí authoritative (join snapshot / correction). Tách khỏi GameScene để scene
// không phình to khi thêm tính năng mới (combat, HP...) — xem docs/Phaser-Frontend-Guide.md mục 19.
export class LocalPlayerController {
  readonly sprite: Phaser.Physics.Arcade.Sprite

  private readonly textureKey: string
  private facing: Facing = 'down'
  private lastDirection: Direction = 'down'
  private wasMoving = false
  private movementThrottle: MovementThrottle = createMovementThrottle()
  private readonly sendMove: (command: PlayerMoveCommand) => void
  private readonly scene: Phaser.Scene
  private nameTag: Phaser.GameObjects.Text | null = null
  private lastFootstepAt = 0

  constructor(
    scene: Phaser.Scene,
    textureKey: string,
    x: number,
    y: number,
    sendMove: (command: PlayerMoveCommand) => void,
    config: SpritesheetConfigDto,
  ) {
    this.sendMove = sendMove
    this.scene = scene
    this.textureKey = textureKey

    const scale = 32 / config.frame_height

    this.sprite = scene.physics.add.sprite(x, y, textureKey, 0)
    this.sprite.setScale(scale)

    const body = this.sprite.body as Phaser.Physics.Arcade.Body
    // setSize luôn dùng local coordinate của sprite (trước scale). Với scale < 1, body bị thu nhỏ
    // trong world space — phải bù ngược để body luôn ~16×12 trong world space bất kể scale.
    body.setSize(PLAYER_BODY_SIZE.width / scale, PLAYER_BODY_SIZE.height / scale)
    body.setOffset(
      (config.frame_width - PLAYER_BODY_SIZE.width / scale) / 2,
      config.frame_height - PLAYER_BODY_SIZE.height / scale - 2 / scale,
    )

    this.sprite.anims.play(idleAnimKey(textureKey, this.facing))
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
    this.sprite.anims.play(moving ? walkAnimKey(this.textureKey, this.facing) : idleAnimKey(this.textureKey, this.facing), true)

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
    this.playFootstepIfNeeded(time, moving)

    if (this.nameTag) {
      updateNameTagPosition(this.nameTag, this.sprite)
    }
  }

  // Tên character lấy từ room_snapshot lúc join (server trả về, không đọc từ token — xem
  // docs/Realtime-Performance-Fixes.md mục 6/nametag), tạo lười lúc có tên thay vì lúc constructor
  // vì tên chưa biết trước khi room_snapshot tới.
  setName(name: string): void {
    if (!this.nameTag) {
      this.nameTag = createNameTag(this.scene, this.sprite, name)
      return
    }
    this.nameTag.setText(name)
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

  private playFootstepIfNeeded(time: number, moving: boolean): void {
    if (!moving) return
    if (time - this.lastFootstepAt < FOOTSTEP_INTERVAL_MS) return

    this.lastFootstepAt = time
    playRandomSfx(FOOTSTEP_SOUNDS, FOOTSTEP_VOLUME)
  }
}

export type MovementKeys = {
  up?: Phaser.Input.Keyboard.Key
  down?: Phaser.Input.Keyboard.Key
  left?: Phaser.Input.Keyboard.Key
  right?: Phaser.Input.Keyboard.Key
}
