# BigTown — Load Test Matrix Report

> 3 Phases × 3 Strategies = 9 test runs. Mỗi ô là một lần chạy độc lập.

---

## Test Matrix

| | Strategy 1: Local All | Strategy 2: Online (local k6→Render) | Strategy 3: Grafana Cloud |
|---|---|---|---|
| **Phase 1: Baseline** | ✅ Done | ✅ Done | ⬜ Pending |
| **Phase 2: Actor per-room** | ⬜ Pending | ⬜ Pending | ⬜ Pending |
| **Phase 3: Tick broadcast** | ⬜ Pending | ⬜ Pending | ⬜ Pending |

---

## Phase 1 — Baseline (Global Mutex `MemoryRoomStore`)

### Strategy 1: Local All (backend + DB + k6 cùng máy)

| Metric | Chat | Movement |
|---|---|---|
| Sent | 16,300 | 289,322 |
| Received | 163,000 | 2,304,907 |
| Delivery rate | 100.0% | 79.7% |
| Cross-room leak | 0 | N/A |
| Correction rate | N/A | 20.3% |
| **p95 latency** | N/A | **17.0ms** |
| RPC errors | 0% | 0.00% |

### Strategy 2: Online (k6 local → Render `bigtown-1.onrender.com`)

| Metric | Chat | Movement |
|---|---|---|
| Sent | 14,400 | 299,013 |
| Received | 143,166 | 1,407,427 |
| Delivery rate | 99.4% | 47.1% |
| Cross-room leak | 0 | N/A |
| Correction rate | N/A | 30.8% |
| **p95 latency** | N/A | **6,429ms** |
| RPC errors | 0% | 0.00% |

### So sánh Strategy 1 vs 2

| Chỉ số | S1 (local) | S2 (Render) | Chênh lệch |
|---|---|---|---|
| Chat delivery | 100% | 99.4% | ~0.6% |
| Move delivery | 79.7% | 47.1% | -32.6pp |
| p95 RPC latency | 17ms | 6,429ms | ×378 |
| RPC throughput | ~965/s | ~997/s | +3% |

Network latency là yếu tố chính kéo delivery rate và p95. Server-side throughput giữa 2 môi trường gần như tương đương (~965 vs ~997 RPC/s).

---

## Phase 2 — Actor Per-Room (pending)

Thay `MemoryRoomStore` (global mutex) bằng `ActorRoomStore` (mỗi room 1 goroutine, giao tiếp qua channel).

Kỳ vọng: p95 latency giảm, throughput tăng, CPU trải đều nhiều core.

---

## Phase 3 — Tick Broadcast (pending)

Gom movement broadcast 100ms/tick thay vì publish mỗi RPC. Giảm tải Centrifuge, tăng delivery rate.

---
## Cấu trúc thư mục

```
testing/loadtest/
├── scripts/                    # k6 scripts (dùng chung mọi phase)
│   ├── chat_load_test.js
│   └── movement_load_test.js
├── seed.sql
├── docs/
│   ├── Concurrency-Strategy-Map.md
│   └── LoadTest-Guide.md
├── phase-1-baseline/
│   ├── strategy-1-local/
│   │   └── results/
│   ├── strategy-2-online/
│   │   └── results/
│   └── strategy-3-grafana/
│       └── results/
├── phase-2-actor/
│   ├── strategy-1-local/
│   │   └── results/
│   ├── strategy-2-online/
│   │   └── results/
│   └── strategy-3-grafana/
│       └── results/
└── phase-3-tick-broadcast/
    ├── strategy-1-local/
    │   └── results/
    ├── strategy-2-online/
    │   └── results/
    └── strategy-3-grafana/
        └── results/
```
