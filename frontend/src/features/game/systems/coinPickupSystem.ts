import Phaser from 'phaser'
import * as editorService from '../services/editor.service'
import type { BootstrapDto } from '../services/realtime.service'
import type { GameScene } from '../phaser/GameScene'

type CoinConfig = {
  type: string
  key: string
}

const COIN_CONFIGS: CoinConfig[] = [
  { type: 'gri', key: 'coin_gri' },
  { type: 'ama', key: 'coin_ama' },
  { type: 'azu', key: 'coin_azu' },
  { type: 'roj', key: 'coin_roj' },
  { type: 'gold', key: 'coin_gems' }
]

export class CoinPickupSystem {
  private scene: GameScene
  private mapCode: string
  private playerSprite: Phaser.GameObjects.Sprite
  private coinsGroup!: Phaser.Physics.Arcade.StaticGroup
  private coinsMap = new Map<string, Phaser.GameObjects.Sprite>()

  constructor(scene: GameScene, bootstrap: BootstrapDto, playerSprite: Phaser.GameObjects.Sprite) {
    this.scene = scene
    this.mapCode = bootstrap.map_code
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
  }

  public renderCoins(coins: editorService.SpawnedCoinDto[]) {
    // Evict all existing coins
    this.coinsMap.forEach((coinSprite) => {
      this.coinsGroup.remove(coinSprite, true, true)
    })
    this.coinsMap.clear()

    if (this.mapCode !== 'winter' && this.mapCode !== 'dark_village') {
      return
    }

    coins.forEach((c) => this.addCoin(c))
  }

  public addCoin(coin: editorService.SpawnedCoinDto) {
    if (this.coinsMap.has(coin.id)) return

    const config = COIN_CONFIGS.find((c) => c.type === coin.type)
    if (!config) return

    // Client-side filtering: skip coins that spawn on blocked tiles (P1)
    const map = this.scene.map
    if (map) {
      const tx = Math.floor(coin.x / map.tileWidth)
      const ty = Math.floor(coin.y / map.tileHeight)
      const collisionLayerName = 'Collision'
      const tile = map.getTileAt(tx, ty, true, collisionLayerName)
      if (tile && tile.index > 0) {
        return
      }
    }

    // Position coordinate relative to center of tile
    const px = coin.x + this.scene.map!.tileWidth / 2
    const py = coin.y + this.scene.map!.tileHeight / 2

    const coinSprite = this.scene.add.sprite(px, py, config.key)
    coinSprite.play(`coin_spin_${coin.type}`)
    coinSprite.setData('id', coin.id)
    coinSprite.setData('type', coin.type)

    // Slight vertical float animation
    this.scene.tweens.add({
      targets: coinSprite,
      y: py - 6,
      duration: 1000 + Math.random() * 300,
      yoyo: true,
      repeat: -1,
      ease: 'Sine.easeInOut'
    })

    this.coinsGroup.add(coinSprite)
    this.coinsMap.set(coin.id, coinSprite)
  }

  public removeCoin(coinId: string) {
    const coinSprite = this.coinsMap.get(coinId)
    if (!coinSprite) return

    this.coinsMap.delete(coinId)
    const body = coinSprite.body as Phaser.Physics.Arcade.StaticBody
    if (body) {
      body.enable = false // Prevent duplicate overlaps on prediction fade
    }

    // Play float and fade locally
    this.scene.tweens.add({
      targets: coinSprite,
      y: coinSprite.y - 20,
      scaleX: 0,
      scaleY: 0,
      alpha: 0,
      duration: 200,
      onComplete: () => {
        this.coinsGroup.remove(coinSprite, true, true)
      }
    })
  }

  private async handlePickup(coinSprite: Phaser.GameObjects.Sprite) {
    const coinId = coinSprite.getData('id') as string
    if (!coinId) return

    // Prediction: immediately hide coin locally
    this.removeCoin(coinId)

    try {
      const res = await editorService.claimCoinPickup(this.mapCode, coinId)
      // Notify Vue state store that balance updated
      window.dispatchEvent(new CustomEvent('game:placementDone', {
        detail: { newCoins: res.new_coins }
      }))
    } catch (err) {
      console.error(`Failed to claim coin pickup for ID ${coinId}:`, err)
    }
  }

  public destroy() {
    this.coinsMap.clear()
    this.coinsGroup.destroy(true, true)
  }
}
