import Phaser from 'phaser'

import { createGameSocket, getDefaultRealtimeUrl, type GameSocket } from '../network/gameSocket'
import { buildMap, PLAYER_DEPTH, type WarpZone } from '../systems/mapSystem'
import { LocalPlayerController, type MovementKeys } from '../systems/localPlayerController'
import { RemotePlayerManager } from '../systems/remotePlayerManager'
import { createAboveLayerFade, updateAboveLayerFade, type AboveLayerFade } from '../systems/aboveLayerFadeSystem'
import type { GameSceneData } from './BootScene'
import { createAnimations } from './playerAnimations'
import * as realtimeService from '../services/realtime.service'

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
  private warpZones: WarpZone[] = []
  private warping = false
  private switchMapHandler: ((e: Event) => void) | null = null

  constructor() {
    super(gameSceneKey)
  }

  init(data: GameSceneData) {
    this.sceneData = data
    this.warping = false
  }

  create() {
    const { bootstrap, characterId, textureKey, spritesheetConfig, characterOptions } = this.sceneData
    this.localCharacterId = characterId

    const { collisionGroup, aboveLayer, warpZones } = buildMap(this, bootstrap)
    this.warpZones = warpZones
    this.aboveLayerFade = aboveLayer ? createAboveLayerFade(aboveLayer) : null

    for (const option of characterOptions) {
      createAnimations(this, option.base_asset_key, option.spritesheet)
    }

    this.localPlayer = new LocalPlayerController(
      this,
      textureKey,
      this.sceneData.warpX ?? bootstrap.spawn_x,
      this.sceneData.warpY ?? bootstrap.spawn_y,
      (command) =>
        this.gameSocket?.sendMove(command).catch(() => {
        }),
      spritesheetConfig,
    )
    this.localPlayer.sprite.setDepth(PLAYER_DEPTH)
    this.physics.add.collider(this.localPlayer.sprite, collisionGroup)
    this.localPlayer.sprite.setCollideWorldBounds(true)

    this.remotePlayers = new RemotePlayerManager(this, characterOptions)

    this.physics.add.collider(this.localPlayer.sprite, this.remotePlayers.group)

    this.setupCamera(bootstrap.map_width, bootstrap.map_height, bootstrap.tile_size)
    const keyboard = this.input.keyboard!
    const upArr = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.UP)
    const downArr = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.DOWN)
    const leftArr = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.LEFT)
    const rightArr = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.RIGHT)
    const upW = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.W)
    const downW = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.S)
    const leftW = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.A)
    const rightW = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.D)
    this.cursors = {
      get up() { return { isDown: upArr.isDown || upW.isDown } as Phaser.Input.Keyboard.Key },
      get down() { return { isDown: downArr.isDown || downW.isDown } as Phaser.Input.Keyboard.Key },
      get left() { return { isDown: leftArr.isDown || leftW.isDown } as Phaser.Input.Keyboard.Key },
      get right() { return { isDown: rightArr.isDown || rightW.isDown } as Phaser.Input.Keyboard.Key },
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
          this.remotePlayers.upsert(p.characterId, p.x, p.y, p.direction, p.moving, p.baseAssetKey, p.name)
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
          event.player.baseAssetKey,
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
      if (this.switchMapHandler) {
        window.removeEventListener('game:switchMap', this.switchMapHandler)
        this.switchMapHandler = null
      }
    })

    this.switchMapHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { mapCode: string }
      if (detail?.mapCode && !this.warping) {
        this.switchToMap(detail.mapCode)
      }
    }
    window.addEventListener('game:switchMap', this.switchMapHandler)
  }

  update(time: number) {
    this.localPlayer.update(time, this.cursors)
    this.remotePlayers.update()
    if (this.aboveLayerFade) {
      updateAboveLayerFade(this, this.aboveLayerFade, this.localPlayer.sprite)
    }
    this.checkWarps()
  }

  private checkWarps() {
    if (this.warping || !this.gameSocket) return
    const px = this.localPlayer.sprite.x
    const py = this.localPlayer.sprite.y
    for (const w of this.warpZones) {
      const bounds = (w.zone.body as Phaser.Physics.Arcade.StaticBody)
      if (px >= bounds.x && px <= bounds.x + bounds.width && py >= bounds.y && py <= bounds.y + bounds.height) {
        this.startWarp(w)
        return
      }
    }
  }

  private async startWarp(warp: WarpZone) {
    this.warping = true
    try {
      await this.gameSocket!.centrifuge.rpc('player_warp', { dest_map: warp.destMap, dest_x: warp.destX, dest_y: warp.destY })
      this.gameSocket?.close()
      this.gameSocket = null
      this.remotePlayers.destroyAll()

      const newBootstrap = await realtimeService.getBootstrap(warp.destMap)
      window.dispatchEvent(new CustomEvent('game:mapChanged', { detail: { mapCode: warp.destMap } }))

      await this.preloadMapAssets(newBootstrap)

      this.scene.restart({
        bootstrap: newBootstrap,
        characterId: this.sceneData.characterId,
        baseAssetKey: this.sceneData.baseAssetKey,
        textureKey: this.sceneData.textureKey,
        spritesheetConfig: this.sceneData.spritesheetConfig,
        characterOptions: this.sceneData.characterOptions,
        warpX: warp.destX,
        warpY: warp.destY,
      })
    } catch {
      this.warping = false
    }
  }

  private preloadMapAssets(bootstrap: realtimeService.BootstrapDto): Promise<void> {
    return new Promise((resolve) => {
      if (this.cache?.tilemap?.exists?.('map')) this.cache.tilemap.remove('map')
      this.load.tilemapTiledJSON('map', `/assets/${bootstrap.tilemap_asset_key}`)
      for (const tilesetName of bootstrap.tileset_asset_key.split(',')) {
        this.load.image(tilesetName, `/assets/tiles/${tilesetName}.png`)
      }
      this.load.once('complete', resolve)
      this.load.start()
    })
  }

  private async switchToMap(mapCode: string) {
    this.warping = true
    try {
      const newBootstrap = await realtimeService.getBootstrap(mapCode)
      window.dispatchEvent(new CustomEvent('game:mapChanged', { detail: { mapCode } }))

      if (this.gameSocket) {
        await this.gameSocket.centrifuge.rpc('player_warp', {
          dest_map: mapCode,
          dest_x: newBootstrap.spawn_x,
          dest_y: newBootstrap.spawn_y,
        })
      }

      await this.preloadMapAssets(newBootstrap)

      this.gameSocket?.close()
      this.gameSocket = null
      this.remotePlayers.destroyAll()

      this.scene.restart({
        bootstrap: newBootstrap,
        characterId: this.sceneData.characterId,
        baseAssetKey: this.sceneData.baseAssetKey,
        textureKey: this.sceneData.textureKey,
        spritesheetConfig: this.sceneData.spritesheetConfig,
        characterOptions: this.sceneData.characterOptions,
        warpX: newBootstrap.spawn_x,
        warpY: newBootstrap.spawn_y,
      })
    } catch (err) {
      console.error('Failed to switch map:', err)
      this.warping = false
    }
  }

  private setupCamera(mapWidthTiles: number, mapHeightTiles: number, tileSize: number) {
    const widthPx = mapWidthTiles * tileSize
    const heightPx = mapHeightTiles * tileSize

    this.cameras.main.setBounds(0, 0, widthPx, heightPx)
    this.cameras.main.startFollow(this.localPlayer.sprite, true)
    this.cameras.main.setZoom(CAMERA_ZOOM)
    this.physics.world.setBounds(0, 0, widthPx, heightPx)
  }
}
