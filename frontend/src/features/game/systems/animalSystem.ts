import Phaser from 'phaser'
import type { NPCSpawnDto } from '../services/realtime.service'
import { getGameTime } from './effects/common'

interface AnimalMeta {
  frame_width: number
  frame_height: number
  columns: number
  row_idle: number
  row_walk: number
  idle_frame_rate: number
  walk_frame_rate: number
  wander_radius: number
  wander_delay_s: number
  wander_duration_s: number
}

interface AnimalState {
  sprite: Phaser.GameObjects.Sprite
  spawnX: number
  spawnY: number
  meta: AnimalMeta
  animKey: string
  lastFacingLeft: boolean
  lastWalking: boolean
}

function detHash(x: number, y: number, n: number): number {
  let h = 0
  h = ((h << 5) - h + x) | 0
  h = ((h << 5) - h + y) | 0
  h = ((h << 5) - h + n) | 0
  return (h >>> 0) / 0xFFFFFFFF
}

function detRandom(seed: number): number {
  const x = Math.sin(seed * 127.1 + 311.7) * 43758.5453
  return x - Math.floor(x)
}

export class AnimalSystem {
  private scene: Phaser.Scene
  private animals: AnimalState[] = []

  constructor(scene: Phaser.Scene) {
    this.scene = scene
  }

  spawnFromBootstrap(spawns: NPCSpawnDto[]): void {
    if (!spawns || spawns.length === 0) return

    for (const s of spawns) {
      const meta = this.parseMeta(s.metadata_json)
      const textureKey = s.asset_key

      if (!this.scene.textures.exists(textureKey)) {
        this.scene.load.spritesheet(textureKey, `/assets/${textureKey}`, {
          frameWidth: meta.frame_width,
          frameHeight: meta.frame_height,
        })
        this.scene.load.once('complete', () => {
          if (this.scene.textures.exists(textureKey)) {
            this.createAnimal(s, meta)
          }
        })
        this.scene.load.start()
      } else {
        this.createAnimal(s, meta)
      }
    }
  }

  private parseMeta(raw: string): AnimalMeta {
    try {
      const parsed = JSON.parse(raw)
      return {
        frame_width: parsed.frame_width || 32,
        frame_height: parsed.frame_height || 32,
        columns: parsed.columns || 2,
        row_idle: parsed.row_idle || 0,
        row_walk: parsed.row_walk || 1,
        idle_frame_rate: parsed.idle_frame_rate || 4,
        walk_frame_rate: parsed.walk_frame_rate || 6,
        wander_radius: parsed.wander_radius || 48,
        wander_delay_s: (parsed.wander_delay_min || 2000) / 1000,
        wander_duration_s: (parsed.wander_delay_max || 5000) / 1000,
      }
    } catch {
      return {
        frame_width: 32, frame_height: 32, columns: 2,
        row_idle: 0, row_walk: 1,
        idle_frame_rate: 4, walk_frame_rate: 6,
        wander_radius: 48, wander_delay_s: 2, wander_duration_s: 5,
      }
    }
  }

  private createAnimal(s: NPCSpawnDto, meta: AnimalMeta): void {
    const textureKey = s.asset_key
    const animKey = textureKey.replace(/\//g, '_').replace(/\.png$/i, '')

    this.ensureAnimations(animKey, textureKey, meta)

    const sprite = this.scene.add.sprite(s.spawn_x, s.spawn_y, textureKey, 0)
    sprite.setOrigin(0.5, 1.0)
    sprite.play(`${animKey}_idle`)

    this.animals.push({
      sprite,
      spawnX: s.spawn_x,
      spawnY: s.spawn_y,
      meta,
      animKey,
      lastFacingLeft: false,
      lastWalking: false,
    })
  }

  private ensureAnimations(animKey: string, textureKey: string, meta: AnimalMeta): void {
    if (this.scene.anims.exists(`${animKey}_idle`)) return

    const idleFrames = Array.from({ length: meta.columns }, (_, i) => meta.row_idle * meta.columns + i)
    const walkFrames = Array.from({ length: meta.columns }, (_, i) => meta.row_walk * meta.columns + i)

    this.scene.anims.create({
      key: `${animKey}_idle`,
      frames: this.scene.anims.generateFrameNumbers(textureKey, { frames: idleFrames }),
      frameRate: meta.idle_frame_rate,
      repeat: -1,
    })
    this.scene.anims.create({
      key: `${animKey}_walk`,
      frames: this.scene.anims.generateFrameNumbers(textureKey, { frames: walkFrames }),
      frameRate: meta.walk_frame_rate,
      repeat: -1,
    })
  }

  update(_time: number, _delta: number): void {
    const worldTime = getGameTime()

    for (const a of this.animals) {
      const pos = this.detPosition(a, worldTime)
      a.sprite.setPosition(pos.x, pos.y)
      a.sprite.setDepth(2 + a.sprite.y / 10000.0)

      if (pos.isWalking !== a.lastWalking) {
        a.lastWalking = pos.isWalking
        if (pos.isWalking) {
          a.sprite.play(`${a.animKey}_walk`, true)
        } else {
          a.sprite.play(`${a.animKey}_idle`, true)
        }
      }

      if (pos.facingLeft !== a.lastFacingLeft) {
        a.lastFacingLeft = pos.facingLeft
        a.sprite.setFlipX(pos.facingLeft)
      }
    }
  }

  // Deterministic free-roaming: animal walks from its current position,
  // not from spawn. Multiple epochs accumulate to form a path.
  private detPosition(a: AnimalState, worldTime: number): { x: number; y: number; isWalking: boolean; facingLeft: boolean } {
    const totalCycle = a.meta.wander_delay_s + a.meta.wander_duration_s
    const gameSeconds = worldTime * 60
    const epoch = Math.floor(gameSeconds / totalCycle)
    const phase = gameSeconds - epoch * totalCycle

    // Accumulate walk offsets from all past epochs
    let accX = a.spawnX
    let accY = a.spawnY

    for (let e = 0; e < epoch; e++) {
      const hash = detHash(a.spawnX, a.spawnY, e)
      const angle = detRandom(hash + 0.1) * Math.PI * 2
      const idleDel = a.meta.wander_delay_s * 0.4 + detRandom(hash + 0.2) * a.meta.wander_delay_s * 0.6
      const walkDur = a.meta.wander_duration_s * 0.5 + detRandom(hash + 0.3) * a.meta.wander_duration_s * 0.5
      const dirX = Math.cos(angle)
      const dirY = Math.sin(angle)
      const speed = a.meta.wander_radius / a.meta.wander_duration_s
      const walkDist = speed * walkDur
      const movedX = accX + dirX * walkDist
      const movedY = accY + dirY * walkDist
      accX = Phaser.Math.Clamp(movedX, a.spawnX - a.meta.wander_radius * 2, a.spawnX + a.meta.wander_radius * 2)
      accY = Phaser.Math.Clamp(movedY, a.spawnY - a.meta.wander_radius * 2, a.spawnY + a.meta.wander_radius * 2)
    }

    // Current epoch
    const curHash = detHash(a.spawnX, a.spawnY, epoch)
    const curAngle = detRandom(curHash + 0.1) * Math.PI * 2
    const curIdleDel = a.meta.wander_delay_s * 0.4 + detRandom(curHash + 0.2) * a.meta.wander_delay_s * 0.6
    const curWalkDur = a.meta.wander_duration_s * 0.5 + detRandom(curHash + 0.3) * a.meta.wander_duration_s * 0.5
    const curDirX = Math.cos(curAngle)
    const curDirY = Math.sin(curAngle)
    const facingLeft = curDirX < 0
    const speed = a.meta.wander_radius / a.meta.wander_duration_s

    if (phase < curIdleDel) {
      return { x: accX, y: accY, isWalking: false, facingLeft: false }
    }

    const walkProgress = (phase - curIdleDel) / curWalkDur
    const clamped = Math.min(walkProgress, 1.0)
    const dist = speed * clamped * curWalkDur
    const curX = accX + curDirX * dist
    const curY = accY + curDirY * dist

    return {
      x: Phaser.Math.Clamp(curX, a.spawnX - a.meta.wander_radius * 2, a.spawnX + a.meta.wander_radius * 2),
      y: Phaser.Math.Clamp(curY, a.spawnY - a.meta.wander_radius * 2, a.spawnY + a.meta.wander_radius * 2),
      isWalking: true,
      facingLeft,
    }
  }

  destroy(): void {
    for (const a of this.animals) {
      a.sprite.destroy()
    }
    this.animals.length = 0
  }
}
