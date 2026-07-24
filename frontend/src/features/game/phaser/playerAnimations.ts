import type { Direction } from '../network/gameEvents'
import type { SpritesheetConfigDto } from '../services/character.service'

// Animation key suffix constants — các key animation thực tế sẽ là "{textureKey}-idle-down", v.v.
// Dùng chung suffix này cho các hàm lookup animation ở local/remote player.
const SUFFIX_IDLE_DOWN = '-idle-down'
const SUFFIX_WALK_DOWN = '-walk-down'
const SUFFIX_IDLE_UP = '-idle-up'
const SUFFIX_WALK_UP = '-walk-up'
const SUFFIX_IDLE_SIDE = '-idle-side'
const SUFFIX_WALK_SIDE = '-walk-side'

// createAnimations tạo animation Phaser từ config spritesheet và textureKey.
// Dùng chung cho mọi loại nhân vật — mỗi loại chỉ khác config (số cột, thứ tự hàng, frame rate).
export function createAnimations(scene: Phaser.Scene, textureKey: string, config: SpritesheetConfigDto): void {
  const rowFrames = (row: number) =>
    Array.from({ length: config.columns }, (_, i) => row * config.columns + i)

  const frames = (row: number) => scene.anims.generateFrameNumbers(textureKey, { frames: rowFrames(row) })

  const safeCreate = (key: string, animConfig: any) => {
    if (!scene.anims.exists(key)) {
      scene.anims.create({ key, ...animConfig })
    }
  }

  safeCreate(textureKey + SUFFIX_IDLE_DOWN, {
    frames: frames(config.row_idle_down),
    frameRate: config.idle_frame_rate,
    repeat: -1,
  })
  safeCreate(textureKey + SUFFIX_WALK_DOWN, {
    frames: frames(config.row_walk_down),
    frameRate: config.walk_frame_rate,
    repeat: -1,
  })
  safeCreate(textureKey + SUFFIX_IDLE_UP, {
    frames: frames(config.row_idle_up),
    frameRate: config.idle_frame_rate,
    repeat: -1,
  })
  safeCreate(textureKey + SUFFIX_WALK_UP, {
    frames: frames(config.row_walk_up),
    frameRate: config.walk_frame_rate,
    repeat: -1,
  })
  safeCreate(textureKey + SUFFIX_WALK_SIDE, {
    frames: frames(config.row_walk_side),
    frameRate: config.walk_frame_rate,
    repeat: -1,
  })
  safeCreate(textureKey + SUFFIX_IDLE_SIDE, {
    frames: [{ key: textureKey, frame: rowFrames(config.row_walk_side)[0] }],
  })
}

export type Facing = 'down' | 'up' | 'side'

export function facingForDirection(direction: Direction): Facing {
  if (direction === 'up') return 'up'
  if (direction === 'left' || direction === 'right') return 'side'
  return 'down'
}

// Các hàm dưới dùng textureKey + suffix để chọn đúng animation key khi character có texture riêng.
// LocalPlayerController và RemotePlayerManager sẽ gọi với textureKey tương ứng của character đó.

export function walkAnimKey(textureKey: string, facing: Facing): string {
  if (facing === 'up') return textureKey + SUFFIX_WALK_UP
  if (facing === 'side') return textureKey + SUFFIX_WALK_SIDE
  return textureKey + SUFFIX_WALK_DOWN
}

export function idleAnimKey(textureKey: string, facing: Facing): string {
  if (facing === 'up') return textureKey + SUFFIX_IDLE_UP
  if (facing === 'side') return textureKey + SUFFIX_IDLE_SIDE
  return textureKey + SUFFIX_IDLE_DOWN
}

// defaultConfig trả về config mặc định (dùng cho fallback khi store chưa có).
// Luôn giữ đồng bộ với backend SpritesheetConfig của "Nhà thám hiểm".
export function defaultConfig(): SpritesheetConfigDto {
  return {
    frame_width: 32,
    frame_height: 32,
    columns: 6,
    row_idle_down: 0,
    row_walk_down: 1,
    row_idle_up: 2,
    row_walk_up: 3,
    row_walk_side: 4,
    walk_frame_rate: 8,
    idle_frame_rate: 4,
  }
}
