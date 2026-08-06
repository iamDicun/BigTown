import Phaser from 'phaser'
import type { BehaviorHandler } from './types'

const GLOW_TEXTURE = 'fx_lantern_glow'

export const glowNight: BehaviorHandler = {
  onCreate(sprite, _meta, ctx) {
    const glow = ctx.scene.add.image(sprite.x, sprite.y - 40, GLOW_TEXTURE)
    glow.setDepth(9001)
    glow.setBlendMode(Phaser.BlendModes.ADD)
    glow.setScale(0.35)
    glow.setAlpha(0)
    sprite.setData('glow', glow)
  },

  onUpdate(sprite, _meta, state) {
    const glow = sprite.getData('glow') as Phaser.GameObjects.Image | undefined
    if (!glow) return

    glow.setPosition(sprite.x, sprite.y - 40)
    glow.setAlpha(state.darkness * 0.8 * state.flicker)
    glow.setVisible(state.darkness > 0.05)
  },

  onDestroy(sprite) {
    const glow = sprite.getData('glow') as Phaser.GameObjects.Image | undefined
    if (glow) {
      glow.destroy()
      sprite.setData('glow', null)
    }
  },
}
