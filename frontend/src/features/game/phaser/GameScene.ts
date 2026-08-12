import Phaser from 'phaser'

import { createGameSocket, getDefaultRealtimeUrl, type GameSocket } from '../network/gameSocket'
import { buildMap, PLAYER_DEPTH, type WarpZone } from '../systems/mapSystem'
import { LocalPlayerController, type MovementKeys } from '../systems/localPlayerController'
import { RemotePlayerManager } from '../systems/remotePlayerManager'
import { createAboveLayerFade, updateAboveLayerFade, type AboveLayerFade } from '../systems/aboveLayerFadeSystem'
import { createEnvironmentFx, updateEnvironmentFx, destroyEnvironmentFx, syncWorldTime, type EnvironmentFx } from '../systems/environmentSystem'
import type { GameSceneData } from './BootScene'
import { preloadSceneKey } from './PreloadScene'
import { createAnimations } from './playerAnimations'
import * as realtimeService from '../services/realtime.service'
import { playMusic } from '@/shared/audio/audio.service'
import { EditorSystem } from '../systems/editorSystem'
import { CoinPickupSystem } from '../systems/coinPickupSystem'
import { AnimalSystem } from '../systems/animalSystem'
import { HelpOverlay } from '../systems/helpOverlay'
import type { SpawnedCoinDto } from '../services/editor.service'

export const gameSceneKey = 'game'

const CAMERA_ZOOM = 2

const EMPTY_CURSORS: MovementKeys = {
  up: { isDown: false } as Phaser.Input.Keyboard.Key,
  down: { isDown: false } as Phaser.Input.Keyboard.Key,
  left: { isDown: false } as Phaser.Input.Keyboard.Key,
  right: { isDown: false } as Phaser.Input.Keyboard.Key,
}

export class GameScene extends Phaser.Scene {
  private sceneData!: GameSceneData
  private localCharacterId = ''
  private cursors!: MovementKeys

  private localPlayer!: LocalPlayerController
  private remotePlayers!: RemotePlayerManager
  private aboveLayerFade: AboveLayerFade | null = null
  private environmentFx: EnvironmentFx = { type: 'none' }
  private gameSocket: GameSocket | null = null
  private warpZones: WarpZone[] = []
  private warping = false
  private switchMapHandler: ((e: Event) => void) | null = null
  private chatFocusHandler: ((e: Event) => void) | null = null
  private loadPlacementsHandler: ((e: Event) => void) | null = null
  private chatFocused = false
  private envSyncHandler: (() => void) | null = null
  private enterKey!: Phaser.Input.Keyboard.Key
  private movementKeyCodes: number[] = []
  private editorSystem!: EditorSystem
  private animalSystem!: AnimalSystem
  private helpOverlay!: HelpOverlay
  private mapCollider!: Phaser.Physics.Arcade.Collider
  private coinPickupSystem: CoinPickupSystem | null = null
  public map!: Phaser.Tilemaps.Tilemap

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

    const { map, collisionGroup, aboveLayer, warpZones } = buildMap(this, bootstrap)
    this.map = map
    this.warpZones = warpZones
    this.aboveLayerFade = aboveLayer ? createAboveLayerFade(this, aboveLayer) : null
    this.input.mouse?.disableContextMenu()

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
    this.mapCollider = this.physics.add.collider(this.localPlayer.sprite, collisionGroup)
    this.localPlayer.sprite.setCollideWorldBounds(true)

    this.remotePlayers = new RemotePlayerManager(this, characterOptions)

    this.physics.add.collider(this.localPlayer.sprite, this.remotePlayers.group)

    this.editorSystem = new EditorSystem(this, bootstrap.map_code, this.localPlayer.sprite, bootstrap.tile_size)

    this.animalSystem = new AnimalSystem(this)
    if (bootstrap.npc_spawns) {
      this.animalSystem.spawnFromBootstrap(bootstrap.npc_spawns)
    }

    if (bootstrap.map_code === 'winter' || bootstrap.map_code === 'dark_village') {
      this.coinPickupSystem = new CoinPickupSystem(this, bootstrap, this.localPlayer.sprite)
    }

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

    this.movementKeyCodes = [
      Phaser.Input.Keyboard.KeyCodes.UP, Phaser.Input.Keyboard.KeyCodes.DOWN,
      Phaser.Input.Keyboard.KeyCodes.LEFT, Phaser.Input.Keyboard.KeyCodes.RIGHT,
      Phaser.Input.Keyboard.KeyCodes.W, Phaser.Input.Keyboard.KeyCodes.A,
      Phaser.Input.Keyboard.KeyCodes.S, Phaser.Input.Keyboard.KeyCodes.D,
    ]

    this.enterKey = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.ENTER)

    const hKey = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.H)
    this.helpOverlay = new HelpOverlay(this)
    hKey.on('down', () => {
      if ((document.activeElement as HTMLElement)?.tagName === 'INPUT') return
      this.helpOverlay.toggle()
    })

    this.chatFocusHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { focused: boolean }
      this.chatFocused = detail.focused
      this.setChatInputCapture(detail.focused)
    }
    window.addEventListener('game:chatFocus', this.chatFocusHandler)

    this.loadPlacementsHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { spawned_coins?: SpawnedCoinDto[] }
      if (detail.spawned_coins && this.coinPickupSystem) {
        this.coinPickupSystem.renderCoins(detail.spawned_coins)
      }
    }
    window.addEventListener('game:loadPlacements', this.loadPlacementsHandler)

    this.input.on('pointerdown', () => {
      if (this.chatFocused) {
        window.dispatchEvent(new CustomEvent('game:blurChatInput'))
      }
    })

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
      onRoomState: (event) => {
        for (const p of event.players) {
          if (p.characterId === this.localCharacterId) continue
          this.remotePlayers.upsert(p.characterId, p.x, p.y, p.direction, p.moving)
        }
      },
      onDecorationPlaced: (event) => {
        this.editorSystem.upsertPlacement(event.placement)
        window.dispatchEvent(new CustomEvent('game:realtimePlacementPlaced', {
          detail: { placement: event.placement }
        }))
      },
      onDecorationDeleted: (event) => {
        this.editorSystem.removePlacementSprite(event.placementId)
        window.dispatchEvent(new CustomEvent('game:realtimePlacementDeleted', {
          detail: { placementId: event.placementId }
        }))
      },
      onCoinSpawned: (event) => {
        this.coinPickupSystem?.addCoin(event.coin)
      },
      onCoinPicked: (event) => {
        this.coinPickupSystem?.removeCoin(event.coinId)
      },
      onCorrection: (event) => this.localPlayer.applyCorrection(event.x, event.y),
    })

    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.environmentFx = destroyEnvironmentFx(this.environmentFx)
      if (this.envSyncHandler) {
        document.removeEventListener('visibilitychange', this.envSyncHandler)
        window.removeEventListener('focus', this.envSyncHandler)
        this.envSyncHandler = null
      }
      this.gameSocket?.close()
      this.gameSocket = null
      this.remotePlayers.destroyAll()
      this.editorSystem.destroy()
      this.animalSystem.destroy()
      if (this.coinPickupSystem) {
        this.coinPickupSystem.destroy()
        this.coinPickupSystem = null
      }
      this.helpOverlay.destroy()
      if (this.switchMapHandler) {
        window.removeEventListener('game:switchMap', this.switchMapHandler)
        this.switchMapHandler = null
      }
      if (this.chatFocusHandler) {
        window.removeEventListener('game:chatFocus', this.chatFocusHandler)
        this.chatFocusHandler = null
      }
      if (this.loadPlacementsHandler) {
        window.removeEventListener('game:loadPlacements', this.loadPlacementsHandler)
        this.loadPlacementsHandler = null
      }
    })

    this.switchMapHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { mapCode: string }
      if (detail?.mapCode && !this.warping) {
        this.switchToMap(detail.mapCode)
      }
    }
    window.addEventListener('game:switchMap', this.switchMapHandler)

    if (bootstrap.music_asset_key) {
      playMusic(bootstrap.music_asset_key)
    }

    this.environmentFx = createEnvironmentFx(this, bootstrap.map_code)

    syncWorldTime()
    this.envSyncHandler = () => syncWorldTime()
    document.addEventListener('visibilitychange', this.envSyncHandler)
    window.addEventListener('focus', this.envSyncHandler)

    window.dispatchEvent(new CustomEvent('game:ready'))
  }

  update(time: number) {
    if (Phaser.Input.Keyboard.JustDown(this.enterKey) && !this.chatFocused) {
      window.dispatchEvent(new CustomEvent('game:focusChatInput'))
      this.chatFocused = true
      this.setChatInputCapture(true)
    }

    const cursors = this.chatFocused ? EMPTY_CURSORS : this.cursors
    this.localPlayer.update(time, cursors)
    this.localPlayer.sprite.setDepth(PLAYER_DEPTH + (this.localPlayer.sprite.y + 16) / 10000.0)
    this.remotePlayers.update()
    if (this.mapCollider) {
      this.mapCollider.active = !this.editorSystem.isPlayerOnBridge()
    }
    this.editorSystem.update()
    this.animalSystem.update(time, this.game.loop.delta)
    if (this.aboveLayerFade) {
      const underPlacement = this.editorSystem.isPlayerBehindDecoration()
      updateAboveLayerFade(this, this.aboveLayerFade, this.localPlayer.sprite, time, underPlacement)
    }
    updateEnvironmentFx(this.environmentFx, time, this.localPlayer.sprite)
    this.checkWarps()
  }

  private setChatInputCapture(focused: boolean): void {
    const kb = this.input?.keyboard
    if (!kb || !kb.enabled) return
    for (const code of this.movementKeyCodes) {
      if (focused) {
        kb.removeCapture?.(code)
      } else {
        kb.addCapture?.(code)
      }
    }
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

      if (this.chatFocusHandler) {
        window.removeEventListener('game:chatFocus', this.chatFocusHandler)
        this.chatFocusHandler = null
      }
      if (this.switchMapHandler) {
        window.removeEventListener('game:switchMap', this.switchMapHandler)
        this.switchMapHandler = null
      }

      this.scene.start(preloadSceneKey, {
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

      this.gameSocket?.close()
      this.gameSocket = null
      this.remotePlayers.destroyAll()

      if (this.chatFocusHandler) {
        window.removeEventListener('game:chatFocus', this.chatFocusHandler)
        this.chatFocusHandler = null
      }
      if (this.switchMapHandler) {
        window.removeEventListener('game:switchMap', this.switchMapHandler)
        this.switchMapHandler = null
      }

      this.scene.start(preloadSceneKey, {
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
