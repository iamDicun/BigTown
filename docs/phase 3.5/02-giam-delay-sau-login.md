# Mục 2 — Giảm delay "bấm login xong lâu mới tới bước tải tài nguyên"

## Nguyên nhân (đã đo bằng `vite build`)

```
dist/assets/GameView-*.js   1,511 kB  (gzip 394 kB)   ← chứa TOÀN BỘ Phaser
dist/assets/index-*.js         56 kB
dist/assets/LoginView-*.js    6.6 kB
```

Route `/game` được lazy-load (`src/features/game/routes.ts`), và Phaser chỉ được `import` bên trong chunk đó. Nên **1.5 MB Phaser chỉ tải + parse SAU khi bạn bấm login và điều hướng sang `/game`**. Trên mobile, riêng việc parse/compile 1.5 MB JS đã mất 1–3 giây → đó là khoảng "đơ" trước khi `PreloadScene` hiện thanh tiến trình.

Ba việc, theo thứ tự tác động: (A) prefetch chunk game ngay khi ở trang login, (B) tách Phaser thành vendor chunk riêng, (C) hiện loading sớm + fetch bootstrap sớm.

---

## A. Prefetch chunk game trong lúc người dùng còn ở trang login

Ý tưởng: trong lúc người dùng đang gõ email/mật khẩu, tải nền sẵn chunk game. Bấm login xong là nó đã nằm trong cache → vào gần như tức thì.

File: `src/features/auth/views/LoginView.vue` — thêm vào `onMounted` sẵn có:

```ts
onMounted(async () => {
  const inside = await initTeams()
  inTeams.value = inside
  if (inside && !skipAutoLogin) {
    handleTeamsLogin()
  } else {
    teamsConnecting.value = false
  }

  // ---- THÊM: warm-up chunk game khi main thread rảnh ----
  const warmGameChunk = () => {
    // Cùng đường dẫn động với game/routes.ts → Vite tái dùng đúng chunk, không tải trùng.
    void import('@/features/game/views/GameView.vue')
  }
  if ('requestIdleCallback' in window) {
    ;(window as any).requestIdleCallback(warmGameChunk, { timeout: 3000 })
  } else {
    setTimeout(warmGameChunk, 800)
  }
})
```

> Vì đây đúng là module Vue Router sẽ lazy-load, trình duyệt tải một lần và dùng lại. Không phá vỡ code-splitting, chỉ **dời thời điểm tải sớm lên**.

Bổ sung nhẹ (tuỳ chọn) — gợi ý trình duyệt ưu tiên: thêm link modulepreload động khi vào trang login. Nhưng cách `import()` ở trên đã đủ tốt và đơn giản hơn; ưu tiên dùng nó trước.

---

## B. Tách Phaser thành vendor chunk riêng (Vite 8 / Rolldown)

Lợi ích: Phaser hiếm khi đổi, tách riêng để **cache độc lập** — mỗi lần bạn sửa code game, người dùng chỉ tải lại phần code nhỏ của bạn, không tải lại 1.5 MB Phaser.

File: `frontend/vite.config.ts`

```ts
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    rollupOptions: {
      output: {
        // Rolldown (Vite 8) — advancedChunks là API chia chunk nâng cao.
        advancedChunks: {
          groups: [
            { name: 'phaser', test: /[\\/]node_modules[\\/]phaser[\\/]/, priority: 10 },
            { name: 'vendor', test: /[\\/]node_modules[\\/]/, priority: 1 },
          ],
        },
      },
    },
  },
})
```

Nếu bản Rolldown của bạn báo không nhận `advancedChunks`, dùng cách hàm `manualChunks` tương thích:
```ts
build: {
  rollupOptions: {
    output: {
      manualChunks(id) {
        if (id.includes('node_modules/phaser')) return 'phaser'
        if (id.includes('node_modules')) return 'vendor'
      },
    },
  },
},
```

Sau khi sửa, chạy lại và kiểm tra đã có chunk `phaser-*.js` tách riêng:
```bash
npx vite build
```
Bạn sẽ thấy `phaser-*.js` ~1.3 MB đứng riêng, còn `GameView-*.js` tụt xuống chỉ còn code game của bạn (vài chục KB).

> Lưu ý: tách chunk **không** giảm tổng byte lần đầu, nhưng kết hợp với mục A (prefetch) thì chunk phaser tải nền sớm; và về sau tận dụng cache tốt hơn nhiều.

---

## C. Hiện loading sớm + fetch bootstrap sớm

Hiện tại `GameCanvas.vue` chỉ gọi `getBootstrap()` + `loadMyCharacter()` **sau khi component mount xong** (tức sau khi chunk 1.5 MB đã tải + parse). Có thể chồng lấn thời gian bằng cách bắn request bootstrap sớm hơn, song song với việc tải chunk.

### C1. Đảm bảo LoadingSplash hiện ngay khi điều hướng

Trong `GameCanvas.vue`, `loading` khởi tạo `= true` (đã đúng). Nhưng splash chỉ render khi GameView chunk đã tải. Để có phản hồi tức thì lúc bấm login, thêm một splash cấp route: hiển thị overlay ngay trong `LoginView.handleSubmit` trước khi `router.push`, hoặc dùng `router.beforeResolve` bật cờ loading toàn cục. Cách nhẹ nhất — trong `handleSubmit`:

```ts
async function handleSubmit(payload: { email: string; password: string }) {
  errorMessage.value = ''
  try {
    await authStore.login(payload.email, payload.password)
    teamsConnecting.value = true // tái dùng overlay có sẵn để không "chớp" màn trắng khi chunk game đang tải
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.push(redirect)
  } catch {
    errorMessage.value = authStore.error
  }
}
```

### C2. (Nâng cao) Prefetch dữ liệu bootstrap song song

Có thể gọi `realtimeService.getBootstrap()` ngay sau khi login thành công và cache kết quả để `GameCanvas` dùng lại, thay vì đợi mount. Nếu muốn giữ đơn giản thì bỏ qua C2 — A + B đã giải quyết phần lớn.

---

## D. (Tuỳ chọn) Giảm `village_adventure.tmj` 1.5 MB

Tilemap JSON này nạp trong `PreloadScene` và phải parse trên main thread. Nếu bạn export từ Tiled:
- Trong Tiled: **Map → Map Properties → Tile Layer Format** chọn **Base64 (zlib compressed)** thay vì CSV → JSON nhỏ hơn nhiều.
- Hoặc **File → Export As** dạng `.json` đã nén, và cân nhắc bỏ các layer/thuộc tính không dùng.
Việc này giúp bước "tải tài nguyên" (thanh progress) chạy nhanh hơn sau khi đã vào được PreloadScene.

---

## E. Cách kiểm chứng

1. DevTools → **Network** → **Fast 3G** + **Disable cache**. Đăng nhập và bấm giờ từ lúc click login đến khi thanh progress hiện. So sánh trước/sau.
2. DevTools → tab **Coverage** (Ctrl/Cmd+Shift+P → "Show Coverage") → record → vào game → xem % JS chưa dùng trong `phaser`/`GameView` chunk.
3. Đặt mốc đo chính xác bằng `performance.mark`:
   ```ts
   // trong handleSubmit ngay trước router.push:
   performance.mark('login-click')
   // trong PreloadScene.preload(), ở đầu hàm:
   performance.mark('preload-start')
   performance.measure('login→preload', 'login-click', 'preload-start')
   console.log(performance.getEntriesByName('login→preload')[0].duration, 'ms')
   ```
4. Sau khi thêm mục B, chạy `npx vite build` và xác nhận có chunk `phaser-*.js` riêng.
