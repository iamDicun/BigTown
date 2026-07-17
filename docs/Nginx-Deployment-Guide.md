# BigTown Nginx Deployment Guide
**Hướng dẫn triển khai reverse proxy, HTTPS/WSS và chuẩn bị cho Microsoft Teams**

---

## 1. Mục tiêu

Tài liệu này mô tả cách triển khai Nginx đúng với Deployment View đã thống nhất:

```text
Client Browser / Microsoft Teams
  | HTTPS / WSS
  v
Nginx
  - Terminate TLS
  - Serve frontend static files hoặc proxy frontend
  - Proxy /api vào backend qua HTTP
  - Proxy /connection/websocket vào backend qua WS
  |
  | HTTP / WS nội bộ
  v
Golang Backend
  - REST API
  - Centrifuge WebSocket endpoint
  - Realtime room channels
  |
  v
PostgreSQL
```

Mục tiêu triển khai theo 2 giai đoạn:

1. **Local Nginx không TLS:** test reverse proxy và WebSocket upgrade trước.
2. **HTTPS/WSS:** thêm TLS để chạy giống môi trường Teams/production.

---

## 2. Hiện trạng dev local

Hiện tại khi dev local chưa có Nginx:

```text
Browser
├── HTTP -> http://localhost:5173      frontend Vite dev server
├── HTTP -> http://localhost:8080/api  backend REST API
└── WS   -> ws://localhost:8080/connection/websocket
```

Backend Gin nhận trực tiếp:

```text
REST API: /api/...
Realtime: /connection/websocket
```

Sau khi thêm Nginx, browser nên chỉ gọi một origin:

```text
http://localhost:8088
```

Hoặc khi có TLS:

```text
https://your-domain.com
wss://your-domain.com/connection/websocket
```

---

## 3. Target local reverse proxy

Giai đoạn local nên chạy như sau:

```text
Browser
  |
  | http://localhost:8088
  v
Nginx local
  ├── /                    -> frontend/dist static files
  ├── /api                 -> http://host.docker.internal:8080/api
  └── /connection/websocket -> ws://host.docker.internal:8080/connection/websocket

Backend
  └── go run ./cmd/server on :8080

PostgreSQL
  └── Docker compose on localhost:5433
```

Local Nginx chưa cần TLS. Mục tiêu là kiểm tra:

- SPA frontend chạy qua Nginx.
- REST API proxy đúng.
- WebSocket/Centrifuge upgrade đúng.
- Chat realtime 2 tab vẫn hoạt động.

---

## 4. Frontend URL config

Khi chạy qua Nginx cùng origin, frontend nên gọi API bằng relative URL:

```env
VITE_API_BASE_URL=/api
```

Khi chạy dev Vite trực tiếp, vẫn có thể dùng:

```env
VITE_API_BASE_URL=http://localhost:8080/api
```

Frontend helper nên hỗ trợ cả absolute và relative URL. Logic mong muốn:

```ts
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api'
const baseUrl = new URL(apiBaseUrl, window.location.origin)
```

Với `/api`, `new URL('/api', window.location.origin)` sẽ thành:

```text
http://localhost:8088/api
```

WebSocket URL cũng nên suy ra từ API base URL:

```text
http://localhost:8088/api -> ws://localhost:8088/connection/websocket
https://domain.com/api    -> wss://domain.com/connection/websocket
```

---

## 5. Nginx local config

Tạo file:

```text
nginx/nginx.local.conf
```

Nội dung đề xuất:

```nginx
events {}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    sendfile on;

    upstream backend_api {
        server host.docker.internal:8080;
    }

    server {
        listen 80;
        server_name localhost;

        root /usr/share/nginx/html;
        index index.html;

        location / {
            try_files $uri $uri/ /index.html;
        }

        location /api/ {
            proxy_pass http://backend_api/api/;
            proxy_http_version 1.1;

            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /connection/websocket {
            proxy_pass http://backend_api/connection/websocket;
            proxy_http_version 1.1;

            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";

            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            proxy_read_timeout 3600s;
            proxy_send_timeout 3600s;
        }
    }
}
```

Ghi chú:

- `host.docker.internal` giúp container Nginx gọi backend đang chạy trên host Windows.
- Nếu backend cũng chạy container cùng Docker network, đổi upstream thành service name, ví dụ `server backend:8080;`.
- WebSocket bắt buộc có `Upgrade` và `Connection "upgrade"`.

---

## 6. Docker compose cho Nginx local

Có 2 lựa chọn.

### Lựa chọn A: thêm vào `backend/docker-compose.yml`

Ưu điểm: dùng chung compose hiện có.

Nhược điểm: compose backend sẽ phụ thuộc frontend build output.

Ví dụ:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: app_db
    ports:
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./internal/database/schema.sql:/docker-entrypoint-initdb.d/01_schema.sql:ro

  nginx:
    image: nginx:1.27-alpine
    restart: unless-stopped
    ports:
      - "8088:80"
    volumes:
      - ../nginx/nginx.local.conf:/etc/nginx/nginx.conf:ro
      - ../frontend/dist:/usr/share/nginx/html:ro
    depends_on:
      - postgres

volumes:
  postgres_data:
```

### Lựa chọn B: tạo compose riêng ở root

Ví dụ:

```text
docker-compose.local.yml
```

Ưu điểm: rõ ràng hơn khi sau này có nhiều service.

Nhược điểm: thêm một file compose.

---

## 7. Quy trình chạy local qua Nginx

### Bước 1: chạy Postgres

```sh
cd backend
docker compose up -d postgres
```

### Bước 2: chạy backend

```sh
cd backend
go run ./cmd/server
```

Backend nên chạy ở:

```text
http://localhost:8080
```

### Bước 3: build frontend với relative API URL

Tạo hoặc sửa `frontend/.env.production`:

```env
VITE_API_BASE_URL=/api
```

Build frontend:

```sh
cd frontend
npm run build
```

### Bước 4: chạy Nginx

Nếu dùng compose trong backend:

```sh
cd backend
docker compose up -d nginx
```

### Bước 5: test browser

Mở:

```text
http://localhost:8088
```

Kiểm tra:

- Register/login local.
- Mở 2 tab.
- Gửi chat trong `ChatPanel`.
- Tab còn lại nhận được message.

---

## 8. Debug Nginx local

Kiểm tra container:

```sh
docker compose ps
```

Xem log Nginx:

```sh
docker compose logs -f nginx
```

Kiểm tra config Nginx trong container:

```sh
docker compose exec nginx nginx -t
```

Nếu REST API lỗi:

- Kiểm tra backend đang chạy `localhost:8080`.
- Kiểm tra upstream `host.docker.internal:8080`.
- Kiểm tra frontend đang dùng `VITE_API_BASE_URL=/api`.

Nếu WebSocket lỗi:

- Kiểm tra route `/connection/websocket` trong backend log.
- Kiểm tra Nginx location có `proxy_http_version 1.1`.
- Kiểm tra có header `Upgrade` và `Connection "upgrade"`.
- Kiểm tra browser devtools tab Network -> WS.

---

## 9. HTTPS/WSS với TLS

Sau khi local Nginx HTTP chạy ổn, thêm TLS.

Production/staging target:

```text
Client
  | HTTPS / WSS
  v
Nginx :443
  | HTTP / WS nội bộ
  v
Backend :8080
```

Nginx TLS server block:

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate     /etc/nginx/certs/fullchain.pem;
    ssl_certificate_key /etc/nginx/certs/privkey.pem;

    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://backend_api/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /connection/websocket {
        proxy_pass http://backend_api/connection/websocket;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$host$request_uri;
}
```

Khi dùng HTTPS, frontend sẽ tự suy ra WebSocket là:

```text
wss://your-domain.com/connection/websocket
```

---

## 10. HTTPS cho Teams dev

Microsoft Teams yêu cầu content URL là HTTPS public domain.

Các cách dev:

- Ngrok.
- Cloudflare Tunnel.
- Microsoft dev tunnels.
- Deploy EC2 tạm với domain + Let's Encrypt.

Ví dụ với ngrok:

```sh
ngrok http 8088
```

Ngrok trỏ vào Nginx local:

```text
https://abc.ngrok-free.app -> http://localhost:8088 -> Nginx -> backend/frontend
```

Khi đó Teams manifest dùng:

```text
contentUrl: https://abc.ngrok-free.app
websiteUrl: https://abc.ngrok-free.app
validDomains: ["abc.ngrok-free.app"]
```

Với cách này frontend/backend cùng origin qua Nginx, nên `VITE_API_BASE_URL=/api` vẫn đúng.

---

## 11. Teams manifest liên quan Nginx

Ví dụ phần quan trọng:

```json
{
  "staticTabs": [
    {
      "entityId": "bigtown",
      "name": "BigTown",
      "contentUrl": "https://abc.ngrok-free.app",
      "websiteUrl": "https://abc.ngrok-free.app",
      "scopes": ["personal"]
    }
  ],
  "validDomains": [
    "abc.ngrok-free.app"
  ],
  "webApplicationInfo": {
    "id": "<TEAMS_CLIENT_ID>",
    "resource": "api://abc.ngrok-free.app/<TEAMS_CLIENT_ID>"
  }
}
```

Backend env tương ứng:

```env
TEAMS_CLIENT_ID=<TEAMS_CLIENT_ID>
TEAMS_TENANT_ID=<tenant-id hoặc common>
```

---

## 12. CORS khi dùng Nginx

Nếu frontend và backend cùng origin qua Nginx:

```text
https://domain.com -> frontend
https://domain.com/api -> backend
```

Thì browser không coi đây là cross-origin. CORS ít gây vấn đề hơn.

Nếu chạy tách origin:

```text
frontend: http://localhost:5173
backend:  http://localhost:8080
```

Thì cần CORS cho frontend origin.

Hiện backend đang hardcode origin dev. Sau này nên đổi sang env:

```env
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:8088,https://abc.ngrok-free.app
```

---

## 13. Checklist trước khi đi tiếp Phaser/map

Trước khi triển khai map/character phức tạp, nên xác nhận hạ tầng local Nginx ổn:

- `http://localhost:8088` load được frontend.
- Register/login thành công qua `/api/auth/*`.
- Refresh token cookie hoạt động.
- `GET /api/realtime/bootstrap` thành công.
- Centrifuge connect qua `/connection/websocket` thành công.
- Chat realtime 2 tab hoạt động qua Nginx.
- Browser devtools không có CORS error.
- Browser devtools Network có WS connection status `101 Switching Protocols`.

---

## 14. Kết luận

Luồng dev nên đi theo thứ tự:

```text
1. Browser direct dev: Vite -> backend
2. Browser qua Nginx local HTTP
3. Browser qua Nginx HTTPS hoặc tunnel HTTPS
4. Teams tab qua HTTPS public domain
5. Phaser map/movement/combat hoàn chỉnh trên nền hạ tầng đã ổn
```

Nginx chỉ là lớp reverse proxy/TLS termination. Business logic vẫn nằm ở backend Golang, realtime transport vẫn là Centrifuge, frontend vẫn chỉ gọi một origin khi deploy qua Nginx.
