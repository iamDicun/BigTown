import Phaser from 'phaser'
import type { DecorationItemDto, PlacementDto } from '../services/editor.service'
import * as editorService from '../services/editor.service'
import { PLAYER_DEPTH } from './mapSystem'

export class EditorSystem {
  private scene: Phaser.Scene
  private placementsGroup!: Phaser.GameObjects.Group
  private collisionGroup!: Phaser.Physics.Arcade.StaticGroup
  private activeDecorationItem: DecorationItemDto | null = null
  private previewSprite: Phaser.GameObjects.Image | null = null
  private editorActive = false
  private deleteModeActive = false
  private canPlace = true
  private mapCode: string
  private playerSprite: Phaser.GameObjects.Sprite
  private isBehindDecoration = false

  private onToggleModeHandler!: (e: Event) => void
  private onToggleDeleteModeHandler!: (e: Event) => void
  private onSelectDecorationHandler!: (e: Event) => void
  private onCancelPlacementHandler!: (e: Event) => void
  private onLoadPlacementsHandler!: (e: Event) => void

  constructor(scene: Phaser.Scene, mapCode: string, playerSprite: Phaser.GameObjects.Sprite) {
    this.scene = scene
    this.mapCode = mapCode
    this.playerSprite = playerSprite
    this.initialize(playerSprite)
  }

  private initialize(playerSprite: Phaser.GameObjects.Sprite) {
    this.placementsGroup = this.scene.add.group()
    this.collisionGroup = this.scene.physics.add.staticGroup()

    // Add collider between player and placed items
    this.scene.physics.add.collider(playerSprite, this.collisionGroup)

    this.onToggleModeHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { active: boolean }
      this.editorActive = detail.active
      if (!this.editorActive) {
        this.clearPreview()
        this.setDeleteMode(false)
      }
    }
    window.addEventListener('game:toggleEditorMode', this.onToggleModeHandler)

    this.onToggleDeleteModeHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { active: boolean }
      this.setDeleteMode(detail.active)
    }
    window.addEventListener('game:toggleDeleteMode', this.onToggleDeleteModeHandler)

    this.onSelectDecorationHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { item: DecorationItemDto }
      this.startPlacement(detail.item)
    }
    window.addEventListener('game:selectDecoration', this.onSelectDecorationHandler)

    this.onCancelPlacementHandler = () => {
      this.clearPreview()
    }
    window.addEventListener('game:cancelPlacement', this.onCancelPlacementHandler)

    this.onLoadPlacementsHandler = (e: Event) => {
      const detail = (e as CustomEvent).detail as { placements: PlacementDto[]; items: DecorationItemDto[] }
      this.drawPlacements(detail.placements, detail.items)
    }
    window.addEventListener('game:loadPlacements', this.onLoadPlacementsHandler)

    // Keyboard ESC key to cancel placement or delete mode
    this.scene.input.keyboard?.on('keydown-ESC', () => {
      if (this.activeDecorationItem) {
        window.dispatchEvent(new CustomEvent('game:placementCancel'))
        this.clearPreview()
      } else if (this.deleteModeActive) {
        window.dispatchEvent(new CustomEvent('game:toggleDeleteMode', { detail: { active: false } }))
        this.setDeleteMode(false)
      }
    })

    // Pointer down handler for placing item
    this.scene.input.on('pointerdown', (pointer: Phaser.Input.Pointer) => {
      if (pointer.leftButtonDown() && this.activeDecorationItem && this.previewSprite && !this.deleteModeActive && this.canPlace) {
        this.confirmPlacement()
      }
    })
  }

  private setDeleteMode(active: boolean) {
    this.deleteModeActive = active
    if (this.deleteModeActive) {
      this.clearPreview()
    }
    
    // Update tint of placed items on delete mode change
    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      if (this.deleteModeActive) {
        sprite.setInteractive()
      } else {
        sprite.disableInteractive()
        sprite.clearTint()
      }
    })
  }

  private startPlacement(item: DecorationItemDto) {
    this.clearPreview()
    this.setDeleteMode(false)
    this.activeDecorationItem = item

    // Parse metadata
    let meta: any = {}
    try {
      meta = JSON.parse(item.metadata_json)
    } catch {}

    const loadAndCreatePreview = () => {
      const hasFrames = meta.frameWidth !== undefined && meta.frameHeight !== undefined
      const frameIndex = meta.frame !== undefined ? meta.frame : 0

      if (hasFrames) {
        this.previewSprite = this.scene.add.image(0, 0, item.asset_key, frameIndex)
      } else {
        this.previewSprite = this.scene.add.image(0, 0, item.asset_key)
      }

      this.previewSprite.setAlpha(0.6)
      this.previewSprite.setDepth(100) // Render on top of everything during preview

      if (meta.anchorX !== undefined && meta.anchorY !== undefined) {
        this.previewSprite.setOrigin(meta.anchorX, meta.anchorY)
      } else {
        this.previewSprite.setOrigin(0.5, 1.0) // default anchor at bottom-middle
      }
    }

    // Load preview texture dynamically if not loaded yet
    if (!this.scene.textures.exists(item.asset_key)) {
      if (meta.frameWidth !== undefined && meta.frameHeight !== undefined) {
        this.scene.load.spritesheet(item.asset_key, `/assets/${item.asset_key}`, {
          frameWidth: meta.frameWidth,
          frameHeight: meta.frameHeight
        })
      } else {
        this.scene.load.image(item.asset_key, `/assets/${item.asset_key}`)
      }
      this.scene.load.once('complete', loadAndCreatePreview)
      this.scene.load.start()
    } else {
      loadAndCreatePreview()
    }
  }

  private clearPreview() {
    this.activeDecorationItem = null
    if (this.previewSprite) {
      this.previewSprite.destroy()
      this.previewSprite = null
    }
  }

  private drawPlacements(placements: PlacementDto[], items: DecorationItemDto[]) {
    const safePlacements = placements || []
    const safeItems = items || []
    const itemMap = new Map<string, DecorationItemDto>()
    for (const item of safeItems) {
      itemMap.set(item.id, item)
    }

    // Gather all asset keys that are not loaded in Phaser
    const assetsToLoad = new Map<string, any>()
    for (const p of safePlacements) {
      const item = itemMap.get(p.item_id)
      if (item && !this.scene.textures.exists(item.asset_key)) {
        try {
          const meta = JSON.parse(item.metadata_json)
          assetsToLoad.set(item.asset_key, meta)
        } catch {
          assetsToLoad.set(item.asset_key, {})
        }
      }
    }

    if (assetsToLoad.size > 0) {
      assetsToLoad.forEach((meta, assetKey) => {
        if (meta.frameWidth !== undefined && meta.frameHeight !== undefined) {
          this.scene.load.spritesheet(assetKey, `/assets/${assetKey}`, {
            frameWidth: meta.frameWidth,
            frameHeight: meta.frameHeight
          })
        } else {
          this.scene.load.image(assetKey, `/assets/${assetKey}`)
        }
      })
      this.scene.load.once('complete', () => {
        this.renderPlacementsGroup(safePlacements, itemMap)
      })
      this.scene.load.start()
    } else {
      this.renderPlacementsGroup(safePlacements, itemMap)
    }
  }

  private renderPlacementsGroup(placements: PlacementDto[], itemMap: Map<string, DecorationItemDto>) {
    // Clear old placements sprites
    this.placementsGroup.clear(true, true)
    
    // Clear old static collision group bodies
    this.collisionGroup.clear(true, true)

    const safePlacements = placements || []
    for (const p of safePlacements) {
      const item = itemMap.get(p.item_id)
      if (!item) continue

      let meta: any = {}
      try {
        meta = JSON.parse(item.metadata_json)
      } catch {}

      const hasFrames = meta.frameWidth !== undefined && meta.frameHeight !== undefined
      const frameIndex = meta.frame !== undefined ? meta.frame : 0

      let sprite: Phaser.GameObjects.Image
      if (hasFrames) {
        sprite = this.scene.add.image(p.x, p.y, item.asset_key, frameIndex)
      } else {
        sprite = this.scene.add.image(p.x, p.y, item.asset_key)
      }

      // Store placement ID and item code on the sprite
      sprite.setData('placementId', p.id)
      sprite.setData('itemCode', item.code)
      
      // Depth sorting based on Y coordinate so elements render in front / behind player
      sprite.setDepth(PLAYER_DEPTH + p.y / 10000.0)

      if (meta.anchorX !== undefined && meta.anchorY !== undefined) {
        sprite.setOrigin(meta.anchorX, meta.anchorY)
      } else {
        sprite.setOrigin(0.5, 1.0)
      }

      // Interactive clicks for deletion
      if (this.deleteModeActive) {
        sprite.setInteractive()
      }
      
      sprite.on('pointerover', () => {
        if (this.deleteModeActive) {
          sprite.setTint(0xff5555) // red highlight for deletion
        }
      })

      sprite.on('pointerout', () => {
        sprite.clearTint()
      })

      sprite.on('pointerdown', (pointer: Phaser.Input.Pointer) => {
        if (this.deleteModeActive && pointer.leftButtonDown()) {
          const placementId = sprite.getData('placementId')
          if (placementId) {
            this.confirmDelete(placementId, sprite)
          }
        }
      })

      // Add to main placement rendering group
      this.placementsGroup.add(sprite)
      // Add to collision group if collides is true
      if (meta.collides) {
        this.scene.physics.add.existing(sprite, true)
        const body = sprite.body as Phaser.Physics.Arcade.StaticBody
        
        // Use custom collision sizes from metadata, or fallback to w/h or frame/texture size
        const bodyW = meta.collision_w ?? meta.w ?? (hasFrames ? meta.frameWidth : sprite.width)
        const bodyH = meta.collision_h ?? meta.h ?? (hasFrames ? meta.frameHeight : sprite.height)
        
        body.setSize(bodyW, bodyH)
        
        const spriteW = hasFrames ? meta.frameWidth : sprite.width
        const spriteH = hasFrames ? meta.frameHeight : sprite.height

        // Calculate offset because origin is bottom-middle (0.5, 1.0)
        if (meta.collision_x !== undefined && meta.collision_y !== undefined) {
          body.setOffset(meta.collision_x, meta.collision_y)
        } else {
          const offX = -bodyW / 2 + spriteW * sprite.originX
          const offY = -bodyH + spriteH * sprite.originY
          body.setOffset(offX, offY)
        }
        body.updateFromGameObject()
        
        this.collisionGroup.add(sprite)
      }

      // P7: colliders phụ từ metadata thay cho hardcode bridge
      if (Array.isArray(meta.extra_colliders)) {
        for (const c of meta.extra_colliders) {
          const zone = this.scene.add.zone(p.x + c.dx, p.y + c.dy, c.w, c.h)
          this.scene.physics.add.existing(zone, true)
          this.collisionGroup.add(zone)
        }
      } else {
        // Fallback bridge collision logic cũ nếu metadata chưa có extra_colliders
        if (item.code.startsWith('deco_bridge_h_')) {
          const zoneTop = this.scene.add.zone(p.x, p.y - 28, 48, 8)
          this.scene.physics.add.existing(zoneTop, true)
          this.collisionGroup.add(zoneTop)

          const zoneBottom = this.scene.add.zone(p.x, p.y - 4, 48, 8)
          this.scene.physics.add.existing(zoneBottom, true)
          this.collisionGroup.add(zoneBottom)
        } else if (item.code.startsWith('deco_bridge_v_')) {
          const zoneLeft = this.scene.add.zone(p.x - 20, p.y - 16, 8, 32)
          this.scene.physics.add.existing(zoneLeft, true)
          this.collisionGroup.add(zoneLeft)

          const zoneRight = this.scene.add.zone(p.x + 20, p.y - 16, 8, 32)
          this.scene.physics.add.existing(zoneRight, true)
          this.collisionGroup.add(zoneRight)
        }
      }
    }
  }

  private async confirmPlacement() {
    if (!this.activeDecorationItem || !this.previewSprite) return

    const x = this.previewSprite.x
    const y = this.previewSprite.y
    const itemId = this.activeDecorationItem.id

    // Call placing API
    try {
      const result = await editorService.placeItem({
        item_id: itemId,
        map_code: this.mapCode,
        x,
        y
      })

      // Reload all placements to sync bodies & sprites correctly
      window.dispatchEvent(new CustomEvent('game:placementDone', {
        detail: { newCoins: result.new_coins, placement: result.placement }
      }))

      this.clearPreview()
    } catch (err: any) {
      console.error('Failed to place item:', err)
      window.dispatchEvent(new CustomEvent('game:placementError', {
        detail: { message: err.message || 'Lỗi không xác định khi đặt vật phẩm' }
      }))
      window.dispatchEvent(new CustomEvent('game:placementCancel'))
      this.clearPreview()
    }
  }

  private async confirmDelete(placementId: string, sprite: Phaser.GameObjects.Image) {
    try {
      const res = await editorService.deletePlacement(placementId)

      // Play a small destruction fade effect
      this.scene.tweens.add({
        targets: sprite,
        alpha: 0,
        scaleX: 0.8,
        scaleY: 0.8,
        duration: 150,
        onComplete: () => {
          sprite.destroy()
        }
      })

      // Notify Vue UI to update coins and reload editor placements
      window.dispatchEvent(new CustomEvent('game:placementDone', {
        detail: { newCoins: res.new_coins, deletedId: placementId }
      }))
    } catch (err: any) {
      console.error('Failed to delete placement:', err)
      window.dispatchEvent(new CustomEvent('game:placementError', {
        detail: { message: err.message || 'Lỗi không xác định khi xóa vật phẩm' }
      }))
    }
  }

  public update() {
    if (this.previewSprite && this.activeDecorationItem) {
      const pointer = this.scene.input.activePointer
      const worldPoint = pointer.positionToCamera(this.scene.cameras.main) as Phaser.Math.Vector2

      // Snapping to 16px grid
      const snappedX = Math.round(worldPoint.x / 16) * 16
      const snappedY = Math.round(worldPoint.y / 16) * 16

      this.previewSprite.setPosition(snappedX, snappedY)

      // Check if coordinates are occupied
      let occupied = false
      window.dispatchEvent(new CustomEvent('game:checkOccupied', {
        detail: {
          x: snappedX,
          y: snappedY,
          callback: (res: boolean) => {
            occupied = res
          }
        }
      }))

      if (occupied) {
        this.previewSprite.setTint(0xff5555) // Red warning
        this.canPlace = false
      } else {
        this.previewSprite.clearTint()
        this.canPlace = true
      }
    }

    // Fade placed items when player is behind them (above layer fade)
    const player = this.playerSprite
    const playerBounds = player.getBounds()
    let localBehindDecoration = false

    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      
      // We only fade items that are tall (height > 32) like houses and trees
      if (sprite.height > 32) {
        const spriteBounds = sprite.getBounds()
        const behind = player.y < sprite.y
        const overlap = Phaser.Geom.Intersects.RectangleToRectangle(playerBounds, spriteBounds)
        const isBehind = behind && overlap

        if (isBehind) {
          localBehindDecoration = true
        }

        const targetAlpha = isBehind ? 0.35 : 1.0
        
        // Use smooth tween for fading
        if (sprite.getData('targetAlpha') !== targetAlpha) {
          sprite.setData('targetAlpha', targetAlpha)
          this.scene.tweens.killTweensOf(sprite)
          this.scene.tweens.add({
            targets: sprite,
            alpha: targetAlpha,
            duration: 150,
            ease: 'Power1'
          })
        }
      }
    })

    this.isBehindDecoration = localBehindDecoration
  }

  public isPlayerBehindDecoration(): boolean {
    return this.isBehindDecoration
  }

  public isPlayerOnBridge(): boolean {
    let onBridge = false
    const playerBounds = this.playerSprite.getBounds()

    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      const itemCode = sprite.getData('itemCode') as string
      if (itemCode && itemCode.startsWith('deco_bridge_')) {
        const spriteBounds = sprite.getBounds()
        if (Phaser.Geom.Intersects.RectangleToRectangle(playerBounds, spriteBounds)) {
          onBridge = true
        }
      }
    })
    return onBridge
  }

  public destroy() {
    window.removeEventListener('game:toggleEditorMode', this.onToggleModeHandler)
    window.removeEventListener('game:toggleDeleteMode', this.onToggleDeleteModeHandler)
    window.removeEventListener('game:selectDecoration', this.onSelectDecorationHandler)
    window.removeEventListener('game:cancelPlacement', this.onCancelPlacementHandler)
    window.removeEventListener('game:loadPlacements', this.onLoadPlacementsHandler)
    this.placementsGroup.destroy(true, true)
    this.collisionGroup.destroy(true, true)
    this.clearPreview()
  }
}
