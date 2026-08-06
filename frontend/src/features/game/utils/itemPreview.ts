import type { DecorationItemDto } from '../services/editor.service'

export function getItemPreviewStyle(item: DecorationItemDto, maxSize = 64): Record<string, any> {
  let meta: any = {}
  try {
    meta = JSON.parse(item.metadata_json)
  } catch {}

  if (meta.frameWidth !== undefined && meta.frameHeight !== undefined && meta.frame !== undefined) {
    let sheetW = 0
    let sheetH = 0

    const key = item.asset_key.toLowerCase()
    if (key.includes('fences.png')) {
      sheetW = 64
      sheetH = 64
    } else if (key.includes('oak_tree_small.png')) {
      sheetW = 96
      sheetH = 48
    } else if (key.includes('bridge_wood.png')) {
      sheetW = 144
      sheetH = 64
    } else if (key.includes('outdoor_decor_free.png')) {
      sheetW = 112
      sheetH = 192
    } else if (key.includes('water_tile.png')) {
      sheetW = 48
      sheetH = 96
    } else if (key.includes('cliff_tile.png')) {
      sheetW = 48
      sheetH = 96
    } else if (key.includes('path_tile.png')) {
      sheetW = 48
      sheetH = 96
    } else if (key.includes('farmland_tile.png')) {
      sheetW = 48
      sheetH = 48
    } else if (key.includes('beach_tile.png')) {
      sheetW = 80
      sheetH = 48
    } else if (key.includes('path_middle.png')) {
      sheetW = 16
      sheetH = 16
    } else if (key.includes('grass_middle.png')) {
      sheetW = 16
      sheetH = 16
    }

    if (sheetW > 0 && sheetH > 0) {
      const cols = sheetW / meta.frameWidth
      const col = meta.frame % cols
      const row = Math.floor(meta.frame / cols)

      const posX = -col * meta.frameWidth
      const posY = -row * meta.frameHeight

      const maxDim = Math.max(meta.frameWidth, meta.frameHeight)
      const scale = maxDim > 0 ? maxSize / maxDim : 1

      return {
        width: `${meta.frameWidth}px`,
        height: `${meta.frameHeight}px`,
        backgroundImage: `url(/assets/${item.asset_key})`,
        backgroundPosition: `${posX}px ${posY}px`,
        backgroundSize: `${sheetW}px ${sheetH}px`,
        imageRendering: 'pixelated',
        transform: `scale(${scale})`,
        transformOrigin: 'center'
      }
    }
  }

  return {
    width: '100%',
    height: '100%',
    backgroundImage: `url(/assets/${item.asset_key})`,
    backgroundPosition: 'center',
    backgroundSize: 'contain',
    backgroundRepeat: 'no-repeat',
    imageRendering: 'pixelated'
  }
}
