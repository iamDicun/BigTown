import Phaser from 'phaser'

import type { SpritesheetConfigDto } from '../services/character.service'
import type { BootstrapDto } from '../services/realtime.service'
import { bootSceneKey, BootScene } from './BootScene'
import { PreloadScene } from './PreloadScene'
import { GameScene } from './GameScene'

export function createGame(
  parent: HTMLElement,
  bootstrap: BootstrapDto,
  characterId: string,
  baseAssetKey: string,
  textureKey: string,
  spritesheetConfig: SpritesheetConfigDto,
): Phaser.Game {
  const game = new Phaser.Game({
    type: Phaser.AUTO,
    parent,
    width: parent.clientWidth || 960,
    height: parent.clientHeight || 540,
    backgroundColor: '#1d2a1d',
    pixelArt: true,
    physics: {
      default: 'arcade',
      arcade: { debug: false },
    },
    scale: {
      mode: Phaser.Scale.RESIZE,
      autoCenter: Phaser.Scale.CENTER_BOTH,
    },
    scene: [BootScene, PreloadScene, GameScene],
  })

  game.scene.start(bootSceneKey, { bootstrap, characterId, baseAssetKey, textureKey, spritesheetConfig })

  return game
}
