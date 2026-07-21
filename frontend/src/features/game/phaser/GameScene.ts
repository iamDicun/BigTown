import Phaser from 'phaser'

import { createGameSocket, getDefaultRealtimeUrl, type GameSocket } from '../network/gameSocket'
import { buildMap, TILE_SIZE } from '../systems/mapSystem'
import { LocalPlayerController, type MovementKeys } from '../systems/localPlayerController'
import { RemotePlayerManager } from '../systems/remotePlayerManager'
import { createAboveLayerFade, updateAboveLayerFade, type AboveLayerFade } from '../systems/aboveLayerFadeSystem'
import type { GameSceneData } from './BootScene'
import { createPlayerAnimations } from './playerAnimations'

export const gameSceneKey = 'game'

const CAMERA_ZOOM = 2

// GameScene chỉ đóng vai trò orchestrator: dựng map/player/remote-players qua các system riêng
// (systems/mapSystem.ts, systems/localPlayerController.ts, systems/remotePlayerManager.ts) và nối
// dây sự kiện realtime (network/gameSocket.ts). Khi thêm tính năng mới (NPC, combat, HP bar, chat
// bubble...), thêm system mới rồi wire vào đây — không nhét thẳng logic vào file này để tránh
// phình to, xem docs/Phaser-Frontend-Guide.md mục 19 ("khi dài ra thì tách sang systems/").
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
    const bootstrap = this.sceneData.bootstrap
    this.localCharacterId = this.sceneData.characterId

    const { collisionGroup, aboveLayer } = buildMap(this, bootstrap)
    this.aboveLayerFade = aboveLayer ? createAboveLayerFade(aboveLayer) : null
    createPlayerAnimations(this)

    this.localPlayer = new LocalPlayerController(this, bootstrap.spawn_x, bootstrap.spawn_y, (command) =>
      this.gameSocket?.sendMove(command).catch(() => {
        // RPC lỗi (mất kết nối tạm thời) — bỏ qua, network tick tiếp theo sẽ tự gửi lại vị trí mới nhất.
      }),
    )
    this.physics.add.collider(this.localPlayer.sprite, collisionGroup)
    this.localPlayer.sprite.setCollideWorldBounds(true)

    this.remotePlayers = new RemotePlayerManager(this)
    // Chặn hình ảnh local player đè lên remote player ngay lập tức (không đợi BE correction) — xem
    // ghi chú trong remotePlayerManager.ts. Vị trí hợp lệ cuối cùng vẫn do BE quyết định qua RPC.
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
            continue
          }
          this.remotePlayers.upsert(p.characterId, p.x, p.y, p.direction, p.moving)
        }
      },
      onPlayerJoined: (event) => {
        if (event.player.characterId === this.localCharacterId) return
        this.remotePlayers.upsert(event.player.characterId, event.player.x, event.player.y, event.player.direction, event.player.moving)
      },
      onPlayerLeft: (event) => this.remotePlayers.remove(event.characterId),
      onPlayerMove: (event) => {
        if (event.characterId === this.localCharacterId) return
        this.remotePlayers.upsert(event.characterId, event.x, event.y, event.direction, event.moving)
      },
      onCorrection: (event) => this.localPlayer.applyCorrection(event.x, event.y),
    })

    // GameCanvas.vue chỉ gọi game.destroy() — Centrifuge connection không tự đóng theo, phải
    // đóng tường minh lúc scene shutdown để tránh leak connection khi rời GameView.
    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.gameSocket?.close()
      this.gameSocket = null
      this.remotePlayers.destroyAll()
    })
  }

  update(time: number) {
    this.localPlayer.update(time, this.cursors)
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
