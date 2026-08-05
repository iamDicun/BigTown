# Mục 1 — Tối ưu "lần đầu vào game load lâu"

## Nguyên nhân (đã đo)

| Tài nguyên | Kích thước | Vấn đề |
|---|---|---|
| `public/assets/sounds/intro.mp3` | **15 MB** | Autoplay ngay khi trang login mount (`AuthLayout.vue:7`), kéo băng thông đúng lúc first paint |
| `sounds/dark_village.mp3` | 8 MB | Nhạc map, tải khi vào map |
| `sounds/winter.mp3` | 6 MB | Nhạc map |
| `sounds/village_adventure.mp3` | 4.7 MB | Nhạc map |
| `src/assets/images/logo.png` | 194 KB | Ảnh logo chưa nén |

Thư mục `sounds/` tổng ~40 MB. `intro.mp3` là thứ nặng nhất và tệ nhất vì nó nổ **ngay lúc mở app**, tranh kết nối với bundle JS + font + logo.

Xử lý theo 3 việc: (A) đừng để nhạc chặn first paint, (B) nén toàn bộ mp3, (C) nén logo.

---

## A. Đừng để `intro.mp3` chặn first paint

Đang bật autoplay full track lúc mount. Đổi thành: chỉ nạp nhạc **sau khi app đã render + người dùng có tương tác đầu tiên** (trình duyệt cũng chặn autoplay trước tương tác, nên tận dụng luôn).

File: `src/app/layouts/AuthLayout.vue`

**Trước:**
```ts
import { playMusic, stopMusic } from '@/shared/audio/audio.service'

onMounted(() => {
  playMusic('/assets/sounds/intro.mp3', { fadeMs: 1600, volume: 0.25 })
})
```

**Sau** — hoãn tới sau khi trình duyệt rảnh, và để `bindUnlockListener` sẵn có trong audio.service tự phát khi người dùng chạm/gõ phím lần đầu:
```ts
import { playMusic, stopMusic } from '@/shared/audio/audio.service'

onMounted(() => {
  // requestIdleCallback: chờ main thread rảnh (sau first paint) mới bắt đầu kéo nhạc.
  const start = () => playMusic('/assets/sounds/intro.mp3', { fadeMs: 1600, volume: 0.25 })
  if ('requestIdleCallback' in window) {
    ;(window as any).requestIdleCallback(start, { timeout: 2000 })
  } else {
    setTimeout(start, 1200)
  }
})
```

`audio.service.ts` đã có sẵn `bindUnlockListener()` nên nếu autoplay bị chặn, nhạc vẫn tự phát ở lần chạm đầu — không cần sửa thêm.

> Nếu muốn triệt để: giữ một đoạn intro ngắn ~20–30s loop (xem phần nén bên dưới cách `-t` cắt) thay vì cả bài dài. File loop 20s ở 96 kbps chỉ ~240 KB.

---

## B. Nén MP3 bằng ffmpeg (phần chính)

### B1. Cài ffmpeg

- **Windows:** tải bản build tại https://www.gyan.dev/ffmpeg/builds/ (gói "release full"), giải nén, thêm thư mục `bin` vào PATH. Kiểm tra: `ffmpeg -version`.
- **macOS:** `brew install ffmpeg`
- **Ubuntu/Debian:** `sudo apt install ffmpeg`

### B2. Xem thông số file gốc trước khi nén

```bash
ffprobe -hide_banner public/assets/sounds/intro.mp3
```
Chú ý 3 dòng: `Duration` (độ dài), `bitrate` (vd 320 kb/s → đây là lý do file to), `Stereo/Mono`. Nhạc 15 MB thường là bài dài + 320 kbps stereo.

### B3. Bảng khuyến nghị bitrate

| Loại | Kênh | Bitrate | Ghi chú |
|---|---|---|---|
| Nhạc nền map / intro | stereo | **96–128 kbps** | Tai người khó phân biệt với 320k khi làm nền game |
| Nhạc nền (muốn nhỏ hơn nữa) | mono | 80–96 kbps | Game top-down không cần stereo rộng |
| SFX ngắn (bước chân, click) | mono | 64–96 kbps | File đã nhỏ sẵn (20–36 KB), thường không cần đụng |

### B4. Lệnh nén nhạc nền (giảm bitrate, giữ stereo)

```bash
ffmpeg -i intro.mp3 -c:a libmp3lame -b:a 112k -ar 44100 intro-min.mp3
```
- `-b:a 112k`: bitrate đích. 320k → 112k là giảm ~65% dung lượng.
- `-ar 44100`: sample rate chuẩn nhạc.

### B5. Nhỏ hơn nữa — đổi sang mono + chuẩn hoá âm lượng

```bash
ffmpeg -i intro.mp3 -c:a libmp3lame -b:a 96k -ac 1 -af loudnorm=I=-16:TP=-1.5:LRA=11 intro-min.mp3
```
- `-ac 1`: trộn xuống mono (thêm ~30–40% giảm dung lượng).
- `-af loudnorm`: chuẩn hoá loudness để các bài nhạc/SFX không chênh to-nhỏ (I=-16 LUFS là mức phổ biến cho web/game).

### B6. Cắt ngắn + tạo đoạn loop (khuyến nghị cho intro)

```bash
# Lấy 24 giây kể từ giây thứ 8, fade in/out để loop mượt
ffmpeg -ss 8 -t 24 -i intro.mp3 -c:a libmp3lame -b:a 96k -ac 2 \
  -af "afade=t=in:st=0:d=1.5,afade=t=out:st=22.5:d=1.5" intro-loop.mp3
```
- `-ss 8` bắt đầu từ giây 8, `-t 24` lấy dài 24s.
- Kết quả ~280 KB thay vì 15 MB.

### B7. Nén VBR (chất lượng ổn định, dung lượng tối ưu)

```bash
ffmpeg -i winter.mp3 -c:a libmp3lame -q:a 5 winter-min.mp3
```
`-q:a` từ 0 (đẹp/nặng) đến 9 (nhỏ/tệ). Mức 5 ≈ ~130 kbps trung bình, cân bằng tốt.

### B8. Nén hàng loạt cả thư mục (Linux/macOS / Git Bash trên Windows)

```bash
cd public/assets/sounds
mkdir -p min
for f in *.mp3; do
  ffmpeg -y -i "$f" -c:a libmp3lame -b:a 112k -ar 44100 "min/$f"
done
# nghe thử trong thư mục min/, ưng thì thay thế file gốc
```

### B9. (Tuỳ chọn) Bản OGG/Opus nhỏ hơn cho trình duyệt hiện đại

Opus cho chất lượng tương đương ở dung lượng thấp hơn mp3 ~30%. Trình duyệt Chrome/Firefox/Edge/Safari mới đều hỗ trợ.
```bash
ffmpeg -i intro.mp3 -c:a libopus -b:a 80k intro.ogg
```
Nếu dùng, Phaser có thể nạp cả 2 định dạng để trình duyệt tự chọn:
```ts
this.load.audio('bg_music', ['/assets/sounds/intro.ogg', '/assets/sounds/intro.mp3'])
```

### B10. Bảng dung lượng mục tiêu sau nén

| File | Hiện tại | Mục tiêu |
|---|---|---|
| intro.mp3 (hoặc loop) | 15 MB | < 0.5 MB |
| dark_village.mp3 | 8 MB | ~2 MB |
| winter.mp3 | 6 MB | ~1.5 MB |
| village_adventure.mp3 | 4.7 MB | ~1.2 MB |

> Không có ffmpeg? Dùng Audacity (Tracks → Resample + Export as MP3, chọn bitrate) hoặc web như https://www.freeconvert.com/audio-compressor. ffmpeg vẫn nhanh và lặp lại được nên khuyến nghị hơn.

---

## C. Nén logo.png (194 KB)

```bash
# pngquant: giảm màu, thường giảm 60–70% mà mắt không phân biệt
npx pngquant --quality=65-85 --output logo-min.png src/assets/images/logo.png

# hoặc chuyển WebP (nhỏ hơn nữa, mọi trình duyệt hiện đại đều hỗ trợ)
ffmpeg -i src/assets/images/logo.png -c:v libwebp -quality 80 logo.webp
```

---

## D. Cách kiểm chứng đã cải thiện

1. DevTools → tab **Network** → biểu tượng throttle chọn **Fast 3G** → tick **Disable cache** → reload trang login.
2. Nhìn cột **Waterfall**: trước khi sửa `intro.mp3` chiếm một thanh dài chạy song song lúc đầu; sau khi hoãn + nén, nó biến mất khỏi đợt đầu.
3. Xem **DOMContentLoaded** và **Load** ở chân tab Network — so sánh số ms trước/sau.
4. Chạy **Lighthouse** (tab Lighthouse → Analyze) và so **First Contentful Paint** / **Total Byte Weight** trước và sau.
