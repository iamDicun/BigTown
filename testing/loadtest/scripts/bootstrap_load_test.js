// BigTown — k6 load test: GET /api/realtime/bootstrap (spike/ramp).
//
// Bootstrap là request MỌI client gọi khi vào game và mỗi lần warp/đổi map.
// Kể từ khi thêm NPC spawn, payload nặng hơn (kèm npc_spawns) và server phải
// đọc maps + npc spawns. Đây là điểm dễ "thundering herd" khi nhiều người login
// cùng lúc (vd sau khi deploy, hoặc giờ cao điểm).
//
// Khác 2 script kia: đây thuần HTTP (không giữ WS), dùng mô hình open-model
// (ramping-arrival-rate) để ÉP một tốc độ request cố định và tìm điểm gãy.
//
// Chạy trên Grafana Cloud (strategy 3):
//   k6 cloud run bootstrap_load_test.js \
//     -e BASE_URL=https://<host> -e ROOMS=10 \
//     -e PEAK_RPS=200
//
// Bootstrap nằm trong nhóm /api (có AuthMiddleware) => cần Bearer token hợp lệ.

import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

const BASE_URL  = __ENV.BASE_URL  || 'http://localhost:8080';
const NUM_ROOMS = parseInt(__ENV.ROOMS || '10', 10);
const PEAK_RPS  = parseInt(__ENV.PEAK_RPS || '200', 10);

const tokens = new SharedArray('tokens', () => JSON.parse(open('./tokens.json')));

const bootOk     = new Counter('bootstrap_ok');       // 200 + có dữ liệu hợp lệ
const bootBad    = new Counter('bootstrap_bad');      // status != 200 hoặc thiếu field
const bootMs     = new Trend('bootstrap_ms', true);   // latency riêng bootstrap
const npcSpawnCnt= new Trend('bootstrap_npc_count');  // số npc_spawns trả về (theo dõi payload)

export const options = {
  scenarios: {
    // Tăng dần tốc độ request tới PEAK_RPS rồi giữ, rồi giảm — soi điểm gãy.
    ramp_arrival: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 100,
      stages: [
        { target: 50,       duration: '30s' },
        { target: PEAK_RPS, duration: '1m'  },
        { target: PEAK_RPS, duration: '2m'  }, // giữ đỉnh
        { target: 0,        duration: '30s' },
      ],
    },
  },
  ext: {
    loadimpact: {
      distribution: {
        singaporeZone: { loadZone: 'amazon:sg:singapore', percent: 100 },
      },
    },
  },
  thresholds: {
    bootstrap_ms:    ['p(95)<600', 'p(99)<1200'],
    http_req_failed: ['rate<0.01'],
    bootstrap_bad:   ['count<10'],
    checks:          ['rate>0.99'],
  },
};

export default function () {
  // Chọn token + room ngẫu nhiên (mô phỏng nhiều user vào nhiều map).
  const t = tokens[Math.floor(Math.random() * tokens.length)];
  const room = 'loadtest-map-' + String(Math.floor(Math.random() * NUM_ROOMS)).padStart(2, '0');

  const t0 = Date.now();
  const res = http.get(`${BASE_URL}/api/realtime/bootstrap?map_code=${room}`, {
    headers: { Authorization: `Bearer ${t.token}` },
    tags: { name: 'GET /realtime/bootstrap' },
  });
  bootMs.add(Date.now() - t0);

  let ok = res.status === 200;
  if (ok) {
    // Xác nhận payload hợp lệ: có spawn_x và tick_rate_ms.
    let hasCore = false, npc = 0;
    try {
      const d = res.json('data') || {};
      hasCore = (d.spawn_x !== undefined) && (d.tick_rate_ms !== undefined);
      npc = Array.isArray(d.npc_spawns) ? d.npc_spawns.length : 0;
    } catch (e) { hasCore = false; }
    ok = hasCore;
    if (ok) npcSpawnCnt.add(npc);
  }

  if (ok) bootOk.add(1); else bootBad.add(1);
  check(res, {
    'bootstrap 200 + payload hợp lệ': () => ok,
  });
}

export function handleSummary(data) {
  const ok  = (data.metrics.bootstrap_ok && data.metrics.bootstrap_ok.values.count) || 0;
  const bad = (data.metrics.bootstrap_bad && data.metrics.bootstrap_bad.values.count) || 0;
  const p95 = (data.metrics.bootstrap_ms && data.metrics.bootstrap_ms.values['p(95)']) || 0;
  const p99 = (data.metrics.bootstrap_ms && data.metrics.bootstrap_ms.values['p(99)']) || 0;
  const npc = (data.metrics.bootstrap_npc_count && data.metrics.bootstrap_npc_count.values.avg) || 0;

  const line =
    `\n===== BigTown bootstrap load summary =====\n` +
    `bootstrap_ok=${ok}  bootstrap_bad=${bad}\n` +
    `p95=${p95.toFixed(1)}ms  p99=${p99.toFixed(1)}ms  npc_spawns trung bình=${npc.toFixed(1)}\n`;

  return { stdout: line, 'bootstrap_summary.json': JSON.stringify(data, null, 2) };
}
