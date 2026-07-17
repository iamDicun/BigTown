import type Phaser from 'phaser'

import type { Direction } from '../network/gameEvents'
import { facingForDirection, idleAnimForFacing, walkAnimForFacing } from '../phaser/playerAnimations'

const TWEEN_DURATION_MS = 100

// Quản lý sprite của các player khác (không phải player local) theo characterId: dựng lúc
// join/room_snapshot, tween lúc nhận player_move, huỷ lúc player_left. Tách khỏi GameScene để
// dễ mở rộng sau (chat bubble, HP bar...) mà không đụng logic player khác — xem
// docs/Phaser-Frontend-Guide.md mục 19.
export class RemotePlayerManager {
  private readonly sprites = new Map<string, Phaser.GameObjects.Sprite>()
  private readonly scene: Phaser.Scene

  constructor(scene: Phaser.Scene) {
    this.scene = scene
  }

  upsert(characterId: string, x: number, y: number, direction: Direction, moving: boolean): void {
    const facing = facingForDirection(direction)
    let sprite = this.sprites.get(characterId)

    if (!sprite) {
      sprite = this.scene.add.sprite(x, y, 'player', 0)
      this.sprites.set(characterId, sprite)
    } else {
      this.scene.tweens.killTweensOf(sprite)
      this.scene.tweens.add({ targets: sprite, x, y, duration: TWEEN_DURATION_MS, ease: 'Linear' })
    }

    sprite.setFlipX(facing === 'side' && direction === 'left')
    sprite.anims.play(moving ? walkAnimForFacing(facing) : idleAnimForFacing(facing), true)
  }

  remove(characterId: string): void {
    const sprite = this.sprites.get(characterId)
    if (!sprite) return

    this.scene.tweens.killTweensOf(sprite)
    sprite.destroy()
    this.sprites.delete(characterId)
  }

  destroyAll(): void {
    for (const characterId of [...this.sprites.keys()]) {
      this.remove(characterId)
    }
  }
}
