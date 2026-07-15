# Frontend — IT Asset & Hardware Tracking System

Vue 3 + Vite + TypeScript. Cấu trúc theo `frontend-architecture-it-asset-tracking.md` (app/shared/features).

## Chạy thử

```sh
npm install
cp .env.example .env   # chỉnh VITE_API_BASE_URL nếu backend không chạy ở localhost:8080
npm run dev
```

Cần backend chạy song song (xem `backend/README.md`) để login/register/refresh/logout hoạt động thật.

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
  `logout` (service → Pinia store → view/form), dùng làm mẫu copy-paste cho feature tiếp theo.
- `features/dashboard` — 1 trang protected rỗng, chỉ để chứng minh route guard + session hoạt động.

Alias `@` trỏ vào `src/` (cấu hình ở `vite.config.ts` + `tsconfig.app.json`).

## Việc cần làm tiếp

`features/devices`, `features/employees`, `features/notifications`, ... vẫn còn trống (theo đúng
schema/module backend tương ứng — xem mục 9 `frontend-architecture-it-asset-tracking.md`). Copy cấu
trúc `features/auth/{services,stores,views,components,routes.ts}` sang từng feature, theo thứ tự ở
mục 23 của file kiến trúc đó.

## Lệnh kiểm tra

```sh
npm run build   # vue-tsc -b && vite build — dùng để catch lỗi type/build trước khi commit
```
