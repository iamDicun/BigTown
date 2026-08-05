import Phaser from 'phaser'
import type { BehaviorHandler } from './types'

export const fadeBehind: BehaviorHandler = {
  onUpdate(sprite, _meta, state) {
    const { player, playerBounds } = state
    const behind =
      player.y < sprite.y - 16 &&
      player.y > sprite.y - sprite.height
    const overlap = Phaser.Geom.Intersects.RectangleToRectangle(
      playerBounds,
      sprite.getBounds(),
    )
    const isBehind = behind && overlap

    if (isBehind) {
      state.isBehindDecoration = true
    }

    const targetAlpha = isBehind ? 0.35 : 1.0
    if (sprite.getData('targetAlpha') !== targetAlpha) {
      sprite.setData('targetAlpha', targetAlpha)
      sprite.scene.tweens.killTweensOf(sprite)
      sprite.scene.tweens.add({
        targets: sprite,
        alpha: targetAlpha,
        duration: 150,
        ease: 'Power1',
      })
    }
  },
}
