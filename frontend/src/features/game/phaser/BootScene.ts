import Phaser from 'phaser'

import type { CharacterOptionDto, SpritesheetConfigDto } from '../services/character.service'
import type { BootstrapDto } from '../services/realtime.service'
import { preloadSceneKey } from './PreloadScene'

export const bootSceneKey = 'boot'

export type GameSceneData = {
  bootstrap: BootstrapDto
  characterId: string
  baseAssetKey: string
  textureKey: string
  spritesheetConfig: SpritesheetConfigDto
  characterOptions: CharacterOptionDto[]
  warpX?: number
  warpY?: number
}

export class BootScene extends Phaser.Scene {
  private sceneData!: GameSceneData

  constructor() {
    super(bootSceneKey)
  }

  init(data: GameSceneData) {
    this.sceneData = data
  }

  create() {
    this.scene.start(preloadSceneKey, this.sceneData)
  }
}
