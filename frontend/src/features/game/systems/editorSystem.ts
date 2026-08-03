import Phaser from 'phaser'
import type { DecorationItemDto, PlacementDto } from '../services/editor.service'
import * as editorService from '../services/editor.service'
import { getDarkness } from './effects/common'

const DECORATION_DEPTH = 3

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
  private tileSize: number

  private onToggleModeHandler!: (e: Event) => void
  private onToggleDeleteModeHandler!: (e: Event) => void
  private onSelectDecorationHandler!: (e: Event) => void
  private onCancelPlacementHandler!: (e: Event) => void
  private onLoadPlacementsHandler!: (e: Event) => void

  constructor(scene: Phaser.Scene, mapCode: string, playerSprite: Phaser.GameObjects.Sprite, tileSize = 16) {
    this.scene = scene
    this.mapCode = mapCode
    this.playerSprite = playerSprite
    this.tileSize = tileSize
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
    if (!this.scene || !this.scene.sys || !this.scene.sys.isActive() || !this.scene.load) {
      return
    }
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

      // Scale preview sprite if asset width is smaller than map tile size
      const assetW = hasFrames ? meta.frameWidth : this.previewSprite.width
      const scale = assetW < this.tileSize ? this.tileSize / assetW : 1.0
      this.previewSprite.setScale(scale)
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
    if (!this.scene || !this.scene.sys || !this.scene.sys.isActive() || !this.scene.load) {
      return
    }
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
    const safePlacements = placements || []
    
    // 1. Gather currently rendered sprites by placementId
    const existingSprites = new Map<string, Phaser.GameObjects.Image>()
    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      const pid = sprite.getData('placementId') as string
      if (pid) {
        existingSprites.set(pid, sprite)
      }
    })

    // 2. Diff and reconcile incoming placements
    for (const p of safePlacements) {
      const item = itemMap.get(p.item_id)
      if (!item) continue

      let meta: any = {}
      try {
        meta = JSON.parse(item.metadata_json)
      } catch {}

      const hasFrames = meta.frameWidth !== undefined && meta.frameHeight !== undefined
      const frameIndex = meta.frame !== undefined ? meta.frame : 0

      // Re-use existing sprite to prevent redraw flickering (PR4)
      if (existingSprites.has(p.id)) {
        const sprite = existingSprites.get(p.id)!
        existingSprites.delete(p.id) // mark as retained
        
        // Sync position (usually unchanged)
        sprite.setPosition(p.x, p.y)
        sprite.setDepth(DECORATION_DEPTH + p.y / 10000.0)
        
        const glow = sprite.getData('glow') as Phaser.GameObjects.Image
        if (glow) {
          glow.setPosition(p.x, p.y - 40)
        }
        
        if (meta.collides && sprite.body) {
          const body = sprite.body as Phaser.Physics.Arcade.StaticBody
          body.updateFromGameObject()
          
          const scale = sprite.scaleX
          const bodyW = (meta.collision_w ?? meta.w ?? (hasFrames ? meta.frameWidth : sprite.width / scale)) * scale
          const bodyH = (meta.collision_h ?? meta.h ?? (hasFrames ? meta.frameHeight : sprite.height / scale)) * scale
          body.setSize(bodyW, bodyH)
          
          const spriteW = (hasFrames ? meta.frameWidth : sprite.width / scale) * scale
          const spriteH = (hasFrames ? meta.frameHeight : sprite.height / scale) * scale

          if (meta.collision_x !== undefined && meta.collision_y !== undefined) {
            body.setOffset(meta.collision_x * scale, meta.collision_y * scale)
          } else {
            const offX = -bodyW / 2 + spriteW * sprite.originX
            const offY = -bodyH + spriteH * sprite.originY
            body.setOffset(offX, offY)
          }
        }
        continue
      }

      // Create new sprite
      let sprite: Phaser.GameObjects.Image
      if (hasFrames) {
        sprite = this.scene.add.image(p.x, p.y, item.asset_key, frameIndex)
      } else {
        sprite = this.scene.add.image(p.x, p.y, item.asset_key)
      }

      // Scale sprite if asset width is smaller than map tile size
      const assetW = hasFrames ? meta.frameWidth : sprite.width
      const scale = assetW < this.tileSize ? this.tileSize / assetW : 1.0
      sprite.setScale(scale)

      sprite.setData('placementId', p.id)
      sprite.setData('itemCode', item.code)
      sprite.setDepth(DECORATION_DEPTH + p.y / 10000.0)

      if (item.code === 'deco_lamppost') {
        const glow = this.scene.add.image(p.x, p.y - 40, 'fx_lantern_glow')
        glow.setDepth(9001) // on top of FX_DEPTH
        glow.setBlendMode(Phaser.BlendModes.ADD)
        glow.setScale(0.35)
        glow.setAlpha(0)
        sprite.setData('glow', glow)
      }

      if (meta.anchorX !== undefined && meta.anchorY !== undefined) {
        sprite.setOrigin(meta.anchorX, meta.anchorY)
      } else {
        sprite.setOrigin(0.5, 1.0)
      }

      if (this.deleteModeActive) {
        sprite.setInteractive()
      }

      sprite.on('pointerover', () => {
        if (this.deleteModeActive) {
          sprite.setTint(0xff5555)
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

      this.placementsGroup.add(sprite)

      // Add static collision body
      if (meta.collides) {
        this.scene.physics.add.existing(sprite, true)
        const body = sprite.body as Phaser.Physics.Arcade.StaticBody
        
        const bodyW = (meta.collision_w ?? meta.w ?? (hasFrames ? meta.frameWidth : sprite.width)) * scale
        const bodyH = (meta.collision_h ?? meta.h ?? (hasFrames ? meta.frameHeight : sprite.height)) * scale
        
        body.updateFromGameObject()
        body.setSize(bodyW, bodyH)
        
        const spriteW = (hasFrames ? meta.frameWidth : sprite.width) * scale
        const spriteH = (hasFrames ? meta.frameHeight : sprite.height) * scale

        if (meta.collision_x !== undefined && meta.collision_y !== undefined) {
          body.setOffset(meta.collision_x * scale, meta.collision_y * scale)
        } else {
          const offX = -bodyW / 2 + spriteW * sprite.originX
          const offY = -bodyH + spriteH * sprite.originY
          body.setOffset(offX, offY)
        }
        this.collisionGroup.add(sprite)
      }

      // Add extra colliders and track their references
      const extraZones: Phaser.GameObjects.Zone[] = []
      if (Array.isArray(meta.extra_colliders)) {
        for (const c of meta.extra_colliders) {
          const zone = this.scene.add.zone(p.x + c.dx, p.y + c.dy, c.w, c.h)
          this.scene.physics.add.existing(zone, true)
          this.collisionGroup.add(zone)
          extraZones.push(zone)
        }
      } else {
        // Fallback bridge logic
        if (item.code.startsWith('deco_bridge_h_')) {
          const z1 = this.scene.add.zone(p.x, p.y - 28, 48, 8)
          this.scene.physics.add.existing(z1, true)
          this.collisionGroup.add(z1)
          extraZones.push(z1)

          const z2 = this.scene.add.zone(p.x, p.y - 4, 48, 8)
          this.scene.physics.add.existing(z2, true)
          this.collisionGroup.add(z2)
          extraZones.push(z2)
        } else if (item.code.startsWith('deco_bridge_v_')) {
          const z1 = this.scene.add.zone(p.x - 20, p.y - 16, 8, 32)
          this.scene.physics.add.existing(z1, true)
          this.collisionGroup.add(z1)
          extraZones.push(z1)

          const z2 = this.scene.add.zone(p.x + 20, p.y - 16, 8, 32)
          this.scene.physics.add.existing(z2, true)
          this.collisionGroup.add(z2)
          extraZones.push(z2)
        }
      }
      sprite.setData('extraZones', extraZones)
    }

    // 3. Remove orphaned sprites and bodies
    existingSprites.forEach((sprite) => {
      const zones = sprite.getData('extraZones') as Phaser.GameObjects.Zone[]
      if (zones) {
        zones.forEach((z) => {
          this.collisionGroup.remove(z, true, true)
          z.destroy()
        })
      }
      const glow = sprite.getData('glow') as Phaser.GameObjects.Image
      if (glow) {
        glow.destroy()
      }
      this.collisionGroup.remove(sprite, true, true)
      this.placementsGroup.remove(sprite, true, true)
    })

    // 4. Update Phaser's spatial physics hash immediately (PR4)
    this.collisionGroup.refresh()
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
      const res = await editorService.deletePlacement(placementId, this.mapCode)

      // Immediately strip physics bodies to update collision grid (PR4)
      const zones = sprite.getData('extraZones') as Phaser.GameObjects.Zone[]
      if (zones) {
        zones.forEach((z) => {
          this.collisionGroup.remove(z, true, true)
          z.destroy()
        })
        sprite.setData('extraZones', null)
      }
      this.collisionGroup.remove(sprite, true, true)
      this.collisionGroup.refresh()

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

      // Snapping to map's tile size grid
      const snappedX = Math.round(worldPoint.x / this.tileSize) * this.tileSize
      const snappedY = Math.round(worldPoint.y / this.tileSize) * this.tileSize

      this.previewSprite.setPosition(snappedX, snappedY)

      // Check if coordinates are occupied
      let occupied = false
      window.dispatchEvent(new CustomEvent('game:checkOccupied', {
        detail: {
          x: snappedX,
          y: snappedY,
          item: this.activeDecorationItem,
          tileSize: this.tileSize,
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

    // Fade placed items when player is behind them (tall items: height > 32)
    const player = this.playerSprite
    const playerBounds = player.getBounds()
    let localBehindDecoration = false

    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      const itemCode = sprite.getData('itemCode') as string
      
      // We only fade trees per user request
      if (itemCode && itemCode.toLowerCase().includes('tree')) {
        const spriteBounds = sprite.getBounds()
        // Only trigger fade if player's Y is behind the bottom base (y - 16) but still within/under the sprite height
        const behind = player.y < sprite.y - 16 && player.y > sprite.y - sprite.height
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

    // Sync lamppost glows with night cycle
    const darkness = getDarkness()
    const time = this.scene.time.now
    const flicker = 0.92 + Math.sin(time * 0.008) * 0.05 + Math.sin(time * 0.021) * 0.03

    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      const glow = sprite.getData('glow') as Phaser.GameObjects.Image
      if (glow) {
        glow.setAlpha(darkness * 0.8 * flicker)
        glow.setVisible(darkness > 0.05)
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

    // Destroy glow images before destroying group
    this.placementsGroup.getChildren().forEach((child) => {
      const sprite = child as Phaser.GameObjects.Image
      const glow = sprite.getData('glow') as Phaser.GameObjects.Image
      if (glow) {
        glow.destroy()
      }
    })

    this.placementsGroup.destroy(true, true)
    this.collisionGroup.destroy(true, true)
    this.clearPreview()
  }
}
