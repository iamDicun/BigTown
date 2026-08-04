# BigTown — Test luồng Teams SSO bằng Postman

Bộ này gồm 2 file import vào Postman:

- `BigTown-Teams-SSO.postman_collection.json` — collection 8 request có sẵn test tự động.
- `BigTown-Teams.postman_environment.json` — environment, điền `base_url` + `sso_token`.

Chạy tuần tự từ request 0 → 7 (request 8 tùy chọn để đo latency). Mở **View → Show Postman Console** để xem log chi tiết (claims token, cảnh báo cookie...).

---

## 1. Lấy SSO token thật để test

Backend verify chữ ký RSA của Microsoft nên **không thể fake token** — phải là JWT thật do Entra ký, có `aud` = `TEAMS_CLIENT_ID`. Ba cách lấy:

**Cách A — Log ra từ Teams (nhanh nhất).** Tạm thêm 1 dòng vào `handleTeamsLogin()` (`LoginView.vue`):
```ts
const ssoToken = await getTeamsSSOToken()
console.log('SSO_TOKEN', ssoToken)   // copy từ DevTools trong Teams, xong xoá đi
```
Mở app trong Teams (desktop bật DevTools hoặc Teams web), copy token, dán vào `sso_token`. Token sống ~1 giờ.

**Cách B — Mint bằng Postman OAuth 2.0 (không cần vào Teams).** Ở request 1, tab **Authorization → OAuth 2.0 → Get New Access Token**:
- Grant type: `Authorization Code` (có PKCE)
- Auth URL: `https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/authorize`
- Token URL: `https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/token`
- Client ID: `f16074d1-4c0b-4c29-8f00-2072a67d61b8` (chính là `webApplicationInfo.id` trong manifest)
- Scope: `api://big-town.vercel.app/f16074d1-4c0b-4c29-8f00-2072a67d61b8/access_as_user`
- Redirect: thêm `https://oauth.pstmn.io/v1/callback` vào Redirect URI của app trên Entra trước.

Token nhận được có đúng `aud`/`scp` như token Teams cấp. Copy `access_token` vào biến `sso_token`.

**Cách C — Đã có sẵn token test** thì dán thẳng vào `sso_token`.

> Request **0. Preflight** sẽ tự decode token và in `aud / tid / oid / name / email / exp` ra Console — đối chiếu `aud` với `TEAMS_CLIENT_ID` của backend trước khi bắn thật.

---

## 2. Điền environment

| Biến | Giá trị |
|---|---|
| `base_url` | Domain backend, vd `https://api.big-town.vercel.app` hoặc `http://localhost:8080` |
| `sso_token` | JWT lấy ở bước 1 |
| `character_name` | Tên nhân vật muốn tạo (mặc định `PostmanTester`) |
| `base_asset_key` | `player` / `knight` / `wizard` / `tanker` / `hunter` |
| `login_email` / `login_password` | (tùy chọn) tài khoản email/pw để đối chiếu latency ở request 8 |

---

## 3. Các request kiểm tra gì

| # | Request | Khẳng định |
|---|---|---|
| 0 | Preflight decode | token đúng JWT, chưa hết hạn, có `oid`; in `aud`/`tid` để soi audience |
| 1 | `POST /api/auth/teams` | 200, có `access_token`/`token_type`/`expires_in`; **kiểm tra cookie `refresh_token`** (HttpOnly, cảnh báo nếu không `SameSite=None; Secure`); đo latency |
| 2 | `GET /characters/options` | access_token dùng được cho route protected |
| 3 | `GET /characters/me` (trước) | user Teams mới → 404; user cũ → 200 |
| 4 | `POST /characters` | 201, **tên lưu đúng bằng tên gửi**, `base_asset_key` đúng |
| 5 | `GET /characters/me` (sau) | tên persist đúng sau reload, `map_id` đã sync |
| 6 | `POST /auth/refresh` | **điểm nghẽn Teams**: cookie có gửi lại được không |
| 7 | `POST /auth/logout` | thu hồi phiên |
| 8 | `POST /auth/login` (tùy chọn) | đối chiếu latency với Teams login |

---

## 4. Kiểm tra DB (chạy trên Postgres)

Sau khi chạy request 1 và 4, đối chiếu dữ liệu đã ghi:

```sql
-- User được tạo/liên kết từ Teams (tên = claim 'name', email = email/preferred_username)
SELECT id, full_name, email, role
FROM app_user
WHERE email = '<email-trong-token>';

-- Bản ghi liên kết Teams: provider='teams', external_subject = oid, tenant_id = tid
SELECT provider, external_subject, tenant_id, email, created_at
FROM user_identities
WHERE provider = 'teams' AND external_subject = '<oid-trong-token>';

-- User Teams KHÔNG có credential (không đăng ký mật khẩu) — đúng thiết kế
SELECT * FROM credential WHERE user_id = '<user-id>';   -- kỳ vọng 0 dòng

-- Refresh token đã lưu (chỉ lưu HASH, không lưu token thô — đúng)
SELECT user_id, left(token_hash, 12) AS hash_prefix, expires_at, revoked_at
FROM refresh_token WHERE user_id = '<user-id>' ORDER BY created_at DESC;

-- Nhân vật vừa tạo, tên khớp
SELECT id, name, base_asset_key, coins, score, map_id
FROM characters WHERE user_id = '<user-id>';
```

Kỳ vọng: `app_user.full_name` = tên hiển thị trong Teams; đúng 1 dòng `user_identities`; không có `credential`; `refresh_token` chỉ lưu hash; `characters.name` = `character_name` đã gửi.

---

## 5. Những điều đã rà & điểm cần chú ý (đọc kỹ)

**Luồng nạp app vào Teams** — manifest hợp lệ: `staticTabs` (personal) + `configurableTabs` (team/groupChat) nên vừa mở riêng vừa gắn vào Team được; `webApplicationInfo` có đủ để bật SSO; `validDomains` khớp domain FE. Cần đảm bảo thêm CSP `frame-ancestors` cho Teams (mục 7 trong doc gốc) nếu không tab sẽ trắng.

**Luồng SSO backend** — chắc: verify chữ ký qua JWKS Microsoft (cache 1h), ràng `aud == TEAMS_CLIENT_ID`, và **có** enforce tenant khi cấu hình tenant cụ thể (`claims.TID`). Dùng `oid`+`tid` làm khóa định danh (đúng, không dùng email). Find-or-create + link theo email chạy trong transaction.

**Lưu DB** — giống hệt đăng ký/đăng nhập thường: cùng bảng `app_user`, cùng cơ chế `refresh_token` (lưu hash), thêm bản ghi `user_identities`. Tên lưu từ claim `name`, email từ `email/preferred_username/unique_name`. User Teams không có `credential` (không có mật khẩu) — đúng.

**FE lưu token** — access_token để trong RAM (`window.__accessToken`), mất khi reload; refresh_token là HttpOnly cookie. Trong Teams, iframe reload sẽ mất access_token, nhưng `LoginView.onMounted` tự chạy lại Teams SSO nên đăng nhập lại im lặng — hợp lý.

### Cấu hình backend (đã set trên Render — dùng để đối chiếu)

Các biến dưới đây đã được cấu hình và deploy trên Render:

```bash
COOKIE_SAME_SITE=none                              # cookie gửi được trong iframe Teams (cross-site)
COOKIE_SECURE=true                                 # bắt buộc khi SameSite=None (Render đã HTTPS)
CORS_ALLOWED_ORIGINS=https://big-town.vercel.app   # trùng origin FE, không dấu / ở cuối
TEAMS_CLIENT_ID=<Application (client) ID>           # = webApplicationInfo.id trong manifest, KHÔNG phải manifest.id
TEAMS_TENANT_ID=<Directory (tenant) ID hoặc common>
```

Nếu `/auth/teams` báo lỗi khi test, đối chiếu nhanh theo triệu chứng:

- **401 `Teams SSO token không hợp lệ`** — thường do `aud` không khớp `TEAMS_CLIENT_ID`. Chạy request 0 (Preflight) để in `aud` thật rồi so với biến đã set. Nếu `aud` ra dạng URI `api://.../<clientid>` thì `TEAMS_CLIENT_ID` phải đặt đúng chuỗi URI đó.
- **401 `Teams token tenant is not allowed`** — `TEAMS_TENANT_ID` là tenant cụ thể nhưng token đến từ tenant khác. Đặt đúng tenant hoặc dùng `common` nếu chủ đích multi-tenant.
- **400 `Dữ liệu Teams SSO không hợp lệ`** — sai field body. Phải là `{ "sso_token": ... }` (snake_case), không phải `ssoToken`. (Doc `Teams-SSO-Setup-Guide.md` mục 5.3/5.4 ghi ví dụ nhầm là `ssoToken`; FE thật đã đúng `sso_token`.)
- **CORS bị chặn ở FE** — kiểm tra `CORS_ALLOWED_ORIGINS` trùng chính xác origin FE, không có dấu `/` cuối. Vì `AllowCredentials=true` nên không được để `*`.
- **Refresh 401 dù đã set cookie None/Secure** — Teams desktop/một số webview vẫn có thể chặn third-party cookie hoàn toàn; khi đó chấp nhận dựa vào SSO re-login mỗi lần mở tab (`LoginView.onMounted` đã tự làm).

### Về độ trễ khi đi qua Teams
Backend `/auth/teams` chỉ chậm hơn login thường ở: verify JWT + **lần gọi đầu tiên** fetch JWKS từ Microsoft (timeout 5s, thường ~100–300ms, cache 1 giờ sau đó). Các lần sau ngang login thường. Phần trễ *thấy được với người dùng* chủ yếu ở phía client Teams — `app.initialize()` (handshake postMessage) và `getAuthToken()` (round-trip Entra, im lặng nếu đã pre-authorize, thường ~200–800ms) — phần này Postman không đo được vì nằm trong Teams host. Request 1 và 8 giúp bạn so phần backend; chạy request 1 hai lần liên tiếp để thấy hiệu ứng cache JWKS.
