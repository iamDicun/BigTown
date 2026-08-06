# BigTown — Design System "Cute Fantasy"

Mục tiêu: đồng bộ toàn bộ UI (login, hotbar, inventory, chat, editor, menu…) về **một ngôn ngữ hình ảnh** lấy cảm hứng từ bộ *Cute Fantasy* của Kenmi — pixel art 16-bit, khung gỗ – giấy da ấm, nút bấm lún, ô đồ dạng hõm gỗ. Điểm mạnh: hệ token hiện tại của bạn (VT323 + tông gỗ/giấy da + `PixelPanel` khung 9-slice) **đã đi đúng hướng**; tài liệu này chuẩn hoá nó và bổ sung vài "đặc trưng game" còn thiếu.

Kèm theo: `pixel-ui.css` — token + class dùng được ngay.

---

## 1. Hướng thẩm mỹ (1 câu)

> Một thị trấn cổ tích ấm cúng: gỗ vàng nâu, giấy da kem, ánh xu vàng, các khung bo góc nhẹ và mọi nút đều "bấm lún" như phím gỗ thật — đủ dễ thương nhưng vẫn chắc như đồ thủ công.

**3 quy tắc bất biến để mọi màn hình trông "cùng một game":**
1. **Khung gỗ – ruột giấy da.** Mọi panel = nền `--pixel-parchment` + viền gỗ nhiều lớp (`--frame-wood`). Không panel nào phẳng/không viền.
2. **Biển hiệu ribbon làm tiêu đề.** Mọi panel/modal có tên đều treo một `.ui-banner` ở mép trên — đây là **chi tiết nhận diện** xuyên suốt.
3. **Chiều sâu bằng bevel pixel cứng, không blur.** Nút/ô/badge dùng `--bevel-out`/`--bevel-in` + cạnh dưới `--lift`. Bóng mềm (`--drop`) **chỉ** dành cho modal lớn.

---

## 2. Token màu

| Vai trò | Biến | Hex |
|---|---|---|
| Viền/contour pixel | `--pixel-outline` | `#2a1e12` |
| Chữ chính | `--pixel-ink` | `#3a2b1a` |
| Gỗ (tối/vừa/sáng) | `--pixel-wood-dark/‑wood/‑wood-light` | `#6b4226` · `#8a5a34` · `#c98a4b` |
| Giấy da (nền panel) | `--pixel-parchment` / `-dark` | `#fdf1d6` · `#f0dfae` |
| Hõm ô đồ | `--pixel-well` / `-dark` | `#e7d2a6` · `#cdb078` |
| Hành động chính / đang chọn | `--pixel-accent` / `-dark` | `#ef8b3d` · `#c96a24` |
| Xác nhận | `--pixel-green` / `-dark` | `#5a9c4a` · `#3c6b34` |
| Xoá / cảnh báo | `--pixel-danger` / `-dark` | `#c94a3c` · `#9e3227` |
| Xu vàng | `--pixel-gold` / `-dark` | `#ffcf5c` · `#e0a022` |
| Kẹo phụ (dùng ít) | `--pixel-berry`, `--pixel-sky` | `#d5628f` · `#5b8fd6` |

**Rủi ro thẩm mỹ có chủ đích (điểm nhấn cute):** giữ nền gỗ/giấy da làm chủ đạo, chỉ thả **màu "kẹo"** (berry/sky/gold) ở chi tiết nhỏ — huy hiệu xu, tab đang chọn, thanh xp — để có nét dễ thương mà không phá sự ấm cúng. Đừng tô berry/sky lên mảng lớn.

---

## 3. Chữ

- **Body:** `--pixel-font` = VT323 (giữ nguyên). Dùng cho nội dung, nhãn, số.
- **Display (heading/banner):** `--pixel-font-display`. Mặc định = VT323 để không phải thêm font. **Nâng cấp (khuyến nghị):** để heading "mập" và cong hơn cho hợp cute-fantasy, đổi sang một pixel font bo tròn:
  - Nếu bạn **đã mua** gói *Cute Fantasy UI* của Kenmi (có kèm .ttf) → nhúng font đó bằng `@font-face`, sẽ khớp 100% với asset.
  - Miễn phí thay thế: `Pixelify Sans` hoặc `Jersey 15` (Google Fonts). Chỉ cần đổi 1 dòng:
    ```css
    :root { --pixel-font-display: "Pixelify Sans", var(--pixel-font); }
    ```
- **Thang cỡ chữ:** hero 34 · title 26 · head 22 · body 19 · label 16 · caption 14 (`--fs-*`). Letter-spacing `0.5px` cho chữ pixel.

---

## 4. Nhịp & khối

- **Lưới 4px:** mọi padding/gap lấy từ `--sp-1..6` (4/8/12/16/24/32). Pixel art "ăn" lưới chẵn.
- **Bo góc nhỏ:** `--r` = 3px (khung pixel chỉ bo rất nhẹ). Ngoại lệ tròn hẳn: badge xu, thanh bar (999px).
- **Viền:** `--bw` = 3px, màu `--pixel-outline`. Đồng bộ mọi thành phần dùng đúng độ dày này để nhìn "cùng bộ".

---

## 5. Bộ component (xem `pixel-ui.css`)

| Class | Dùng cho |
|---|---|
| `.ui-panel`, `.ui-panel--book` | Mọi khung nền. `--book` cho menu/inventory lớn (có gáy sách). |
| `.ui-banner` | **Tiêu đề ribbon** treo mép trên panel — dùng ở mọi modal. |
| `.ui-btn` (+ `--confirm/--danger/--ghost/--icon/--sm`) | Mọi nút. Tự có hover sáng + bấm lún + focus ring. |
| `.ui-slot` (+ `.is-active/.is-locked`) + `.ui-slot-key/-icon` | Ô hotbar / ô inventory. |
| `.ui-badge--coin` | Hiển thị xu. |
| `.ui-tab` (+ `.is-active`) | Tab danh mục inventory. |
| `.ui-input` | Ô search, ô chat, form login. |
| `.ui-tooltip`, `.ui-toast` (+ `--ok`) | Nhắc & thông báo. |
| `.ui-bar` (+ `--hp/--xp`) | Thanh máu/xp/stamina. |
| `.ui-overlay`, `.ui-pop` | Nền mờ + hiệu ứng bung modal. |
| `.ui-scroll` | Thanh cuộn kiểu gỗ. |

Nguyên tắc: **component mới chỉ ráp từ các class này**, không tự đặt màu/hex rời rạc. Muốn đổi tông cả game → sửa token ở `:root`, không sửa từng file.

---

## 6. Áp dụng vào các màn hình hiện có

Đổi dần, mỗi PR một cụm. Bảng ánh xạ:

- **Hotbar.vue** → khung dùng `.ui-panel` thu nhỏ; mỗi ô đổi sang `.ui-slot`/`.ui-slot-key`/`.ui-slot-icon`; nút 🗑️/🖌️ dùng `.ui-btn--icon`. (Bạn đang tự viết bevel bằng tay — thay bằng class để đồng bộ.)
- **InventoryModal.vue** → `.ui-overlay > .ui-panel.ui-panel--book.ui-pop`; tiêu đề "KHO ĐỒ VẬT" thành `.ui-banner`; tab dùng `.ui-tab`; ô search `.ui-input`; lưới item dùng `.ui-slot`; vùng cuộn thêm `.ui-scroll`.
- **EditorPanel.vue** → cụm coins đổi sang `.ui-badge--coin`; `error-toast` đổi sang `.ui-toast`.
- **ChatPanel.vue** → khung `.ui-panel`, ô nhập `.ui-input`.
- **AudioSettingsPanel.vue** → `.ui-panel` + `.ui-banner` "ÂM THANH" + slider bọc `.ui-bar` style.
- **AuthCard.vue / LoginView / Navbar** → `PixelPanel` giữ nguyên (nó chính là `.ui-panel`); nút login đổi sang `.ui-btn`, nút Teams `.ui-btn` nền `--pixel-sky`; thêm `.ui-banner` "BIGTOWN".
- **LoadingSplash.vue** → thanh tiến trình đổi sang `.ui-bar--xp`.

Gợi ý gộp `PixelPanel.vue` và `.ui-panel` làm một (PixelPanel chỉ cần thêm slot `title` render `.ui-banner`), để không có hai định nghĩa khung song song.

---

## 7. "Trông như game thật" — gắn asset 9-slice thật

CSS ở trên đã cho cảm giác pixel-fantasy, nhưng bước nhảy lớn nhất về độ "thật" là **dùng chính khung PNG của gói Cute Fantasy UI** (nếu bạn đã mua — license của Kenmi cho phép dùng trong game thương mại, **không** cho phát tán lại file gốc). Cách nhúng khung 9-slice bằng `border-image`:

```css
.ui-panel--asset {
  /* frame_XX.png là khung viền của gói UI, ví dụ ô 16x16, viền dày 6px */
  border-image: url("/assets/ui/frame_wood.png") 6 6 6 6 fill repeat;
  border-width: 18px;                 /* = 6px * 3 (scale x3 cho nét pixel to) */
  border-style: solid;
  image-rendering: pixelated;
  background: none;                   /* để 'fill' của border-image lo phần ruột */
  box-shadow: var(--drop);
}
```

- `border-image-slice: 6 6 6 6 fill` = cắt 6px mỗi cạnh làm góc, `fill` giữ phần giữa để lát ruột.
- `border-width` nên là bội số nguyên của slice để pixel không méo (6→12/18/24).
- Áp cùng công thức cho **nút** (dùng frame nút của gói, 3 trạng thái normal/hover/pressed) và **ô item** (dùng tile slot). Khi đó CSS token vẫn giữ nguyên cho màu chữ/xu/hiệu ứng, chỉ phần "khung" chuyển sang ảnh thật.

Lộ trình an toàn: **giữ cả hai** — class `.ui-panel` (thuần CSS, luôn chạy) làm nền, và biến thể `.ui-panel--asset` chỉ bật khi đã bỏ file khung vào `/public/assets/ui/`. Chưa có asset thì UI vẫn đẹp; có asset thì lên "chuẩn Kenmi".

> Lưu ý bản quyền: chỉ dùng file bạn đã mua/được cấp phép; đừng commit asset trả phí lên repo public. Để chúng ở nơi private hoặc .gitignore nếu repo mở.

---

## 8. Chuyển động (tiết chế)

- Nút/ô: bấm lún `translateY` (đã có trong class). Hover sáng nhẹ.
- Modal: `.ui-pop` bung 0.16s. Ô đang chọn: viền cam + glow nhẹ.
- Xu tăng: cân nhắc số nảy nhẹ (scale 1→1.15→1). Đừng hơn — nhiều animation làm rối và mất chất "thủ công".
- Luôn tôn trọng `prefers-reduced-motion` (đã xử lý trong CSS).

---

## 9. Do / Don't

**Do**
- Mọi màu lấy từ token; mọi khung/nút/ô lấy từ class.
- Viền `3px` `--pixel-outline` đồng nhất; padding theo lưới 4px.
- Một điểm nhấn nổi bật mỗi màn (thường là ribbon tiêu đề hoặc nút chính), còn lại giữ trầm.

**Don't**
- Không hex rời rạc trong component; không trộn bo góc lớn (12px+) với khung pixel.
- Không blur bóng ở nút/ô nhỏ (mất chất pixel) — chỉ blur ở `--drop` của modal.
- Không phủ berry/sky lên mảng lớn; chúng là "kẹo" điểm xuyết.
- Không để hai hệ khung song song (PixelPanel cũ vs class mới) — hợp nhất.

---

## 10. Việc nên làm theo thứ tự

1. Import `pixel-ui.css` vào `style.css`. Kiểm tra token không phá layout hiện tại (tên `--pixel-*` cũ giữ nguyên).
2. (Tuỳ chọn) Đổi `--pixel-font-display` sang Pixelify Sans / font Kenmi.
3. Refactor **Hotbar** + **InventoryModal** sang class chung (đang là nơi bạn thấy "chưa thật") — thêm `.ui-banner`, `.ui-slot`, `.ui-btn`.
4. Lan sang Chat / Editor coins / Audio / Login.
5. Nếu có gói UI Kenmi: thêm `/public/assets/ui/` + bật biến thể `--asset` cho panel/nút/slot.
