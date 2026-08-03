# BigTown — Placement + Collision đúng chuẩn (Phaser 3)

Hướng dẫn xử lý bài toán: khi **đặt (placement) đồ vật** trong game, đồ vật bị chặn nguyên khối thay vì chỉ chặn ở chân như lúc bạn vẽ collision bằng tay.

---

## 1. Vấn đề & nguyên nhân gốc

Khi **vẽ map bằng tay**, bạn tự tay vẽ khung collision chỉ ở **phần đế/chân nhà** — nên phần trên không chặn, đi vòng ra sau được, che/khuất và mờ đúng ý.

Khi **placement bằng code**, phần lớn trường hợp bạn đang lấy luôn **bounding box của cả sprite** làm collision → nó chặn nguyên cục nhà.

**Chìa khoá:** tách *"khung collision" (footprint)* ra khỏi *"kích thước sprite"*. Placement không nên tự sinh collision từ ảnh, mà nên **tái sử dụng đúng footprint bạn đã định nghĩa cho mỗi loại object** — khai báo một lần, dùng chung cho cả map thủ công lẫn placement.

---

## 2. Ba phần cần làm (map thẳng vào API Phaser)

| Mục tiêu | Kỹ thuật | API Phaser |
|---|---|---|
| Chỉ chặn ở chân nhà | Footprint tách khỏi sprite | `body.setSize(w,h)` + `body.setOffset(x,y)` |
| Đi ra sau được + che/khuất đúng | Y-sort theo chân | `setOrigin(0.5, 1)` + `setDepth(obj.y)` |
| Mờ khi nhân vật bị che | Fade theo overlap | so sánh `y` + `getBounds()` + lerp `alpha` |

### 2.1. Footprint — khung collision = chân nhà

Trong Arcade Physics, **body va chạm độc lập với sprite**. Bạn thu nhỏ body về đúng phần đế:

```js
const fp = def.footprint;          // ví dụ { x: 4, y: 62, w: 56, h: 16 }
obj.body.setSize(fp.w, fp.h);      // body chỉ to bằng chân nhà
obj.body.setOffset(fp.x, fp.y);    // đặt đúng vào đáy sprite
```

> **Lưu ý quan trọng:** `setOffset` luôn tính từ **góc trên‑trái của frame gốc**, *không* phụ thuộc `origin` bạn đặt. Vì vậy cứ dùng thẳng `footprint.x` / `footprint.y` đo trên ảnh.

Chỉ riêng bước này đã cho phép đi vòng ra sau nhà, vì phần thân trên không còn nằm trong tập collision.

### 2.2. Y-sort — đi ra sau & che khuất

Đặt `origin` ở **đáy giữa** cho cả nhà và nhân vật ⇒ toạ độ `y` chính là vị trí **chân**. Mỗi frame gán `depth = y`:

```js
sprite.setOrigin(0.5, 1);
// trong update():
sprite.setDepth(sprite.y);
```

Ai có `y` lớn hơn (thấp hơn trên màn hình) sẽ được vẽ đè lên. Đây là cách hầu hết game top‑down (Stardew, Pokémon…) làm.

### 2.3. Fade — mờ khi nhân vật ở sau

Mỗi frame, nếu nhân vật đang ở **sau** (`player.y < house.y`) **và** hai sprite chồng nhau thì hạ `alpha` của nhà xuống, nội suy cho mượt:

```js
const behind  = player.y < house.y;
const overlap = Phaser.Geom.Intersects.RectangleToRectangle(
  player.getBounds(), house.getBounds()
);
const target = (behind && overlap) ? 0.5 : 1.0;
house.setAlpha(Phaser.Math.Linear(house.alpha, target, 0.18));
```

> Nâng cấp "xịn" hơn sau này: thay vì làm mờ cả nhà, vẽ **silhouette viền nhân vật** đè lên, hoặc đục một vùng mờ quanh nhân vật bằng mask. Bản `alpha 0.5` là đủ để làm trước.

---

## 3. Nguồn dữ liệu duy nhất

Điểm khiến "map vẽ tay đẹp mà placement lỗi" là do chúng dùng **hai nguồn collision khác nhau**. Hãy khai báo footprint + anchor **một lần cho mỗi loại object**, rồi cả hai đọc chung:

```js
const OBJECT_TYPES = {
  house: {
    texture: 'house',
    frameW: 64, frameH: 80,
    footprint: { x: 4, y: 62, w: 56, h: 16 }, // chỉ chân nhà
  },
  // tree: { texture:'tree', frameW:48, frameH:72, footprint:{ x:18, y:60, w:12, h:10 } },
};
```

Muốn bền nhất: làm một **editor nhỏ trong app** để vẽ footprint cho từng loại object một lần, lưu ra JSON, rồi map thủ công và placement cùng đọc file đó → kết quả giống hệt lúc vẽ tay.

---

## 4. File chạy được đầy đủ

Lưu thành `index.html`, mở bằng trình duyệt (cần mạng để tải Phaser CDN).

**Điều khiển:** `WASD`/mũi tên di chuyển · **Click** đặt nhà · **D** bật/tắt debug collision.

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>BigTown · Placement Skeleton</title>
  <style>
    html, body { margin: 0; background: #6a8f5f; overflow: hidden; font-family: sans-serif; }
    #hud { position: absolute; top: 10px; left: 10px; color: #fff; font-size: 13px;
           line-height: 1.6; text-shadow: 0 1px 2px rgba(0,0,0,.6); pointer-events: none; }
    #hud b { color: #ffe08a; }
  </style>
  <script src="https://cdn.jsdelivr.net/npm/phaser@3.80.0/dist/phaser.min.js"></script>
</head>
<body>
  <div id="hud"><b>WASD / mũi tên</b>: di chuyển · <b>Click</b>: đặt nhà · <b>D</b>: debug</div>

<script>
// ===== Nguồn sự thật duy nhất: footprint + kích thước cho mỗi loại object =====
const OBJECT_TYPES = {
  house: {
    texture: 'house',
    frameW: 64, frameH: 80,
    footprint: { x: 4, y: 62, w: 56, h: 16 }, // CHỈ chân nhà, không phải cả sprite
  },
};

const GRID = 16;            // snap placement cho gọn (để 1 nếu không muốn snap)
const PLAYER_SPEED = 170;

class MainScene extends Phaser.Scene {
  constructor() { super('main'); }

  // ----- Tạo texture tạm bằng Graphics (thay bằng this.load.image asset thật) -----
  preload() {
    let g = this.make.graphics({ add: false });
    g.fillStyle(0xc0563a, 1); g.fillTriangle(32, 0, 4, 28, 60, 28);     // mái
    g.fillStyle(0xcba97d, 1); g.fillRect(6, 28, 52, 52);               // thân
    g.fillStyle(0x7a5230, 1); g.fillRect(26, 54, 12, 26);             // cửa
    g.fillStyle(0x8fb8cf, 1); g.fillRect(12, 36, 12, 12); g.fillRect(40, 36, 12, 12); // cửa sổ
    g.generateTexture('house', 64, 80); g.destroy();

    g = this.make.graphics({ add: false });
    g.fillStyle(0x2b4b8f, 1); g.fillRect(5, 12, 14, 20);  // thân
    g.fillStyle(0x4c7be0, 1); g.fillCircle(12, 9, 8);     // đầu
    g.generateTexture('player', 24, 32); g.destroy();
  }

  create() {
    // Nhóm object đã đặt. immovable = không bị đẩy khi va chạm.
    this.obstacles = this.physics.add.group({ immovable: true });

    // Nhân vật: origin (0.5,1) => y là vị trí CHÂN, thuận cho y-sort.
    this.player = this.physics.add.sprite(200, 200, 'player').setOrigin(0.5, 1);
    this.player.body.setSize(16, 10).setOffset(4, 22); // body = chân, không phải cả người
    this.player.setCollideWorldBounds(true);

    this.physics.add.collider(this.player, this.obstacles);

    this.placeObject('house', 360, 180);
    this.placeObject('house', 520, 300);

    this.cursors = this.input.keyboard.createCursorKeys();
    this.wasd = this.input.keyboard.addKeys('W,A,S,D');

    // Placement bằng click (có snap grid).
    this.input.on('pointerdown', (p) => {
      const x = Math.round(p.worldX / GRID) * GRID;
      const y = Math.round(p.worldY / GRID) * GRID;
      this.placeObject('house', x, y);
    });

    // Debug để NHÌN THẤY footprint khác bounding box.
    this.physics.world.createDebugGraphic();
    this.input.keyboard.on('keydown-D', () => {
      this.physics.world.drawDebug = !this.physics.world.drawDebug;
      this.physics.world.debugGraphic.visible = this.physics.world.drawDebug;
    });
  }

  // ----- Trái tim của placement -----
  placeObject(typeKey, x, y) {
    const def = OBJECT_TYPES[typeKey];
    const obj = this.obstacles.create(x, y, def.texture);
    obj.setOrigin(0.5, 1);          // y = chân => y-sort chuẩn
    obj.typeKey = typeKey;

    // *** MẤU CHỐT ***: body = FOOTPRINT, không phải cả sprite.
    const fp = def.footprint;
    obj.body.setSize(fp.w, fp.h);
    obj.body.setOffset(fp.x, fp.y); // offset tính từ góc trên-trái frame gốc
    obj.body.moves = false;         // object đứng yên
    return obj;
  }

  update() {
    // 1) Di chuyển
    let vx = 0, vy = 0;
    if (this.cursors.left.isDown  || this.wasd.A.isDown) vx -= 1;
    if (this.cursors.right.isDown || this.wasd.D.isDown) vx += 1;
    if (this.cursors.up.isDown    || this.wasd.W.isDown) vy -= 1;
    if (this.cursors.down.isDown  || this.wasd.S.isDown) vy += 1;
    const len = Math.hypot(vx, vy) || 1;
    this.player.body.setVelocity((vx / len) * PLAYER_SPEED, (vy / len) * PLAYER_SPEED);

    // 2) Y-SORT: depth = y của chân
    this.player.setDepth(this.player.y);
    this.obstacles.children.iterate((obj) => {
      obj.setDepth(obj.y);

      // 3) FADE: player ở sau (y nhỏ hơn) và hai sprite chồng nhau
      const behind  = this.player.y < obj.y;
      const overlap = Phaser.Geom.Intersects.RectangleToRectangle(
        this.player.getBounds(), obj.getBounds()
      );
      const target = (behind && overlap) ? 0.5 : 1.0;
      obj.setAlpha(Phaser.Math.Linear(obj.alpha, target, 0.18));
    });
  }
}

new Phaser.Game({
  type: Phaser.AUTO,
  width: 800, height: 600,
  backgroundColor: '#6a8f5f',
  physics: { default: 'arcade', arcade: { debug: false } }, // bật debug bằng phím D
  scene: [MainScene],
});
</script>
</body>
</html>
```

---

## 5. Port vào project thật — checklist

- [ ] Thay `generateTexture(...)` bằng `this.load.image('house', 'assets/house.png')`.
- [ ] Đo lại `footprint` (px) trên ảnh thật, điền vào `OBJECT_TYPES`.
- [ ] Đảm bảo **cả** map thủ công **và** placement đọc chung `OBJECT_TYPES`.
- [ ] Nếu dùng **Tilemap + pathfinding** (NPC tự đi): khi đặt object, ngoài tạo body footprint, đánh dấu các tile bị footprint phủ là `blocked` để NPC cũng né.
- [ ] Bật phím `D` để kiểm tra bằng mắt: khung collision phải nằm gọn ở chân, khác hẳn kích thước sprite.

---

## 6. Bẫy thường gặp

- **Quên `setOrigin(0.5, 1)`** → `depth = y` sort sai, che/khuất lộn xộn.
- **Dùng `setSize` với `center = true`** → body tự canh giữa, lệch khỏi chân. Cứ để mặc định rồi `setOffset` tay.
- **Fade giật/nháy** → luôn `Phaser.Math.Linear(...)` thay vì gán thẳng `alpha`.
- **Object bị player đẩy trôi** → đặt `immovable: true` cho group và `body.moves = false`.
