import type Phaser from 'phaser'

import type { Direction } from '../network/gameEvents'
import { facingForDirection, idleAnimForFacing, walkAnimForFacing } from '../phaser/playerAnimations'

const TWEEN_DURATION_MS = 100

// Kích thước hàng rào va chạm khớp với body local player (xem localPlayerController.ts), quy đổi
// từ offset-góc-trên-trái (setOffset 8,18 trên frame 32x32 origin 0.5/0.5) sang offset-tính-từ-tâm
// sprite: center = (x - 16 + 8 + 16/2, y - 16 + 18 + 12/2) = (x, y + 8).
const BODY_SIZE = { width: 16, height: 12 }
const BODY_CENTER_OFFSET = { x: 0, y: 8 }

// Quản lý sprite của các player khác (không phải player local) theo characterId: dựng lúc
// join/room_snapshot, tween lúc nhận player_move, huỷ lúc player_left. Tách khỏi GameScene để
// dễ mở rộng sau (chat bubble, HP bar...) mà không đụng logic player khác — xem
// docs/Phaser-Frontend-Guide.md mục 19.
//
// Sprite hiển thị KHÔNG gắn physics body trực tiếp — vẫn là scene.add.sprite thường + tween như
// trước, để không đụng tới cách render/nội suy đã hoạt động đúng. Hàng rào va chạm với local player
// là 1 Zone + STATIC body RIÊNG, đặt lại đúng vị trí server xác nhận mỗi lần upsert (không tween).
// Lý do tách riêng: gắn dynamic body thẳng lên sprite đang bị tween khiến Arcade tự đồng bộ lại vị
// trí body từ transform mỗi step rồi chạy step vật lý trên top của vị trí đó — dynamic body dù
// immovable vẫn tham gia integration mỗi frame, xung đột với tween liên tục ghi đè x/y, biểu hiện
// ra là remote player bị giật/rung qua lại (chạy xa rồi snap về) khi local player chạm vào. Static
// body không tham gia step integration, chỉ cần setPosition + updateFromGameObject() là xong, tránh
// hoàn toàn xung đột đó (cùng kỹ thuật buildCollisionGroup() trong mapSystem.ts đã dùng cho tường).
export class RemotePlayerManager {
  private readonly sprites = new Map<string, Phaser.GameObjects.Sprite>()
  private readonly zones = new Map<string, Phaser.GameObjects.Zone>()
  private readonly scene: Phaser.Scene
  // Group vật lý (static) chứa toàn bộ zone va chạm hiện có — GameScene chỉ cần đăng ký 1 collider
  // với group này lúc create(), member thêm/xoá sau tự động được collider áp dụng.
  readonly group: Phaser.Physics.Arcade.StaticGroup

  constructor(scene: Phaser.Scene) {
    this.scene = scene
    this.group = scene.physics.add.staticGroup()
  }

  upsert(characterId: string, x: number, y: number, direction: Direction, moving: boolean): void {
    const facing = facingForDirection(direction)
    let sprite = this.sprites.get(characterId)

    if (!sprite) {
      sprite = this.scene.add.sprite(x, y, 'player', 0)
      this.sprites.set(characterId, sprite)

      const zone = this.scene.add.zone(x + BODY_CENTER_OFFSET.x, y + BODY_CENTER_OFFSET.y, BODY_SIZE.width, BODY_SIZE.height)
      this.scene.physics.add.existing(zone, true)
      this.group.add(zone)
      this.zones.set(characterId, zone)
    } else {
      this.scene.tweens.killTweensOf(sprite)
      this.scene.tweens.add({ targets: sprite, x, y, duration: TWEEN_DURATION_MS, ease: 'Linear' })
    }

    // Zone va chạm snap thẳng về vị trí server xác nhận ngay (không tween theo) — chỉ cần đúng vị
    // trí cuối để chặn local player, sai lệch trong khoảng tween 100ms là không đáng kể về gameplay.
    const zone = this.zones.get(characterId)!
    zone.setPosition(x + BODY_CENTER_OFFSET.x, y + BODY_CENTER_OFFSET.y)
    ;(zone.body as Phaser.Physics.Arcade.StaticBody).updateFromGameObject()

    sprite.setFlipX(facing === 'side' && direction === 'left')
    sprite.anims.play(moving ? walkAnimForFacing(facing) : idleAnimForFacing(facing), true)
  }

  remove(characterId: string): void {
    const sprite = this.sprites.get(characterId)
    if (sprite) {
      this.scene.tweens.killTweensOf(sprite)
      sprite.destroy()
      this.sprites.delete(characterId)
    }

    const zone = this.zones.get(characterId)
    if (zone) {
      this.group.remove(zone)
      zone.destroy()
      this.zones.delete(characterId)
    }
  }

  destroyAll(): void {
    const characterIds = new Set([...this.sprites.keys(), ...this.zones.keys()])
    for (const characterId of characterIds) {
      this.remove(characterId)
    }
  }
}
