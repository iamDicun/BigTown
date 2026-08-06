import Phaser from 'phaser'
import type { NPCSpawnDto } from '../services/realtime.service'

interface AnimalMeta {
  frame_width: number
  frame_height: number
  columns: number
  row_idle: number
  idle_frame_rate: number
}

export class AnimalSystem {
  private scene: Phaser.Scene
  private sprites: Phaser.GameObjects.Sprite[] = []

  constructor(scene: Phaser.Scene) {
    this.scene = scene
  }

  spawnFromBootstrap(spawns: NPCSpawnDto[]): void {
    if (!spawns || spawns.length === 0) return

    const pending: NPCSpawnDto[] = []
    for (const s of spawns) {
      const meta = this.parseMeta(s.metadata_json)
      if (this.scene.textures.exists(s.asset_key)) {
        this.createAnimal(s.spawn_x, s.spawn_y, s.asset_key, meta)
      } else {
        this.scene.load.spritesheet(s.asset_key, `/assets/${s.asset_key}`, {
          frameWidth: meta.frame_width,
          frameHeight: meta.frame_height,
        })
        pending.push(s)
      }
    }

    if (pending.length > 0) {
      this.scene.load.once('complete', () => {
        for (const s of pending) {
          if (this.scene.textures.exists(s.asset_key)) {
            this.createAnimal(s.spawn_x, s.spawn_y, s.asset_key, this.parseMeta(s.metadata_json))
          }
        }
      })
      this.scene.load.start()
    }
  }

  private parseMeta(raw: string): AnimalMeta {
    try {
      const p = JSON.parse(raw)
      return {
        frame_width: p.frame_width || 32,
        frame_height: p.frame_height || 32,
        columns: p.columns || 2,
        row_idle: p.row_idle || 0,
        idle_frame_rate: p.idle_frame_rate || 4,
      }
    } catch {
      return { frame_width: 32, frame_height: 32, columns: 2, row_idle: 0, idle_frame_rate: 4 }
    }
  }

  private createAnimal(x: number, y: number, textureKey: string, meta: AnimalMeta): void {
    const animKey = textureKey.replace(/\//g, '_').replace(/\.png$/i, '')

    if (!this.scene.anims.exists(`${animKey}_idle`)) {
      const idleFrames = Array.from({ length: meta.columns }, (_, i) => meta.row_idle * meta.columns + i)
      this.scene.anims.create({
        key: `${animKey}_idle`,
        frames: this.scene.anims.generateFrameNumbers(textureKey, { frames: idleFrames }),
        frameRate: meta.idle_frame_rate,
        repeat: -1,
      })
    }

    const sprite = this.scene.add.sprite(x, y, textureKey, 0)
    sprite.setOrigin(0.5, 1.0)
    sprite.setDepth(2 + y / 10000.0)
    sprite.play(`${animKey}_idle`)
    this.sprites.push(sprite)
  }

  update(_time: number, _delta: number): void {}

  destroy(): void {
    for (const s of this.sprites) s.destroy()
    this.sprites.length = 0
  }
}
