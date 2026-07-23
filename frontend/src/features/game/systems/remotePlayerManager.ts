import type Phaser from 'phaser'

import type { Direction } from '../network/gameEvents'
import { facingForDirection, idleAnimKey, walkAnimKey } from '../phaser/playerAnimations'
import type { SpritesheetConfigDto } from '../services/character.service'
import { createNameTag, updateNameTagPosition } from './nameTagSystem'

const TWEEN_DURATION_MS = 100

// Bán kính vùng chặn quanh remote player PHẢI khớp (hoặc lớn hơn 1 chút) minDistancePx mà server
// dùng để validate va chạm (xem backend room_usecase.go: minDistancePx = 24.0, so khoảng cách giữa
// 2 tâm sprite x/y — đúng giá trị FE gửi lên mỗi tick). Trước đây hàng rào này là hình chữ nhật nhỏ
// (16x12, dùng lại kích thước body va chạm với TƯỜNG) — nhỏ hơn hẳn vùng 24px server thực sự từ
// chối, nên local player vẫn lách được vào vùng server coi là "occupied" trước khi bị chặn hình ảnh,
// rồi bị RPC reject/correction giật vị trí về sau — đúng hiện tượng "đi qua rồi giật lại". Đổi sang
// hình tròn tâm đúng bằng x/y remote (không lệch tâm) bán kính = minDistancePx + margin nhỏ (bù
// chênh lệch giữa tâm sprite local và body va chạm-với-tường của nó, vốn có offset lệch tâm riêng)
// để local player luôn bị chặn hình ảnh TRƯỚC khi tiến đủ gần để server phải từ chối.
const REMOTE_BLOCK_RADIUS = 26

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
  private readonly nameTags = new Map<string, Phaser.GameObjects.Text>()
  private readonly scene: Phaser.Scene
  private readonly textureKey: string
  private readonly spriteScale: number
  readonly group: Phaser.Physics.Arcade.StaticGroup

  constructor(scene: Phaser.Scene, textureKey: string, config: SpritesheetConfigDto) {
    this.scene = scene
    this.textureKey = textureKey
    this.spriteScale = 32 / config.frame_height
    this.group = scene.physics.add.staticGroup()
  }

  // name chỉ cần truyền lúc tạo mới (room_snapshot/player_joined luôn có name) — player_move không
  // mang name (không đổi sau khi join) nên gọi upsert() không cần truyền lại, tham số optional.
  upsert(characterId: string, x: number, y: number, direction: Direction, moving: boolean, name?: string): void {
    const facing = facingForDirection(direction)
    let sprite = this.sprites.get(characterId)

    if (!sprite) {
      sprite = this.scene.add.sprite(x, y, this.textureKey, 0)
      sprite.setScale(this.spriteScale)
      this.sprites.set(characterId, sprite)

      const zone = this.scene.add.zone(x, y, REMOTE_BLOCK_RADIUS * 2, REMOTE_BLOCK_RADIUS * 2)
      this.scene.physics.add.existing(zone, true)
      ;(zone.body as Phaser.Physics.Arcade.StaticBody).setCircle(REMOTE_BLOCK_RADIUS)
      this.group.add(zone)
      this.zones.set(characterId, zone)

      this.nameTags.set(characterId, createNameTag(this.scene, sprite, name ?? ''))
    } else {
      this.scene.tweens.killTweensOf(sprite)
      this.scene.tweens.add({ targets: sprite, x, y, duration: TWEEN_DURATION_MS, ease: 'Linear' })
    }

    // Zone va chạm snap thẳng về vị trí server xác nhận ngay (không tween theo) — chỉ cần đúng vị
    // trí cuối để chặn local player, sai lệch trong khoảng tween 100ms là không đáng kể về gameplay.
    const zone = this.zones.get(characterId)!
    zone.setPosition(x, y)
    ;(zone.body as Phaser.Physics.Arcade.StaticBody).updateFromGameObject()

    sprite.setFlipX(facing === 'side' && direction === 'left')
    sprite.anims.play(moving ? walkAnimKey(this.textureKey, facing) : idleAnimKey(this.textureKey, facing), true)
  }

  // Gọi mỗi frame từ GameScene.update() — name tag không tween theo sprite, mà đọc lại vị trí
  // render thật (đã nội suy bởi tween) mỗi frame để luôn bám đúng đầu sprite (xem nameTagSystem.ts).
  update(): void {
    for (const [characterId, sprite] of this.sprites) {
      const nameTag = this.nameTags.get(characterId)
      if (nameTag) {
        updateNameTagPosition(nameTag, sprite)
      }
    }
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

    const nameTag = this.nameTags.get(characterId)
    if (nameTag) {
      nameTag.destroy()
      this.nameTags.delete(characterId)
    }
  }

  destroyAll(): void {
    const characterIds = new Set([...this.sprites.keys(), ...this.zones.keys(), ...this.nameTags.keys()])
    for (const characterId of characterIds) {
      this.remove(characterId)
    }
  }
}
