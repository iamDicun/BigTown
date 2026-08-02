import Phaser from 'phaser'
import * as editorService from '../services/editor.service'
import type { BootstrapDto } from '../services/realtime.service'
import type { GameScene } from '../phaser/GameScene'

type CoinConfig = {
  type: string
  key: string
  max: number
}

const COIN_CONFIGS: CoinConfig[] = [
  { type: 'gri', key: 'coin_gri', max: 20 },
  { type: 'ama', key: 'coin_ama', max: 20 },
  { type: 'azu', key: 'coin_azu', max: 20 },
  { type: 'roj', key: 'coin_roj', max: 20 },
  { type: 'gold', key: 'coin_gems', max: 20 }
]

export class CoinPickupSystem {
  private scene: GameScene
  private mapCode: string
  private playerSprite: Phaser.GameObjects.Sprite
  private coinsGroup!: Phaser.Physics.Arcade.StaticGroup
  private bootstrap: BootstrapDto
  private activeCounts: Record<string, number> = {
    gri: 0,
    ama: 0,
    azu: 0,
    roj: 0,
    gold: 0
  }

  constructor(scene: GameScene, bootstrap: BootstrapDto, playerSprite: Phaser.GameObjects.Sprite) {
    this.scene = scene
    this.mapCode = bootstrap.map_code
    this.bootstrap = bootstrap
    this.playerSprite = playerSprite
    this.initialize()
  }

  private initialize() {
    this.coinsGroup = this.scene.physics.add.staticGroup()

    // Setup overlap between player and coins
    this.scene.physics.add.overlap(this.playerSprite, this.coinsGroup, (_player, coinObj) => {
      this.handlePickup(coinObj as Phaser.GameObjects.Sprite)
    })

    // Setup spin animations for all 5 coin types if they don't exist
    for (const config of COIN_CONFIGS) {
      const animKey = `coin_spin_${config.type}`
      if (!this.scene.anims.exists(animKey)) {
        this.scene.anims.create({
          key: animKey,
          frames: this.scene.anims.generateFrameNumbers(config.key, { start: 0, end: 3 }),
          frameRate: 8,
          repeat: -1
        })
      }
    }

    // Spawn initial local coins for each type
    this.spawnInitialCoins()
  }

  private spawnInitialCoins() {
    for (const config of COIN_CONFIGS) {
      for (let i = 0; i < config.max; i++) {
        this.spawnRandomCoin(config.type)
      }
    }
  }

  private spawnRandomCoin(type: string) {
    // Only spawn coins on winter and dark_village maps
    if (this.mapCode !== 'winter' && this.mapCode !== 'dark_village') {
      return
    }

    const config = COIN_CONFIGS.find(c => c.type === type)
    if (!config) return

    // Guard max limit per type
    if (this.activeCounts[type] >= config.max) {
      return
    }

    const map = this.scene.map
    if (!map) return

    const tileWidth = map.width
    const tileHeight = map.height
    const tileSize = map.tileWidth

    let spawned = false
    let attempts = 0

    const collisionLayerName = this.bootstrap.collision_layer_name || 'Collision'

    while (!spawned && attempts < 100) {
      attempts++
      // Select random coordinate avoiding boundaries
      const tx = Phaser.Math.Between(2, tileWidth - 3)
      const ty = Phaser.Math.Between(2, tileHeight - 3)

      const px = tx * tileSize + tileSize / 2
      const py = ty * tileSize + tileSize / 2

      // 1. Check tilemap collision layer
      const tile = map.getTileAt(tx, ty, true, collisionLayerName)
      if (tile && tile.index > 0) {
        continue // blocked by tilemap collision
      }

      // 2. Check overlap with existing placements via event
      let overlapPlacement = false
      window.dispatchEvent(new CustomEvent('game:checkOccupied', {
        detail: {
          x: tx * tileSize,
          y: ty * tileSize,
          item: {
            code: 'coin_temp',
            metadata_json: '{"collides":true,"collision_w":16,"collision_h":16}'
          },
          callback: (res: boolean) => {
            overlapPlacement = res
          }
        }
      }))

      if (overlapPlacement) {
        continue // blocked by placements
      }

      // 3. Ensure coin is not spawned right under the local player
      const dist = Phaser.Math.Distance.Between(px, py, this.playerSprite.x, this.playerSprite.y)
      if (dist < 64) {
        continue
      }

      // Create coin sprite using type key
      const coin = this.scene.add.sprite(px, py, config.key)
      coin.play(`coin_spin_${type}`)
      coin.setData('coinType', type)

      // Add a slight vertical floating animation
      this.scene.tweens.add({
        targets: coin,
        y: py - 6,
        duration: 1000 + Math.random() * 300,
        yoyo: true,
        repeat: -1,
        ease: 'Sine.easeInOut'
      })

      this.coinsGroup.add(coin)
      this.activeCounts[type]++
      spawned = true
    }
  }

  private async handlePickup(coin: Phaser.GameObjects.Sprite) {
    const body = coin.body as Phaser.Physics.Arcade.StaticBody
    if (body) {
      body.enable = false // Prevent duplicate pickups
    }

    const type = coin.getData('coinType') as string
    this.activeCounts[type] = Math.max(0, this.activeCounts[type] - 1)

    // Play pickup scale/float/fade tween
    this.scene.tweens.add({
      targets: coin,
      y: coin.y - 20,
      scaleX: 0,
      scaleY: 0,
      alpha: 0,
      duration: 200,
      onComplete: () => {
        this.coinsGroup.remove(coin, true, true)
        // Re-spawn another random coin of the same type after 5 seconds
        this.scene.time.delayedCall(5000, () => {
          this.spawnRandomCoin(type)
        })
      }
    })

    // HTTP Claim call to update balance based on coin type
    try {
      const res = await editorService.claimCoinPickup(this.mapCode, type)
      // Notify Vue state store that balance updated
      window.dispatchEvent(new CustomEvent('game:placementDone', {
        detail: { newCoins: res.new_coins }
      }))
    } catch (err) {
      console.error(`Failed to claim coin pickup for type ${type}:`, err)
    }
  }

  public destroy() {
    this.coinsGroup.destroy(true, true)
  }
}
