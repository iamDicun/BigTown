import type Phaser from 'phaser'

import type { Direction } from '../network/gameEvents'
import { facingForDirection, idleAnimForFacing, walkAnimForFacing } from '../phaser/playerAnimations'

const TWEEN_DURATION_MS = 100

// Cùng kích thước/offset body với local player (xem localPlayerController.ts) — mục đích là chặn
// hình ảnh 2 player đè lên nhau NGAY trên client, không phải đợi round-trip player_move ->
// BE validate minDistance -> player_position_correction mới bị kéo lại (khoảng trễ throttle 100ms +
// RTT khiến sprite có thể đi xuyên qua nhau trước khi bị chặn). Quyết định vị trí "hợp lệ" cuối
// cùng vẫn do BE (RoomUsecase.MovePlayer, minDistance 24px) — collider này chỉ là hàng rào thị giác.
const BODY_SIZE = { width: 16, height: 12 }
const BODY_OFFSET = { x: 8, y: 18 }

// Quản lý sprite của các player khác (không phải player local) theo characterId: dựng lúc
// join/room_snapshot, tween lúc nhận player_move, huỷ lúc player_left. Tách khỏi GameScene để
// dễ mở rộng sau (chat bubble, HP bar...) mà không đụng logic player khác — xem
// docs/Phaser-Frontend-Guide.md mục 19.
export class RemotePlayerManager {
  private readonly sprites = new Map<string, Phaser.Physics.Arcade.Sprite>()
  private readonly scene: Phaser.Scene
  // Group vật lý chứa toàn bộ remote sprite hiện có — GameScene chỉ cần đăng ký 1 collider với
  // group này lúc create(), member thêm/xoá sau tự động được collider áp dụng.
  readonly group: Phaser.Physics.Arcade.Group

  constructor(scene: Phaser.Scene) {
    this.scene = scene
    this.group = scene.physics.add.group()
  }

  upsert(characterId: string, x: number, y: number, direction: Direction, moving: boolean): void {
    const facing = facingForDirection(direction)
    let sprite = this.sprites.get(characterId)

    if (!sprite) {
      sprite = this.scene.physics.add.sprite(x, y, 'player', 0)
      const body = sprite.body as Phaser.Physics.Arcade.Body
      body.setSize(BODY_SIZE.width, BODY_SIZE.height)
      body.setOffset(BODY_OFFSET.x, BODY_OFFSET.y)
      // Immovable: local player va vào remote sprite chỉ có local player bị đẩy ra — vị trí remote
      // vẫn hoàn toàn do tween/server quyết định, không bị vật lý local xô lệch.
      body.setImmovable(true)
      this.group.add(sprite)
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
    this.group.remove(sprite)
    sprite.destroy()
    this.sprites.delete(characterId)
  }

  destroyAll(): void {
    for (const characterId of [...this.sprites.keys()]) {
      this.remove(characterId)
    }
  }
}
