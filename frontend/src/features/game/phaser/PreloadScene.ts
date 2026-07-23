import Phaser from 'phaser'

import type { GameSceneData } from './BootScene'
import { gameSceneKey } from './GameScene'
import { getCharacterSpriteUrl } from '../services/character.service'

export const preloadSceneKey = 'preload'

export class PreloadScene extends Phaser.Scene {
  private sceneData!: GameSceneData

  constructor() {
    super(preloadSceneKey)
  }

  init(data: GameSceneData) {
    this.sceneData = data
  }

  preload() {
    const bootstrap = this.sceneData.bootstrap
    const options = this.sceneData.characterOptions

    this.load.tilemapTiledJSON('map', `/assets/${bootstrap.tilemap_asset_key}`)

    for (const tilesetName of bootstrap.tileset_asset_key.split(',')) {
      this.load.image(tilesetName, `/assets/tiles/${tilesetName}.png`)
    }

    for (const option of options) {
      this.load.spritesheet(
        option.base_asset_key,
        getCharacterSpriteUrl(option.base_asset_key),
        { frameWidth: option.spritesheet.frame_width, frameHeight: option.spritesheet.frame_height },
      )
    }
  }

  create() {
    this.scene.start(gameSceneKey, this.sceneData)
  }
}
