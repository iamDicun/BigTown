import Phaser from 'phaser'

import { createGameSocket, getDefaultRealtimeUrl, type GameSocket } from '../network/gameSocket'
import { buildMap, TILE_SIZE } from '../systems/mapSystem'
import { LocalPlayerController, type MovementKeys } from '../systems/localPlayerController'
import { RemotePlayerManager } from '../systems/remotePlayerManager'
import { createAboveLayerFade, updateAboveLayerFade, type AboveLayerFade } from '../systems/aboveLayerFadeSystem'
import type { GameSceneData } from './BootScene'
import { createAnimations } from './playerAnimations'

export const gameSceneKey = 'game'

const CAMERA_ZOOM = 2

export class GameScene extends Phaser.Scene {
  private sceneData!: GameSceneData
  private localCharacterId = ''
  private cursors!: MovementKeys

  private localPlayer!: LocalPlayerController
  private remotePlayers!: RemotePlayerManager
  private aboveLayerFade: AboveLayerFade | null = null
  private gameSocket: GameSocket | null = null

  constructor() {
    super(gameSceneKey)
  }

  init(data: GameSceneData) {
    this.sceneData = data
  }

  create() {
    const { bootstrap, characterId, textureKey, spritesheetConfig } = this.sceneData
    this.localCharacterId = characterId

    const { collisionGroup, aboveLayer } = buildMap(this, bootstrap)
    this.aboveLayerFade = aboveLayer ? createAboveLayerFade(aboveLayer) : null
    createAnimations(this, textureKey, spritesheetConfig)

    this.localPlayer = new LocalPlayerController(
      this,
      textureKey,
      bootstrap.spawn_x,
      bootstrap.spawn_y,
      (command) =>
        this.gameSocket?.sendMove(command).catch(() => {
          // RPC lỗi (mất kết nối tạm thời) — bỏ qua, network tick tiếp theo sẽ tự gửi lại vị trí mới nhất.
        }),
      spritesheetConfig,
    )
    this.physics.add.collider(this.localPlayer.sprite, collisionGroup)
    this.localPlayer.sprite.setCollideWorldBounds(true)

    this.remotePlayers = new RemotePlayerManager(this, textureKey, spritesheetConfig)
    this.physics.add.collider(this.localPlayer.sprite, this.remotePlayers.group)

    this.setupCamera(bootstrap.map_width, bootstrap.map_height)
    const keyboard = this.input.keyboard!
    this.cursors = {
      up: keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.UP),
      down: keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.DOWN),
      left: keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.LEFT),
      right: keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.RIGHT),
    }

    this.gameSocket = createGameSocket(getDefaultRealtimeUrl(), {
      channel: bootstrap.default_channel,
      onRoomSnapshot: (data) => {
        for (const p of data.players) {
          if (p.characterId === this.localCharacterId) {
            this.localPlayer.applyServerPosition(p.x, p.y, p.direction)
            this.localPlayer.setName(p.name)
            continue
          }
          this.remotePlayers.upsert(p.characterId, p.x, p.y, p.direction, p.moving, p.name)
        }
      },
      onPlayerJoined: (event) => {
        if (event.player.characterId === this.localCharacterId) return
        this.remotePlayers.upsert(
          event.player.characterId,
          event.player.x,
          event.player.y,
          event.player.direction,
          event.player.moving,
          event.player.name,
        )
      },
      onPlayerLeft: (event) => this.remotePlayers.remove(event.characterId),
      onPlayerMove: (event) => {
        if (event.characterId === this.localCharacterId) return
        this.remotePlayers.upsert(event.characterId, event.x, event.y, event.direction, event.moving)
      },
      onCorrection: (event) => this.localPlayer.applyCorrection(event.x, event.y),
    })

    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.gameSocket?.close()
      this.gameSocket = null
      this.remotePlayers.destroyAll()
    })
  }

  update(time: number) {
    this.localPlayer.update(time, this.cursors)
    this.remotePlayers.update()
    if (this.aboveLayerFade) {
      updateAboveLayerFade(this, this.aboveLayerFade, this.localPlayer.sprite)
    }
  }

  private setupCamera(mapWidthTiles: number, mapHeightTiles: number) {
    const widthPx = mapWidthTiles * TILE_SIZE
    const heightPx = mapHeightTiles * TILE_SIZE

    this.cameras.main.setBounds(0, 0, widthPx, heightPx)
    this.cameras.main.startFollow(this.localPlayer.sprite, true)
    this.cameras.main.setZoom(CAMERA_ZOOM)
    this.physics.world.setBounds(0, 0, widthPx, heightPx)
  }
}
