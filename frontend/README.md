# Frontend — BigTown

Vue 3 + Vite + TypeScript. Cấu trúc giữ theo `app/shared/features`, trong đó game MVP nằm ở `features/game`.

## Chạy thử

```sh
npm install
cp .env.example .env   # chỉnh VITE_API_BASE_URL nếu backend không chạy ở localhost:8080
npm run dev
```

Cần backend chạy song song (xem `backend_boilerplate/README.md`) để login/register/refresh/logout hoạt động thật.

## Đã wiring sẵn (boilerplate, không phải chỉ có md)

- `shared/api/http.ts` — fetch wrapper: tự gắn `Authorization: Bearer <access_token>`, tự nhận diện
  401 và refresh 1 lần duy nhất dù nhiều request cùng fail (single-flight), unwrap response envelope
  `{success, data}` / `{success, code, message}` khớp `backend/internal/response/response.go`.
- `shared/api/tokenStorage.ts` — access token chỉ sống trong biến runtime, không localStorage.
- `shared/utils/jwt.ts` — decode JWT payload phía client (chỉ để đọc `role`/`user_id` cho UI/guard,
  không xác thực chữ ký — verify thật vẫn ở backend).
- `app/router` — `vue-router` compose route theo từng feature (`features/*/routes.ts`), guard xử lý
  `requiresAuth`/`guestOnly`/`roles` theo `to.meta`.
- `app/providers/pinia.ts`, `app/layouts/{AppLayout,AuthLayout}.vue`, `app/layouts/components/Navbar.vue`.
- `features/auth` — **feature đầy đủ, chạy thật với backend**: `login`, `register`, `refresh`,
  `logout` (service → Pinia store → view/form).
- `features/game` — shell cho map/canvas Phaser, chat panel, leaderboard panel, WebSocket event types.

Alias `@` trỏ vào `src/` (cấu hình ở `vite.config.ts` + `tsconfig.app.json`).

## Việc cần làm tiếp

- Gắn Phaser thật vào `features/game/components/GameCanvas.vue`.
- Implement `features/game/network/gameSocket.ts` với auth token và reconnect.
- Nối REST API cho avatar, shop, inventory và leaderboard.

## Lệnh kiểm tra

```sh
npm run build   # vue-tsc -b && vite build — dùng để catch lỗi type/build trước khi commit
```
