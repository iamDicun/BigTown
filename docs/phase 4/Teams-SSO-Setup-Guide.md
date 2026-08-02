# BigTown — Hướng dẫn tích hợp đăng nhập & nhúng ứng dụng vào Microsoft Teams

> Tài liệu này hướng dẫn: (1) cho phép người dùng **đăng nhập BigTown bằng tài khoản Teams (SSO)**, và (2) **nhúng BigTown thành một tab chạy ngay trong Microsoft Teams**.
>
> **Điểm mấu chốt cần biết trước:** phần backend Go **đã làm gần như xong**. Endpoint `POST /api/auth/teams`, bộ xác thực token Microsoft (`MicrosoftTokenVerifier`, có cache JWKS), và luồng find-or-create user + liên kết `user_identity` đều đã tồn tại trong `internal/module/auth/`. Việc còn lại **không phải viết thêm code Go**, mà là **cấu hình phía Microsoft, tích hợp SDK ở frontend, và đóng gói app**. Tài liệu tập trung vào phần đó.

---

## Mục lục — Tại sao cần từng bước & ý nghĩa

Đọc bảng này trước để hiểu bức tranh tổng thể: mỗi bước tồn tại để giải quyết điều gì, nếu bỏ thì hỏng ở đâu.

| # | Bước | Tại sao cần / Ý nghĩa |
|---|------|------------------------|
| **0** | [Hiểu luồng SSO end-to-end](#0-hiểu-luồng-sso-end-to-end) | Nếu không nắm luồng, mọi cấu hình sau chỉ là chép mù và cực khó debug. Bước này vẽ rõ token đi từ Teams → frontend → backend → DB như thế nào, để bạn biết mỗi giá trị cấu hình phục vụ mắt xích nào. |
| **1** | [Đăng ký App trên Microsoft Entra ID](#1-đăng-ký-app-trên-microsoft-entra-id-azure-ad) | Đây là nơi Microsoft "biết đến" app của bạn và cấp **Client ID**. Không có nó, Microsoft không cấp token, và `MicrosoftTokenVerifier` (check `aud == clientID`) sẽ từ chối mọi token. Đây là gốc rễ của toàn bộ tin cậy. |
| **2** | [Cấu hình Expose an API + scope `access_as_user`](#2-expose-an-api--scope-access_as_user) | Teams chỉ chịu cấp SSO token cho app khi app khai báo rõ "tôi là một API có thể được truy cập thay mặt người dùng". `Application ID URI` và scope này là điều kiện để `getAuthToken()` ở frontend hoạt động. |
| **3** | [Ủy quyền cho client Teams](#3-ủy-quyền-cho-các-client-của-teams-pre-authorize) | Nếu không pre-authorize các client ID của Teams, mỗi người dùng sẽ bị bắt bấm màn hình "consent" — hỏng trải nghiệm SSO liền mạch. Bước này khiến đăng nhập trở nên "im lặng". |
| **4** | [Điền biến môi trường backend](#4-cấu-hình-backend-biến-môi-trường) | Nối các giá trị từ Entra vào `TEAMS_CLIENT_ID` / `TEAMS_TENANT_ID` mà code đã đọc sẵn. Sai/thiếu ở đây → verifier trả "token không hợp lệ" dù token thật ra đúng. |
| **5** | [Tích hợp Teams SDK ở frontend](#5-frontend-lấy-sso-token-và-đổi-lấy-phiên-bigtown) | Đây là mắt xích lấy token thật từ Teams (`getAuthToken()`) rồi gọi API backend đã có. Không có bước này thì backend dù sẵn sàng cũng không có token nào để verify. |
| **6** | [Viết Teams App Manifest](#6-teams-app-manifest--đóng-gói-tab) | Manifest chính là phần **"nhúng app vào Team"**. Nó khai báo BigTown là một *tab*, trỏ tới URL frontend, và bật SSO qua `webApplicationInfo`. Đây là thứ biến website thành app Teams. |
| **7** | [Cấu hình cho phép nhúng iframe (CSP)](#7-cho-phép-nhúng-trong-iframe-csp--x-frame-options) | App Teams chạy trong iframe. Nếu frontend chặn iframe (`X-Frame-Options: DENY` hoặc thiếu `frame-ancestors`), tab sẽ **trắng trơn** dù mọi thứ khác đúng. Đây là lỗi phổ biến nhất và khó đoán nhất. |
| **8** | [Upload & test trong Teams](#8-đóng-gói-upload-và-kiểm-thử) | Kiểm thử thật trong môi trường Teams (iframe + SSO thật) khác hẳn test trên trình duyệt thường. Bước này bắt các lỗi chỉ xuất hiện trong ngữ cảnh Teams. |
| **9** | [Bảo mật & lưu ý vận hành](#9-bảo-mật--lưu-ý-vận-hành) | SSO liên quan trực tiếp tới danh tính người dùng; bỏ qua các kiểm tra tenant/audience có thể mở đường cho token từ tenant lạ. Bước này chốt lại các bảo vệ. |

> **Thứ tự làm khuyên dùng:** 0 → 1 → 2 → 3 → 4 → **test API bằng Postman trước (mục 5.4)** → 5 → 6 → 7 → 8. Lý do: nên xác nhận backend verify token đúng **trước khi** đụng vào iframe Teams, vì debug SSO bên trong Teams rất khó chịu. Tách nhỏ để cô lập lỗi từng lớp.

---

## 0. Hiểu luồng SSO end-to-end

```
┌──────────┐   getAuthToken()    ┌──────────────┐   POST /api/auth/teams   ┌──────────────┐
│  Teams    │ ──────────────────▶ │  Frontend     │ ───────────────────────▶ │  Backend Go   │
│ (client)  │   trả SSO token     │ (BigTown web) │   { ssoToken }           │              │
└──────────┘   (JWT do Entra ký)  └──────────────┘                          │ 1. verify JWT │
                                          ▲                                  │    qua JWKS   │
                                          │  { accessToken, refreshToken }   │    Microsoft  │
                                          └──────────────────────────────────│ 2. find/create│
                                             (phiên đăng nhập của BigTown)    │    user+iden. │
                                                                             │ 3. cấp JWT    │
                                                                             │    của BigTown│
                                                                             └──────────────┘
```

Diễn giải:

1. Người dùng mở BigTown (đang là một tab bên trong Teams). Frontend gọi `microsoftTeams.authentication.getAuthToken()`. Microsoft Entra cấp một **SSO token** (JWT) đại diện cho người dùng đó, có `aud` = Client ID của app bạn.
2. Frontend gửi token này lên `POST /api/auth/teams` (endpoint **đã có sẵn**).
3. Backend (`MicrosoftTokenVerifier.Verify`) tải khóa công khai của Microsoft (JWKS, có cache 1 giờ), verify chữ ký, kiểm `aud == TEAMS_CLIENT_ID` và `tid` thuộc tenant cho phép. Lấy ra `oid` (định danh người dùng bất biến), `tid` (tenant), email, tên.
4. `TeamsLogin` usecase (**đã có sẵn**) tìm `user_identity` theo `(provider='teams', tenant_id, oid)`. Có rồi → đăng nhập. Chưa có → tạo user (hoặc gắn vào user cùng email) + tạo bản ghi identity, trong một transaction.
5. Backend trả về **access/refresh token của chính BigTown** — từ đây frontend dùng phiên BigTown như đăng nhập thường, không phụ thuộc token Microsoft nữa.

> Ý nghĩa quan trọng: token Microsoft **chỉ dùng một lần để chứng minh danh tính**, sau đó hệ thống chạy bằng phiên riêng của BigTown. Đây là lý do luồng này gọn và an toàn.

---

## 1. Đăng ký App trên Microsoft Entra ID (Azure AD)

**Mục tiêu:** có được `Client ID` và `Tenant ID`.

1. Vào [Microsoft Entra admin center](https://entra.microsoft.com) → **Applications** → **App registrations** → **New registration**.
2. Đặt tên (ví dụ `BigTown`). Chọn **Supported account types**:
   - Chỉ nội bộ tổ chức bạn → *Single tenant*.
   - Cho nhiều tổ chức → *Multitenant*. (Ảnh hưởng tới `TEAMS_TENANT_ID` ở mục 4.)
3. Redirect URI: có thể để trống lúc này (SSO qua Teams SDK không bắt buộc redirect URI kiểu web truyền thống).
4. Bấm **Register**. Ở trang Overview, chép lại:
   - **Application (client) ID** → chính là `TEAMS_CLIENT_ID`.
   - **Directory (tenant) ID** → dùng cho `TEAMS_TENANT_ID` nếu single-tenant.

> Ý nghĩa: bản ghi này là "danh tính" của app trước Microsoft. Client ID sẽ nằm trong `aud` của mọi SSO token — và `MicrosoftTokenVerifier` từ chối token nếu `aud` không khớp. Sai một ký tự là toàn bộ đăng nhập fail.

---

## 2. Expose an API + scope `access_as_user`

**Mục tiêu:** khai báo app là một API có thể được gọi thay mặt người dùng, để Teams đồng ý cấp SSO token.

1. Trong app vừa tạo → **Expose an API** → **Add** (Application ID URI). Đặt dạng:
   ```
   api://<domain-frontend-cua-ban>/<client-id>
   ```
   Ví dụ nếu frontend ở `big-town.vercel.app`:
   ```
   api://big-town.vercel.app/<client-id>
   ```
2. **Add a scope** → tên scope: `access_as_user`.
   - Who can consent: **Admins and users**.
   - Điền các nhãn hiển thị (admin/user consent title & description), ví dụ "Truy cập BigTown thay mặt bạn".
   - State: **Enabled**.

> Ý nghĩa: `getAuthToken()` bên frontend chỉ hoạt động khi app có Application ID URI hợp lệ và scope `access_as_user`. Đây là "hợp đồng" giữa app và Teams về việc token được cấp cho ai và làm gì. Thiếu bước này, `getAuthToken()` sẽ lỗi.

---

## 3. Ủy quyền cho các client của Teams (pre-authorize)

**Mục tiêu:** để người dùng **không** bị bắt bấm màn hình consent mỗi lần → SSO "im lặng".

Vẫn trong **Expose an API**, mục **Authorized client applications** → **Add a client application**, thêm các client ID chính thức của Microsoft Teams (Teams web, desktop, mobile) và tick chọn scope `access_as_user` vừa tạo. Danh sách client ID của Teams được Microsoft công bố trong tài liệu "Enable SSO for tab app" — tra cứu bản mới nhất tại Microsoft Learn vì ID có thể được cập nhật theo thời gian.

> Ý nghĩa: pre-authorize nghĩa là "các client Teams này được phép lấy token cho scope của tôi mà không cần hỏi lại người dùng". Bỏ qua bước này thì đăng nhập vẫn chạy nhưng kèm popup consent gây khó chịu, phá vỡ tính "liền mạch" của SSO.

---

## 4. Cấu hình backend (biến môi trường)

Code đã đọc sẵn hai biến này (`internal/platform/config/config.go` → `TeamsConfig`). Chỉ cần điền vào `.env`:

```bash
TEAMS_CLIENT_ID=<Application (client) ID từ bước 1>
# Single-tenant: dùng Directory (tenant) ID.
# Multi-tenant: để "common" (mặc định) hoặc "organizations".
TEAMS_TENANT_ID=<tenant-id hoặc common>
```

> Ý nghĩa & cạm bẫy:
> - `MicrosoftTokenVerifier` dùng `TEAMS_CLIENT_ID` để check `aud` — sai giá trị này là fail toàn bộ.
> - `TEAMS_TENANT_ID`: nếu để một tenant cụ thể, verifier sẽ **từ chối** token từ tenant khác (`Teams token tenant is not allowed`). Để `common` sẽ chấp nhận mọi tenant — chỉ dùng khi bạn chủ đích làm multi-tenant, và khi đó cần cân nhắc kiểm soát tenant nào được phép đăng nhập (xem mục 9).
> - Verifier tự động tải JWKS từ `https://login.microsoftonline.com/<tenant>/discovery/v2.0/keys`, cache 1 giờ — không cần cấu hình gì thêm.

---

## 5. Frontend: lấy SSO token và đổi lấy phiên BigTown

Frontend đã có `auth.service.ts` / `auth.store.ts` tham chiếu teams — phần dưới là các mảnh cần đảm bảo có mặt.

### 5.1 Cài SDK

```bash
npm install @microsoft/teams-js
```

### 5.2 Khởi tạo SDK khi app chạy trong Teams

```ts
import { app, authentication } from "@microsoft/teams-js";

export async function initTeams(): Promise<boolean> {
  try {
    await app.initialize();
    return true; // đang chạy bên trong Teams
  } catch {
    return false; // chạy như web thường (ngoài Teams) -> dùng login email/mật khẩu
  }
}
```

### 5.3 Lấy SSO token và gọi backend

```ts
export async function loginWithTeams(): Promise<void> {
  const ssoToken = await authentication.getAuthToken();
  // Gọi endpoint ĐÃ CÓ SẴN của backend:
  const res = await fetch("/api/auth/teams", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include", // để nhận refresh_token cookie giống luồng login thường
    body: JSON.stringify({ ssoToken }),
  });
  if (!res.ok) throw new Error("Teams login failed");
  const data = await res.json();
  // lưu access token vào auth.store như luồng đăng nhập thường
}
```

> Kiểm tra tên field JSON mà `TeamsLogin` handler mong đợi trong `auth/delivery/dto.go` (ví dụ `ssoToken` hay `sso_token`) và khớp đúng ở body request.

### 5.4 Test backend TRƯỚC khi đụng iframe (khuyến nghị mạnh)

Trước khi làm manifest, xác nhận backend verify đúng: lấy tạm một SSO token thật (log ra từ `getAuthToken()` khi chạy thử, hoặc dùng token test) rồi gọi thẳng bằng `curl`/Postman:

```bash
curl -X POST https://<backend>/api/auth/teams \
  -H "Content-Type: application/json" \
  -d '{"ssoToken":"<token>"}'
```

Kỳ vọng: trả access/refresh token của BigTown. Nếu lỗi ở đây, vấn đề nằm ở cấu hình Entra (mục 1–4), **không phải** ở iframe — cô lập được lỗi sớm.

---

## 6. Teams App Manifest — đóng gói tab

**Đây là phần "nhúng app vào Team".** Một Teams app package là file `.zip` gồm 3 thứ: `manifest.json`, `color.png` (192×192), `outline.png` (32×32).

`manifest.json` tối thiểu cho một **tab** có SSO:

```json
{
  "$schema": "https://developer.microsoft.com/en-us/json-schemas/teams/v1.19/MicrosoftTeams.schema.json",
  "manifestVersion": "1.19",
  "id": "<TẠO-MỘT-GUID-MỚI-cho-app>",
  "version": "1.0.0",
  "developer": {
    "name": "BigTown",
    "websiteUrl": "https://big-town.vercel.app",
    "privacyUrl": "https://big-town.vercel.app/privacy",
    "termsOfUseUrl": "https://big-town.vercel.app/terms"
  },
  "name": { "short": "BigTown", "full": "BigTown" },
  "description": { "short": "BigTown trong Teams", "full": "Chơi BigTown ngay trong Microsoft Teams" },
  "icons": { "color": "color.png", "outline": "outline.png" },
  "accentColor": "#1e2327",

  "staticTabs": [
    {
      "entityId": "bigtown-home",
      "name": "BigTown",
      "contentUrl": "https://big-town.vercel.app",
      "scopes": ["personal"]
    }
  ],
  "configurableTabs": [
    {
      "configurationUrl": "https://big-town.vercel.app/teams-config",
      "scopes": ["team", "groupChat"]
    }
  ],

  "webApplicationInfo": {
    "id": "<TEAMS_CLIENT_ID — TRÙNG với Application (client) ID ở bước 1>",
    "resource": "api://big-town.vercel.app/<TEAMS_CLIENT_ID>"
  },

  "validDomains": ["big-town.vercel.app"]
}
```

Giải nghĩa các phần và **tại sao** cần:

- `staticTabs` (scope `personal`): tab cá nhân, mỗi người tự mở BigTown cho riêng mình. Không có nó thì app không có gì để hiển thị.
- `configurableTabs` (scope `team`/`groupChat`): cho phép **gắn BigTown vào một Team/kênh** (đúng yêu cầu "gắn App này vào Team"). `configurationUrl` trỏ tới một trang cấu hình nhỏ ở frontend, gọi `pages.config.setConfig(...)` của Teams SDK để chốt `contentUrl`.
- `webApplicationInfo`: **mắt xích bật SSO**. `id` phải trùng Client ID, `resource` phải trùng Application ID URI ở mục 2. Sai một trong hai → `getAuthToken()` fail.
- `validDomains`: whitelist domain mà tab được phép điều hướng tới. Thiếu domain frontend ở đây → Teams chặn tải trang.

> Công cụ khuyên dùng: **Developer Portal for Teams** (dev.teams.microsoft.com) có UI tạo/validate manifest và xuất `.zip`, đỡ sai cú pháp so với viết tay.

---

## 7. Cho phép nhúng trong iframe (CSP / X-Frame-Options)

**Đây là lỗi phổ biến & khó đoán nhất.** App Teams chạy trong `<iframe>` của `teams.microsoft.com`. Nếu frontend BigTown gửi header chặn iframe, tab sẽ **trắng trơn** dù SSO và manifest đều đúng.

Đảm bảo response của frontend (trang được nhúng):

- **Không** set `X-Frame-Options: DENY` hoặc `SAMEORIGIN`.
- Có header cho phép Teams làm ancestor:
  ```
  Content-Security-Policy: frame-ancestors 'self' teams.microsoft.com *.teams.microsoft.com *.skype.com *.office.com *.microsoft.com;
  ```

Vì frontend deploy trên Vercel, cấu hình trong `vercel.json`:

```json
{
  "headers": [
    {
      "source": "/(.*)",
      "headers": [
        {
          "key": "Content-Security-Policy",
          "value": "frame-ancestors 'self' teams.microsoft.com *.teams.microsoft.com *.skype.com *.office.com *.microsoft.com;"
        }
      ]
    }
  ]
}
```

> Ý nghĩa: `frame-ancestors` là "ai được phép nhúng trang này vào iframe của họ". Không liệt kê domain Teams = Teams bị từ chối nhúng = màn hình trắng. Kiểm tra bằng DevTools trong Teams desktop (có thể bật) nếu nghi ngờ.

---

## 8. Đóng gói, upload và kiểm thử

1. Nén `manifest.json` + `color.png` + `outline.png` thành một `.zip` (ba file **ở gốc** zip, không nằm trong thư mục con).
2. Trong Teams: **Apps** → **Manage your apps** → **Upload an app** → **Upload a custom app** → chọn `.zip`.
3. Test theo thứ tự:
   - Tab **personal** mở BigTown và tự đăng nhập được (SSO im lặng, không popup).
   - **Add to a team/channel**: thêm tab BigTown vào một Team → xác nhận đúng yêu cầu "gắn App vào Team".
   - Thử trên **Teams desktop và web**, và nếu có, **mobile** — hành vi iframe/SSO có thể khác nhau.

> Nếu tổ chức bật kiểm duyệt app, có thể cần quản trị viên phê duyệt trong Teams Admin Center trước khi người dùng khác cài được. "Upload custom app" thường phải được bật trong policy của tổ chức.

---

## 9. Bảo mật & lưu ý vận hành

- **Kiểm soát tenant.** Nếu để `TEAMS_TENANT_ID=common` (multi-tenant), bất kỳ tài khoản Microsoft nào ở bất kỳ tổ chức nào cũng đăng nhập được. Nếu chỉ muốn nội bộ, đặt tenant ID cụ thể (verifier đã có sẵn kiểm tra `tid`). Với multi-tenant nhưng muốn giới hạn, cân nhắc thêm allowlist tenant ở tầng usecase.
- **Audience.** `MicrosoftTokenVerifier` đã ràng `jwt.WithAudience(clientID)` — giữ nguyên, đừng nới lỏng. Đây là hàng rào chống token cấp cho app khác bị dùng lại.
- **`oid` là khóa định danh bất biến**, không phải email. Code đã dùng `oid` + `tid` làm khóa `user_identity` — đúng, vì email có thể đổi còn `oid` thì không. Không đổi sang khóa theo email.
- **Liên kết theo email khi tạo mới.** `TeamsLogin` hiện gắn identity vào user cùng email nếu đã tồn tại. Cân nhắc rủi ro chiếm tài khoản: chỉ auto-link khi email đã được Microsoft xác minh (thường đúng với tài khoản tổ chức) — ghi chú lại nếu sau này mở cho tài khoản cá nhân.
- **JWKS cache & xoay khóa.** Microsoft xoay khóa ký định kỳ; cache 1 giờ của verifier tự làm mới khi gặp `kid` lạ. Không cần can thiệp, nhưng biết điều này để không hoảng khi thấy verifier gọi ra mạng.
- **HTTPS bắt buộc.** SSO và nhúng iframe Teams đều yêu cầu HTTPS end-to-end (Vercel đã đáp ứng cho frontend; đảm bảo backend cũng sau TLS — xem `docs/Nginx-Deployment-Guide.md`).

---

## Phụ lục — Checklist nhanh

- [ ] Entra: đăng ký app, chép Client ID + Tenant ID (mục 1).
- [ ] Expose an API: Application ID URI + scope `access_as_user` (mục 2).
- [ ] Pre-authorize các client Teams (mục 3).
- [ ] `.env`: `TEAMS_CLIENT_ID`, `TEAMS_TENANT_ID` (mục 4).
- [ ] **Test `POST /api/auth/teams` bằng Postman trước** (mục 5.4).
- [ ] Frontend: `@microsoft/teams-js`, `app.initialize()`, `getAuthToken()` → gọi backend (mục 5).
- [ ] `manifest.json` + 2 icon, `webApplicationInfo` khớp Client ID / Application ID URI (mục 6).
- [ ] CSP `frame-ancestors` cho Teams trên Vercel (mục 7).
- [ ] Đóng gói `.zip`, upload, test personal + team, desktop + web (mục 8).
- [ ] Rà soát kiểm soát tenant & audience (mục 9).
