import type Phaser from 'phaser'

const NAME_TAG_DEPTH = 5
const NAME_TAG_OFFSET_Y = -20
// Khớp font pixel dùng chung toàn hệ thống (xem --pixel-font trong assets/css/style.css).
// fontSize nhỏ vì camera zoom x2 (GameScene.ts CAMERA_ZOOM) sẽ phóng to gấp đôi kích thước hiển thị
// thật trên màn hình. resolution tăng để canvas render chữ nét hơn khi bị phóng to bởi zoom, tránh
// chữ bị mờ/vỡ nét — stroke mỏng hơn để không bết chữ lại thành 1 khối ở size nhỏ.
const NAME_TAG_STYLE = {
  fontFamily: '"VT323", "Segoe UI", sans-serif',
  fontSize: '8px',
  color: '#ffffff',
  stroke: '#000000',
  strokeThickness: 2,
  resolution: 2,
}

// Tên hiển thị trên đầu nhân vật luôn đọc lại vị trí sprite mỗi frame (updateNameTagPosition) thay
// vì gắn làm con của sprite hoặc tween theo — cách này áp dụng được cho cả local player (di chuyển
// bằng Arcade velocity) lẫn remote player (di chuyển bằng tween nội suy), không cần quan tâm cơ chế
// di chuyển bên dưới là gì, và luôn khớp đúng vị trí render thật của sprite mỗi frame.
export function createNameTag(scene: Phaser.Scene, sprite: Phaser.GameObjects.Sprite, name: string): Phaser.GameObjects.Text {
  const text = scene.add.text(sprite.x, sprite.y + NAME_TAG_OFFSET_Y, name, NAME_TAG_STYLE)
  text.setOrigin(0.5, 1)
  text.setDepth(NAME_TAG_DEPTH)
  return text
}

export function updateNameTagPosition(nameTag: Phaser.GameObjects.Text, sprite: Phaser.GameObjects.Sprite): void {
  nameTag.setPosition(sprite.x, sprite.y + NAME_TAG_OFFSET_Y)
}
