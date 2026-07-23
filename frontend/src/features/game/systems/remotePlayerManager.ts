import type Phaser from 'phaser'

import type { Direction } from '../network/gameEvents'
import { facingForDirection, idleAnimKey, walkAnimKey } from '../phaser/playerAnimations'
import type { CharacterOptionDto } from '../services/character.service'
import { createNameTag, updateNameTagPosition } from './nameTagSystem'

const TWEEN_DURATION_MS = 100
const REMOTE_BLOCK_RADIUS = 26

type RemoteEntry = {
  sprite: Phaser.GameObjects.Sprite
  zone: Phaser.GameObjects.Zone
  nameTag: Phaser.GameObjects.Text
  baseAssetKey: string
}

export class RemotePlayerManager {
  private readonly entries = new Map<string, RemoteEntry>()
  private readonly scene: Phaser.Scene
  private readonly optionsMap: Map<string, CharacterOptionDto>
  readonly group: Phaser.Physics.Arcade.StaticGroup

  constructor(scene: Phaser.Scene, options: CharacterOptionDto[]) {
    this.scene = scene
    this.group = scene.physics.add.staticGroup()
    this.optionsMap = new Map(options.map((o) => [o.base_asset_key, o]))
  }

  upsert(
    characterId: string,
    x: number,
    y: number,
    direction: Direction,
    moving: boolean,
    baseAssetKey?: string,
    name?: string,
  ): void {
    const facing = facingForDirection(direction)
    let entry = this.entries.get(characterId)

    const resolveKey = (key: string) => (key === 'cute_fantasy/player_base' ? 'player' : key)

    if (!entry) {
      const rawKey = baseAssetKey ?? 'player'
      const assetKey = resolveKey(rawKey)
      const option = this.optionsMap.get(assetKey)
      const scale = option ? 32 / option.spritesheet.frame_height : 1

      const sprite = this.scene.add.sprite(x, y, assetKey, 0)
      sprite.setScale(scale)

      const zone = this.scene.add.zone(x, y, REMOTE_BLOCK_RADIUS * 2, REMOTE_BLOCK_RADIUS * 2)
      this.scene.physics.add.existing(zone, true)
      ;(zone.body as Phaser.Physics.Arcade.StaticBody).setCircle(REMOTE_BLOCK_RADIUS)
      this.group.add(zone)

      const nameTag = createNameTag(this.scene, sprite, name ?? '')

      entry = { sprite, zone, nameTag, baseAssetKey: assetKey }
      this.entries.set(characterId, entry)
    } else {
      if (baseAssetKey) {
        entry.baseAssetKey = resolveKey(baseAssetKey)
      }
      this.scene.tweens.killTweensOf(entry.sprite)
      this.scene.tweens.add({ targets: entry.sprite, x, y, duration: TWEEN_DURATION_MS, ease: 'Linear' })
    }

    entry.zone.setPosition(x, y)
    ;(entry.zone.body as Phaser.Physics.Arcade.StaticBody).updateFromGameObject()

    entry.sprite.setFlipX(facing === 'side' && direction === 'left')
    entry.sprite.anims.play(
      moving ? walkAnimKey(entry.baseAssetKey, facing) : idleAnimKey(entry.baseAssetKey, facing),
      true,
    )
  }

  update(): void {
    for (const [, entry] of this.entries) {
      updateNameTagPosition(entry.nameTag, entry.sprite)
    }
  }

  remove(characterId: string): void {
    const entry = this.entries.get(characterId)
    if (!entry) return

    this.scene.tweens.killTweensOf(entry.sprite)
    entry.sprite.destroy()
    entry.nameTag.destroy()
    this.group.remove(entry.zone)
    entry.zone.destroy()
    this.entries.delete(characterId)
  }

  destroyAll(): void {
    for (const characterId of [...this.entries.keys()]) {
      this.remove(characterId)
    }
  }
}
