# BigTown — Engineering Learning & Implementation Roadmap

> Mục tiêu: **không phát triển thêm gameplay feature**, mà biến BigTown thành một project để học Software Engineering, Backend Engineering, DevOps và Production Engineering.

## Phạm vi đã chọn

Các mục sẽ học và triển khai (theo thứ tự ưu tiên mới):

- [x] 1. Containerization
- [ ] 2. CI/CD
- [ ] 3. Unit Testing (Backend)
- [ ] 4. Unit Testing (Frontend)
- [ ] 5. Integration Testing
- [ ] 6. E2E Testing
- [x] 7. Observability
- [x] 8. Load Testing
- [ ] 9. Security Scanning
- [x] 10. Database Engineering
- [ ] 11. Release Engineering
- [ ] 12. Chaos / Failure Testing
- [ ] 13. Documentation / ADR

> **Đã bỏ:** Distributed System — không phù hợp với chi phí deploy.

---

# 0. Nguyên tắc học

## Không làm theo kiểu "thêm tool cho đẹp"

Mỗi công nghệ phải trả lời được:

1. Nó giải quyết vấn đề gì?
2. Vì sao BigTown cần nó?
3. Nếu không dùng nó thì chuyện gì xảy ra?
4. Có trade-off gì?
5. Làm thế nào chứng minh nó hoạt động?

Ví dụ:

> "Thêm Prometheus"

Không được coi là hoàn thành chỉ vì Prometheus chạy.

Phải đạt:

```text
BigTown
  ↓
Expose metrics
  ↓
Prometheus scrape
  ↓
Grafana dashboard
  ↓
Load test
  ↓
Quan sát metrics thay đổi
  ↓
Rút ra bottleneck
```

---

# 1. Mục tiêu cuối cùng

Sau khi hoàn thành roadmap, BigTown nên có pipeline:

```text
Developer
   │
   ▼
Feature / Refactor branch
   │
   ▼
Pull Request
   │
   ├── Lint
   ├── Unit Test
   ├── Integration Test
   ├── E2E Test
   ├── Security Scan
   └── Build
   │
   ▼
Code Review
   │
   ▼
Merge main
   │
   ▼
Release
   │
   ├── Build Docker Image
   ├── Deploy
   └── Smoke Test
   │
   ▼
Production
   │
   ├── Logs
   ├── Metrics
   ├── Monitoring
   └── Alerts
   │
   ▼
Load / Failure / Security Testing
```

---

# 2. Containerization `[ĐÃ HOÀN THÀNH]`

## Mục tiêu

Docker hóa toàn bộ stack để:

- Dev environment nhất quán (không còn "trên máy tôi chạy được")
- CI/CD có thể build + test trong container
- 1 lệnh `docker compose up` là chạy được toàn bộ

## Kiến trúc Local vs Production

```
LOCAL (Docker)                          PRODUCTION
──────────────                          ──────────

Browser                                 Browser
   │                                       │
   ▼                                       ├──→ Vercel (static files)
localhost:3000 (Nginx)                     │
   │                                       └──→ Render (API + WebSocket)
   ├── /          → static files              │
   ├── /api/*     → proxy backend:8080        ▼
   ├── /connection/* → proxy backend:8080   Render
   │   (WebSocket upgrade)                  (tự xử lý routing,
   └── /metrics   → proxy backend:8080       reverse proxy,
         │                                  HTTPS, CORS
         ▼                                  không cần Nginx)
      backend:8080
         │
         ▼
      postgres:5432
```

| Khác biệt | Local (Docker) | Production |
|-----------|---------------|------------|
| Frontend serve | Nginx trong container | Vercel CDN |
| API routing | Nginx proxy `/api/*` → backend | Render tự reverse proxy |
| WebSocket | Nginx proxy `/connection/*` (upgrade) | Render tự xử lý |
| HTTPS | Không (localhost) | Render + Vercel tự động |
| CORS | Không cần (cùng origin qua Nginx) | Cần cấu hình `CORS_ALLOWED_ORIGINS` |
| `VITE_API_BASE_URL` | `/api` (relative, cùng origin) | `https://...render.com/api` (absolute) |

> **Nguyên tắc:** Docker dùng Nginx để mô phỏng 1 domain duy nhất cho cả frontend + backend, tránh CORS khi dev local. Production thì Vercel + Render là 2 domain riêng, backend đã cấu hình CORS sẵn.

## Stack hiện tại

```text
docker compose up
    │
    ├── postgres:16-alpine (port 5433)
    │   └── auto-migrate: schema.sql → seed.sql khi volume còn trống
    │
    ├── backend:8080 (build từ backend/Dockerfile)
    │   ├── Go binary, Centrifuge embedded (WebSocket trong process)
    │   └── Kết nối PostgreSQL qua container network
    │
    └── frontend:3000 (build từ frontend/Dockerfile)
        ├── Vue 3 + Phaser 4, build bằng Vite
        └── Nginx serve static + reverse proxy API/WS → backend
```

## Cách dùng

```bash
# Build
docker compose build

# Chạy
docker compose up -d

# Truy cập
# Frontend: http://localhost:3000
# Backend:  http://localhost:8080
# Postgres: localhost:5433

# Dừng
docker compose down

# Xoá sạch data (chạy lại từ đầu)
docker compose down -v
```

## Definition of Done

- [x] `frontend/Dockerfile` — multi-stage build (Node build → Nginx serve)
- [x] `frontend/.dockerignore`
- [x] `frontend/nginx.conf` — SPA routing + reverse proxy
- [x] `frontend/src/features/game/network/gameSocket.ts` — hỗ trợ `VITE_API_BASE_URL` relative
- [x] `docker-compose.yml` ở root — postgres + backend + frontend
- [x] `docker compose up` → toàn bộ stack chạy locally
- [x] `.env.example` — JWT_SECRET cho dev
- [x] `schema.sql` + `seed.sql` — gộp tất cả migration vào 2 file duy nhất

---

# 3. CI/CD

## Mục tiêu

Từ:

```text
git push
→ tự test
→ tự build
→ tự deploy
```

thay vì thao tác thủ công.

## Pipeline — xây dựng từng bước

### Bước 1: Basic pipeline (làm trước)

```text
Pull Request
     │
     ▼
┌───────────────┐
│ Lint           │
│ (gofmt,        │
│  go vet,       │
│  vue-tsc,      │
│  eslint)       │
└───────┬───────┘
        ▼
┌───────────────┐
│ Build          │
│ (go build,     │
│  vite build)   │
└───────┬───────┘
        ▼
┌───────────────┐
│ Deploy         │
│ (backend →     │
│  Render,       │
│  frontend →    │
│  Vercel)       │
└───────────────┘
```

### Bước 2: Thêm test sau khi pipeline cơ bản ổn định

```text
Pull Request
     │
     ▼
┌───────────────┐
│ Lint           │
└───────┬───────┘
        ▼
┌───────────────┐
│ Unit Tests     │
└───────┬───────┘
        ▼
┌───────────────┐
│ Integration    │
│ Tests          │
└───────┬───────┘
        ▼
┌───────────────┐
│ E2E Tests      │
└───────┬───────┘
        ▼
┌───────────────┐
│ Build          │
└───────┬───────┘
        ▼
       Deploy
```

### Bước 3: Thêm security + release sau khi testing đã ổn

```text
Pull Request
     │
     ▼
┌───────────────┐
│ Lint           │
└───────┬───────┘
        ▼
┌───────────────┐
│ Unit Tests     │
└───────┬───────┘
        ▼
┌───────────────┐
│ Integration    │
│ Tests          │
└───────┬───────┘
        ▼
┌───────────────┐
│ E2E Tests      │
└───────┬───────┘
        ▼
┌───────────────┐
│ Security Scan  │
└───────┬───────┘
        ▼
┌───────────────┐
│ Build          │
└───────┬───────┘
        ▼
       Merge
        │
        ▼
┌───────────────┐
│ Release        │
│ (tag + deploy  │
│  + smoke test) │
└───────────────┘
```

## Branch strategy

Đơn giản:

```text
main
  ↑
feature/*
bugfix/*
refactor/*
```

PR bắt buộc pass CI trước khi merge.

## GitHub Actions cần có

```text
.github/
└── workflows/
    ├── ci.yml        # lint + test + build
    ├── e2e.yml       # Playwright E2E tests
    ├── security.yml  # Trivy + OWASP ZAP
    └── deploy.yml    # deploy backend + frontend
```

Ban đầu chỉ cần `ci.yml` cho lint + build + deploy.

## Công cụ lint

| Layer    | Tool              |
|----------|-------------------|
| Backend  | `gofmt`, `go vet` |
| Frontend | `vue-tsc`, ESLint |

## Definition of Done

- [x] `ci.yml` chạy lint + build tự động trên mỗi PR
- [x] `deploy.yml` trigger Render deploy hook khi merge main
- [ ] Backend deploy lên Render thành công qua CI/CD
- [ ] Frontend deploy lên Vercel thành công qua CI/CD
- [ ] PR không pass CI thì không merge được (branch protection)

---

# 4. Unit Testing

## Mục tiêu

Test từng function/module riêng lẻ, không cần database hay network.

## Backend (Go)

Các module nên test:

```text
Auth
├── JWT generation / validation
├── Password hashing / verification
└── Token expiration

Realtime
├── Room actor state management
├── Movement validation
├── Join/Leave room logic
└── Message broadcast

Editor
├── Placement validation
├── Map actor logic
└── Item pickup logic

Chat
├── Message persistence
├── Rate limiting logic
└── Room subscription
```

## Frontend (Vue)

Các module nên test:

```text
Auth
├── Login form validation
├── Token storage
└── Auth state management (Pinia store)

Game
├── Movement interpolation
├── Network message parsing
└── State synchronization
```

## Công cụ

| Layer    | Tool    |
|----------|---------|
| Backend  | `go test`, `testify` |
| Frontend | Vitest |

## Definition of Done

- [ ] Auth module có unit test
- [ ] Realtime module có unit test
- [ ] Editor module có unit test
- [ ] Frontend auth store có unit test
- [ ] Unit test chạy tự động trên CI
- [ ] Coverage > 0% ban đầu (tăng dần theo thời gian)

---

# 5. Integration Testing

## Mục tiêu

Test sự tương tác giữa các module, bao gồm database thật (không mock).

```text
Backend handler
   ↓
Use case layer
   ↓
Repository layer
   ↓
PostgreSQL (test db)
```

## Các flow nên test

### INT-01 — Auth flow

```text
Register user
→ Login
→ Get JWT
→ Call protected endpoint
→ Verify response
```

### INT-02 — Character flow

```text
Login
→ Create character
→ Verify character exists in DB
→ Select character
→ Enter game
```

### INT-03 — Chat flow

```text
Login as User A + User B
→ User A sends message
→ Verify message in DB
→ User B receives message via WebSocket
```

### INT-04 — Realtime flow

```text
Login
→ Join room
→ Send movement
→ Verify broadcast received
→ Verify validation (reject invalid movement)
```

## Infrastructure

```text
Docker Compose
├── PostgreSQL (test instance)
├── Backend
└── Centrifuge
```

## Công cụ

| Layer    | Tool              |
|----------|-------------------|
| Backend  | `go test` + `testcontainers-go` hoặc Docker Compose |
| DB       | PostgreSQL test instance, migrate + seed |

## Definition of Done

- [ ] Có Docker Compose cho test infrastructure
- [ ] Auth flow integration test
- [ ] Character flow integration test
- [ ] Chat flow integration test
- [ ] Realtime flow integration test
- [ ] Integration test chạy tự động trên CI
- [ ] Test DB được tạo/sạch sau mỗi lần chạy

---

# 6. E2E Testing

## Mục tiêu

Kiểm thử hệ thống từ góc nhìn người dùng thật.

Không mock toàn bộ backend.

Ví dụ:

```text
Browser
  ↓
Vue
  ↓
API
  ↓
Go
  ↓
PostgreSQL
```

## Công cụ đề xuất

- Playwright
- Chromium
- Test environment riêng
- Docker Compose cho dependency

## Các flow nên test

### E2E-01 — Register

```text
Open website
→ Register
→ Submit
→ Registration successful
```

### E2E-02 — Login

```text
Open website
→ Login
→ Receive authenticated state
→ Redirect to game
```

### E2E-03 — Character

```text
Login
→ Create / select character
→ Enter game
```

### E2E-04 — Gameplay

```text
Login
→ Enter map
→ Move
→ Change map
→ Collect coin
```

### E2E-05 — Chat

```text
User A login
User B login
→ Same room
→ A sends message
→ B receives message
```

### E2E-06 — Realtime

```text
User A
    │
    │ movement
    ▼
Backend
    │
    ▼
User B sees updated position
```

## Ghi chú quan trọng

Đừng cố test mọi animation, sound effect hoặc pixel-perfect UI ngay từ đầu.

Ưu tiên:

- Authentication
- API integration
- Persistence
- Realtime
- Critical user journey

## Definition of Done

- [ ] Playwright chạy được locally
- [ ] Có test login
- [ ] Có test game entry
- [ ] Có test chat
- [ ] Có test realtime
- [ ] Test chạy được trên CI
- [ ] Test failure có screenshot/video/log để debug

---

# 7. Observability `[ĐÃ HOÀN THÀNH]`

## Hiện trạng

Đã triển khai:

- **Prometheus**: metrics endpoint tại `/metrics`, scrape từ `host.docker.internal:8080/metrics` và Render production
- **Grafana**: dashboard "BigTown Realtime" với 8 panels (goroutines, CPU, heap, GC, HTTP throughput, OS memory)
- **Middleware**: structured logging (`logger.go`, `requestID.go`), Prometheus metrics (`prometheus.go`)
- **Stack**: `testing/loadtest/monitoring/docker-compose.yml` chạy Prometheus + Grafana + Node Exporter

## Ba trụ cột

```text
Logs    ← đã có structured logging
Metrics ← đã có Prometheus + Grafana
Traces  ← học sau
```

## Các dashboard đã có

### BigTown Realtime

```text
Goroutines (leak detection)
CPU per Core
Heap Memory (MB)
GC Pause (ms) & GC Runs/sec
HTTP Throughput/sec
OS Memory
```

## Những thứ có thể thêm sau

### Dashboard — API

```text
Requests/sec
Error rate
P50
P95
P99
```

### Dashboard — Realtime

```text
Active players
Connections
Messages/sec
Movement/sec
Rejected movements
```

### Dashboard — PostgreSQL

```text
Connections
Query latency
Errors
```

## Ghi chú

Một metric chỉ có giá trị nếu bạn biết:

> "Nếu metric này tăng thì tôi phải làm gì?"

Ví dụ:

```text
P95 latency ↑
        ↓
Check CPU
        ↓
Check DB latency
        ↓
Check goroutines
        ↓
Profile
```

---

# 8. Load Testing `[ĐÃ HOÀN THÀNH]`

## Hiện trạng

Đã triển khai đầy đủ:

- **5 kịch bản k6**: smoke test, bootstrap load, chat load, movement load, placement load
- **4 phase kết quả**: baseline → actor per-room → tick broadcast → full 4-scenario
- **Report**: `testing/loadtest/report.md` so sánh toàn bộ 4 phase
- **Infrastructure**: seed 100 users + 10 maps, token generator

## Các scenario đã có

```text
Smoke test         → 1 VU, pre-flight check
Bootstrap load     → ramping arrival rate, peak ~200 RPS
Chat load          → 100 VU constant soak 5min, REST + WS
Movement load      → 100 VU constant soak 5min, WS RPC + broadcast
Placement load     → 100 VU constant soak 5min, REST + WS delivery
```

## Tool

k6

## Ghi chú

> **Không optimize chỉ vì cảm giác code chậm.**
> Hãy benchmark → tìm bottleneck → sửa → benchmark lại.

---

# 9. Security Scanning

## Mục tiêu

Chuyển từ:

> "Có authentication"

thành:

> "Có security model và đã kiểm thử nó."

## Đã có (code-level)

```text
Password hashing    → bcrypt
JWT                 → generation + validation + expiration
Auth middleware     → JWT Bearer token
CORS                → config
CSP headers         → vercel.json (Teams iframe)
```

## Cần làm (automated scanning)

### Dependency Scanning

```text
Trivy           → scan Docker image
Dependabot      → Go modules + npm packages
```

### SAST (Static Analysis)

```text
gosec           → Go security scanner
npm audit       → npm vulnerability check
```

### DAST (Dynamic Analysis)

```text
OWASP ZAP       → scan running application
```

## Các test case cần kiểm thử

### Authentication

- password hashing
- JWT expiration
- refresh token
- token rotation
- logout/invalidation
- cookie configuration
- secret management

### Authorization

Test:

```text
User A
  ↓
request resource của User B
  ↓
403 / reject
```

Kiểm tra IDOR/BOLA.

### API Security

- input validation
- rate limiting
- CORS
- security headers
- brute-force protection
- payload size limits
- SQL injection
- error information leakage

### WebSocket Security

Test:

```text
invalid token
expired token
missing token
unauthorized room
invalid message
movement spoofing
```

## Ghi chú

Không chỉ chạy scanner rồi tick "done".

Mỗi finding nên được ghi:

```text
Finding
→ Risk
→ Why it happens
→ Fix
→ Retest
```

## Definition of Done

- [ ] Trivy scan Docker image trên CI
- [ ] gosec chạy trên CI
- [ ] npm audit chạy trên CI
- [ ] OWASP ZAP scan định kỳ
- [ ] Rate limiting triển khai
- [ ] WebSocket security audited
- [ ] Mỗi finding có documented fix

---

# 10. Database Engineering `[ĐÃ HOÀN THÀNH]`

## Hiện trạng

Đã học và triển khai:

- **EXPLAIN ANALYZE**: phân tích query performance
- **Index**: tối ưu hóa index cho các query thật
- **Connection pool**: test với 10/50/100 connections
- **Transaction**: atomic operation (buy item → deduct coin → insert)
- **Race condition**: ngăn chặn double-spend với optimistic locking
- **Migration**: versioned SQL migrations trong `backend/internal/database/`

---

# 11. Release Engineering

## Mục tiêu

Học cách đưa phần mềm từ:

```text
code
```

đến:

```text
release
```

một cách có kiểm soát.

## Semantic Versioning

Học:

```text
MAJOR.MINOR.PATCH
```

Ví dụ:

```text
1.0.0
1.1.0
1.1.1
```

## Release flow

```text
main
 ↓
CI (all tests pass)
 ↓
git tag v1.1.0
 ↓
Build Docker image
 ↓
Push to container registry
 ↓
Deploy
 ↓
Smoke test
 ↓
Release confirmed / Rollback
```

## Changelog

Ví dụ:

```markdown
# v1.1.0

## Added
- E2E testing
- Prometheus metrics

## Changed
- Improved realtime validation

## Fixed
- Chat reconnect issue

## Security
- Added rate limiting
```

## Rollback

Phải trả lời được:

> "Deploy lỗi thì quay về version trước bằng cách nào?"

Đừng chỉ học deploy.

Hãy học:

```text
Deploy
Rollback
Verify
```

## Definition of Done

- [ ] Semantic versioning (git tags)
- [ ] Automated changelog generation
- [ ] Docker image build + push trên CI
- [ ] Release pipeline tự động khi tag version
- [ ] Smoke test sau deploy
- [ ] Rollback procedure documented

---

# 12. Chaos / Failure Testing

## Mục tiêu

Học cách hệ thống phản ứng khi dependency chết.

## Test 1 — PostgreSQL chết

```text
Start BigTown
↓
Stop PostgreSQL
↓
Call API
```

Ghi lại:

- error
- timeout
- retry
- recovery

## Test 2 — WebSocket disconnect

```text
Player
 ↓
disconnect
 ↓
reconnect
```

Kiểm tra state.

## Test 3 — Backend restart

```text
Backend
 ↓
kill
 ↓
restart
```

Kiểm tra:

- player reconnect
- authentication
- world state
- persisted data

## Test 4 — Network latency

Mô phỏng latency nếu môi trường cho phép.

## Ghi chú

Không cần bắt đầu bằng Chaos Monkey hay framework phức tạp.

Bắt đầu thủ công:

```text
kill process
stop container
disconnect network
restart service
```

Sau đó mới tự động hóa.

## Definition of Done

- [ ] PostgreSQL down → documented behavior + recovery
- [ ] WebSocket disconnect → documented behavior + recovery
- [ ] Backend restart → documented behavior + recovery
- [ ] Network latency → documented behavior
- [ ] Ít nhất 1 test tự động hóa trên CI

---

# 13. Documentation / ADR

## Mục tiêu

Không chỉ code biết tại sao hệ thống được thiết kế như vậy.

Con người sau này cũng phải biết.

## Hiện trạng

Đã có:

- `docs/Architecture.md` — kiến trúc tổng thể
- `backend/ARCHITECTURE_GUIDE.md` — backend architecture
- `docs/Storage-Design.md` — database design
- `docs/Realtime-*-*.md` — realtime architecture decisions
- `testing/loadtest/docs/` — load test guides
- ~18 tài liệu phase docs

Chưa có:

- `docs/adr/` thư mục ADR chính thức
- Documentation chuẩn hóa về testing, deployment, security
- Runbook

## Cấu trúc hoàn thiện

```text
docs/
├── architecture.md          ← đã có
├── development.md           ← cần viết (local setup, build, run)
├── testing.md               ← cần viết (cách chạy test các loại)
├── deployment.md            ← cần viết (cách deploy, rollback)
├── security.md              ← cần viết (security model, findings)
├── performance.md           ← cần viết (load test results, baselines)
├── runbook.md               ← cần viết (incident response)
└── adr/
    ├── 001-centrifuge-realtime.md
    ├── 002-server-authoritative-movement.md
    ├── 003-movement-not-persisted.md
    ├── 004-chat-persisted.md
    ├── 005-redis-usage.md
    └── 006-ci-cd-pipeline.md
```

## ADR Template

```markdown
# ADR-001: Realtime Architecture

## Status

Accepted

## Context

BigTown requires realtime player movement.

## Decision

Use Centrifuge as the realtime layer.

## Alternatives

- Raw WebSocket
- Socket.IO
- LiveKit

## Reason

...

## Trade-offs

...

## Consequences

...
```

## Các ADR nên viết

| ADR | Topic |
|-----|-------|
| 001 | Tại sao dùng Centrifuge? |
| 002 | Tại sao server-authoritative movement? |
| 003 | Tại sao movement không persist vào PostgreSQL? |
| 004 | Tại sao chat được persist? |
| 005 | Tại sao dùng Redis? |
| 006 | Tại sao chọn CI/CD pipeline này? |

## Definition of Done

- [ ] `docs/adr/` có ít nhất 6 ADR
- [ ] `docs/development.md` — hướng dẫn local dev
- [ ] `docs/testing.md` — hướng dẫn chạy unit/integration/e2e
- [ ] `docs/deployment.md` — hướng dẫn deploy + rollback
- [ ] `docs/security.md` — security model + findings
- [ ] `docs/performance.md` — load test baselines + results
- [ ] `docs/runbook.md` — incident response guide

---

# 14. Thứ tự triển khai

```text
Phase 0 ─ Containerization
│
├── Frontend Dockerfile (multi-stage)
├── Root docker-compose.yml (full stack)
├── .env.example
└── docker compose up → toàn bộ stack chạy locally
       │
       ▼
Phase 1 ─ CI/CD cơ bản
│
├── .github/workflows/ci.yml (lint + build)
├── .github/workflows/deploy.yml
└── Branch protection rules
       │
       ▼
Phase 2a ─ Unit Testing (Backend)
│
├── Security: JWT + password
├── Auth: usecase + repository
├── Character: usecase + repository
├── Chat: usecase + repository
├── Editor: usecase + repository
├── Realtime: usecase + room actor
├── Leaderboard: usecase
└── Chạy trên CI
       │
       ▼
Phase 2b ─ Unit Testing (Frontend)
│
├── Auth store (Pinia)
├── Game store (Pinia)
├── GameSocket URL parsing
└── Chạy trên CI (Vitest)
       │
       ▼
Phase 3 ─ Integration Testing
│
├── Docker Compose test infra
├── Auth + character + chat + realtime flows
└── Chạy trên CI
       │
       ▼
Phase 4 ─ E2E Testing
│
├── Playwright setup + 6 scenarios
├── Docker Compose full stack
└── Chạy trên CI (e2e.yml)
       │
       ▼
Phase 5 ─ Security Scanning
│
├── Tích hợp vào CI (security.yml)
├── Trivy + gosec + npm audit
├── OWASP ZAP định kỳ
└── Rate limiting + WebSocket security
       │
       ▼
Phase 6 ─ Release Engineering
│
├── Semantic versioning + git tags
├── Docker image build/push tự động
├── Changelog generation
└── Rollback procedure
       │
       ▼
Phase 7 ─ Chaos / Failure Testing
│
├── Manual chaos experiments
├── Documented recovery behavior
└── Tự động hóa 1 test trên CI
       │
       ▼
Phase 8 ─ Documentation / ADR
│
├── 6 ADRs
├── development.md, testing.md, deployment.md
├── security.md, performance.md, runbook.md
└── Chuẩn hóa toàn bộ docs
```

---

# 15. Checklist tổng

## Containerization `[DONE]`

- [x] `frontend/Dockerfile` — multi-stage build
- [x] `frontend/.dockerignore`
- [x] `frontend/nginx.conf` — SPA + API/WS proxy
- [x] `docker-compose.yml` — full stack
- [x] `.env.example`
- [x] `docker compose up` → full stack chạy locally
- [x] `schema.sql` + `seed.sql` — gộp tất cả migration

## CI/CD

- [x] `ci.yml` — lint + build backend + frontend
- [x] `deploy.yml` — trigger Render deploy hook
- [ ] Render deploy hook URL đã cấu hình
- [ ] Branch protection: PR phải pass CI mới merge

## Unit Testing

### Backend (Go)

- [ ] Security: JWT + password
- [ ] Auth usecase + repository
- [ ] Character usecase + repository
- [ ] Chat usecase + repository
- [ ] Editor usecase + repository
- [ ] Realtime usecase + room actor
- [ ] Leaderboard usecase
- [ ] Chạy trên CI (`go test ./...`)

### Frontend (Vitest)

- [ ] Auth store
- [ ] Game store
- [ ] GameSocket URL parsing
- [ ] Chạy trên CI (`npx vitest --run`)

## Integration Testing

- [ ] Docker Compose test infra
- [ ] Auth flow integration test
- [ ] Character flow integration test
- [ ] Chat flow integration test
- [ ] Realtime flow integration test
- [ ] Chạy trên CI

## E2E Testing

- [ ] Playwright setup
- [ ] Login test
- [ ] Character test
- [ ] Game entry test
- [ ] Chat test
- [ ] Realtime test
- [ ] Chạy trên CI

## Observability `[DONE]`

- [x] Structured logging
- [x] Prometheus
- [x] Grafana
- [x] HTTP metrics
- [x] Realtime metrics
- [x] Runtime metrics
- [ ] Alerts

## Load Testing `[DONE]`

- [x] Smoke test
- [x] Bootstrap load test
- [x] Chat load test
- [x] Movement load test
- [x] Placement load test
- [x] Baseline report
- [x] Bottleneck analysis
- [x] Optimization benchmark

## Security

- [x] Password hashing (bcrypt)
- [x] JWT auth
- [x] CORS
- [x] CSP headers
- [ ] Trivy container scan
- [ ] gosec SAST
- [ ] npm audit
- [ ] OWASP ZAP
- [ ] Rate limiting
- [ ] Input validation audit
- [ ] WebSocket security audit

## Database Engineering `[DONE]`

- [x] EXPLAIN ANALYZE
- [x] Index optimization
- [x] Connection pool
- [x] Transactions
- [x] Race condition
- [x] Migration strategy

## Release Engineering

- [ ] Semantic versioning
- [ ] Git tags
- [ ] Changelog
- [ ] Release pipeline
- [ ] Smoke test
- [ ] Rollback procedure

## Chaos / Failure Testing

- [ ] PostgreSQL down test
- [ ] WebSocket disconnect test
- [ ] Backend restart test
- [ ] Network latency test
- [ ] Recovery verification
- [ ] 1 automated test on CI

## Documentation

- [x] Architecture doc
- [ ] ADRs (6)
- [ ] Development guide
- [ ] Testing guide
- [ ] Deployment guide
- [ ] Security doc
- [ ] Performance doc
- [ ] Runbook

---

# 16. Ghi chú học tập quan trọng

## 1. Không học tất cả cùng lúc

Mỗi phase nên kết thúc bằng một artifact cụ thể.

```text
Containerization
→ docker compose up chạy full stack

CI/CD
→ GitHub Actions pipeline hoạt động

Unit Testing
→ test suite chạy pass

Integration Testing
→ Docker Compose test flow pass

E2E Testing
→ Playwright scenarios chạy pass

Security
→ security report

Release
→ v1.0.0 release

Chaos
→ failure test report

Documentation
→ ADRs + runbook
```

## 2. Luôn đo trước và sau

Đặc biệt với:

- performance
- database
- load test

## 3. Ghi lại những thứ đã học

Mỗi phase nên có:

```text
What I learned
What surprised me
What failed
Why it failed
How I fixed it
What trade-off exists
```

## 4. Không chạy theo tool

Ví dụ không cần:

```text
Kafka
RabbitMQ
NATS
Temporal
Kubernetes
Istio
```

chỉ để CV có nhiều keyword.

Chỉ dùng khi BigTown có vấn đề phù hợp để giải quyết.

## 5. Mỗi vấn đề nên tạo thành một engineering experiment

Ví dụ:

```text
Question:
Can BigTown support 500 realtime players?

Hypothesis:
Current architecture may bottleneck at X.

Experiment:
Run k6 with 100 / 250 / 500 players.

Measure:
P95
CPU
RAM
Goroutines
DB
WebSocket

Result:
...

Conclusion:
...
```

Đây mới là cách biến BigTown thành **engineering laboratory**.

---

# 17. Definition of Done cho toàn project

Có thể coi roadmap hoàn thành khi:

```text
[Containerization]
        ↓
Full stack runs in Docker
        ↓
[CI/CD]
        ↓
Every PR validated automatically
        ↓
[Unit + Integration + E2E Testing]
        ↓
Automated tests at all levels
        ↓
[Observability]
        ↓
System behavior visible
        ↓
[Load Testing]
        ↓
Known capacity + bottlenecks
        ↓
[Security]
        ↓
Known security risks + mitigations (automated)
        ↓
[Database]
        ↓
Known query/transaction behavior
        ↓
[Release]
        ↓
Repeatable deployment + rollback
        ↓
[Chaos / Failure Testing]
        ↓
Known recovery behavior
        ↓
[Documentation]
        ↓
Architecture is explainable
```

## Kết quả cuối

BigTown không cần thêm gameplay.

Thứ bạn đang xây dựng lúc này là:

> **Một realtime multiplayer application được sử dụng như một laboratory để thực hành Software Engineering, Backend Engineering, DevOps, Security, Performance và Production Engineering.**
