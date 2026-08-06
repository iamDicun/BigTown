// BigTown — k6 load test: đặt vật phẩm (placement) + broadcast decoration_placed.
//
// Đây là hot path GHI mới kể từ khi thêm editor/inventory: mỗi lần đặt sẽ đi qua
// room actor (serialize theo map) -> trừ coin -> reply HTTP -> broadcast
// "decoration_placed" xuống room channel -> write-behind persist DB.
//
// Mỗi VU: mở 1 WS (để NHẬN broadcast), rồi POST /api/editor/place định kỳ.
// Đo: latency REST place, latency place->broadcast (khớp theo placement.id),
// tỉ lệ fanout (mỗi lần đặt tới ~mọi thành viên cùng room).
//
// Chạy trên Grafana Cloud (strategy 3):
//   k6 cloud run placement_load_test.js \
//     -e WS_URL=wss://<host>/connection/websocket \
//     -e BASE_URL=https://<host> \
//     -e ROOMS=10 -e ORIGIN=https://big-town.vercel.app
//
// Ràng buộc backend (đã đối chiếu code):
//   - Toạ độ phải snap theo tile_size (=16) và nằm trong map (4000x4000).
//   - UNIQUE (map_id, x, y): 2 lần đặt trùng ô -> lỗi "occupied". Mỗi VU đặt ở
//     cột riêng để không đụng nhau (giả định <= ~120 VU/room-space).
//   - Character phải có coins >= giá item. Seed loadtest mặc định coins=0 =>
//     PHẢI nạp coin trước (xem guide, mục "Nạp coin").
//   - decoration_placed KHÔNG mang roomId; cách ly room do channel đảm bảo (đã
//     được chat test chứng minh qua cross_room_leak), nên script này không kiểm
//     lại leak mà tập trung throughput/latency của đường ghi+broadcast.

import ws from 'k6/ws';
import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// ---- cấu hình ----
const WS_URL         = __ENV.WS_URL         || 'ws://localhost:8080/connection/websocket';
const BASE_URL       = __ENV.BASE_URL       || 'http://localhost:8080';
const NUM_ROOMS      = parseInt(__ENV.ROOMS || '10', 10);
const HOLD_MS        = parseInt(__ENV.HOLD_MS || '290000', 10);
const PLACE_EVERY_MS = parseInt(__ENV.PLACE_EVERY_MS || '1000', 10);
const TILE           = parseInt(__ENV.TILE || '16', 10);
const ORIGIN         = __ENV.ORIGIN || '';

const tokens = new SharedArray('tokens', () => JSON.parse(open('./tokens.json')));

// ---- custom metrics ----
const placeSent      = new Counter('place_sent');        // POST place 200
const placeErr       = new Counter('place_error');       // POST place != 200 (trừ occupied)
const placeOccupied  = new Counter('place_occupied');    // ô đã bị chiếm (không tính lỗi)
const placeRestMs    = new Trend('place_rest_ms', true); // latency REST place
const placeDeliverMs = new Trend('place_deliver_ms', true); // place -> nhận broadcast
const placeBroadcast = new Counter('place_broadcast');   // số decoration_placed nhận
const wsConnectErrors= new Counter('ws_connect_errors');

export const options = {
  scenarios: {
    soak_100vu: {
      executor: 'constant-vus',
      vus: 100,
      duration: '5m',
      gracefulStop: '30s',
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
    place_rest_ms:     ['p(95)<800'],
    place_deliver_ms:  ['p(95)<1500'],
    place_error:       ['count<10'],
    checks:            ['rate>0.99'],
    ws_connect_errors: ['count<5'],
  },
};

function assign() {
  const idx = (__VU - 1) % tokens.length;
  const room = 'loadtest-map-' + String(idx % NUM_ROOMS).padStart(2, '0');
  return { token: tokens[idx].token, room };
}

// setup() chạy 1 lần: lấy catalog item, chọn item rẻ nhất để tiết kiệm coin.
export function setup() {
  const room = 'loadtest-map-00';
  const t = tokens[0].token;
  const params = {
    headers: { Authorization: `Bearer ${t}` },
    tags: { name: 'GET /editor (setup)' },
  };
  const res = http.get(`${BASE_URL}/api/editor?map_code=${room}`, params);
  let itemId = null;
  try {
    const items = (res.json('data.items')) || [];
    let cheapest = null;
    for (const it of items) {
      if (cheapest === null || it.price < cheapest.price) cheapest = it;
    }
    if (cheapest) itemId = cheapest.id;
  } catch (e) { /* ignore */ }

  if (!itemId) {
    throw new Error('setup: không lấy được item_id từ /api/editor — kiểm tra seed items + token.');
  }
  return { itemId };
}

export default function (data) {
  const { token, room } = assign();
  const channel = 'room:' + room;
  const itemId = data.itemId;

  const params = { tags: { room }, headers: {} };
  if (ORIGIN) params.headers['Origin'] = ORIGIN;

  const state = { seq: 0, timer: null };
  const pending = {}; // placementId -> ts gửi POST (đo delivery)

  const res = ws.connect(WS_URL, params, function (socket) {
    socket.on('open', function () {
      socket.send(JSON.stringify({ id: 1, connect: { token: token } }));
    });

    socket.on('message', function (raw) {
      const parts = String(raw).split('\n');
      for (const p of parts) {
        if (!p) continue;
        let obj;
        try { obj = JSON.parse(p); } catch (e) { continue; }

        if (Object.keys(obj).length === 0) { socket.send('{}'); continue; }

        if (obj.id === 1 && obj.connect) {
          socket.send(JSON.stringify({ id: 2, subscribe: { channel: channel } }));
          continue;
        }

        if (obj.id === 2 && obj.subscribe) {
          check(true, { 'subscribed to room': () => true });
          state.timer = socket.setInterval(function () {
            placeItem(room, token, itemId, state, pending);
          }, PLACE_EVERY_MS);
          continue;
        }

        if (obj.push && obj.push.pub && obj.push.pub.data) {
          const d = obj.push.pub.data;
          if (d.type === 'decoration_placed' && d.placement) {
            placeBroadcast.add(1);
            const id = d.placement.id;
            check(id, { 'decoration_placed có id': (v) => v && v.length > 0 });
            if (pending[id] !== undefined) {
              placeDeliverMs.add(Date.now() - pending[id]);
              delete pending[id];
            }
          }
        }
      }
    });

    socket.on('error', function (e) {
      if (e && e.error && !String(e.error).includes('1000') && !String(e.error).includes('1001')) {
        wsConnectErrors.add(1);
      }
    });

    socket.setTimeout(function () {
      if (state.timer) socket.clearInterval(state.timer);
      socket.close();
    }, HOLD_MS);
  });

  if (!res || res.status !== 101) wsConnectErrors.add(1);
}

// ---- helper: POST /api/editor/place ở ô riêng của VU, snap theo TILE ----
function placeItem(room, token, itemId, state, pending) {
  const seq = state.seq++;
  // Mỗi VU 1 dải cột riêng (bước 2 cột), đi xuống theo hàng; wrap sang cột kế khi hết hàng.
  const col = 10 + (__VU - 1) * 2 + Math.floor(seq / 230);
  const row = 10 + (seq % 230);
  if (col >= 249 || row >= 249) return; // hết không gian an toàn -> bỏ qua

  const x = col * TILE;
  const y = row * TILE;

  const payload = JSON.stringify({ item_id: itemId, map_code: room, x: x, y: y });
  const t0 = Date.now();
  const res = http.post(`${BASE_URL}/api/editor/place`, payload, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    tags: { room, name: 'POST /editor/place' },
  });

  placeRestMs.add(Date.now() - t0);

  if (res.status === 200) {
    placeSent.add(1);
    let id = null;
    try { id = res.json('data.placement.id'); } catch (e) { /* ignore */ }
    if (id) pending[id] = t0;
    check(res, { 'place 200': () => true });
  } else {
    // Ô bị chiếm là bình thường trong stress, không tính là lỗi hạ tầng.
    const body = String(res.body || '');
    if (res.status === 409 || body.indexOf('occupied') >= 0 || body.indexOf('chiếm') >= 0) {
      placeOccupied.add(1);
    } else {
      placeErr.add(1);
      check(res, { 'place 200': () => false });
    }
  }
}

// ---- tóm tắt ----
export function handleSummary(data) {
  const sent = (data.metrics.place_sent && data.metrics.place_sent.values.count) || 0;
  const bc   = (data.metrics.place_broadcast && data.metrics.place_broadcast.values.count) || 0;
  const occ  = (data.metrics.place_occupied && data.metrics.place_occupied.values.count) || 0;
  const err  = (data.metrics.place_error && data.metrics.place_error.values.count) || 0;
  const p95r = (data.metrics.place_rest_ms && data.metrics.place_rest_ms.values['p(95)']) || 0;
  const p95d = (data.metrics.place_deliver_ms && data.metrics.place_deliver_ms.values['p(95)']) || 0;

  const vus = (data.metrics.vus && data.metrics.vus.values.max) || 100;
  const membersPerRoom = Math.max(1, Math.floor(vus / NUM_ROOMS));
  const expected = sent * membersPerRoom;
  const ratio = expected ? (bc / expected * 100).toFixed(1) : 'N/A';

  const line =
    `\n===== BigTown placement load summary =====\n` +
    `place_sent=${sent}  place_broadcast=${bc}  occupied=${occ}  error=${err}\n` +
    `members/room≈${membersPerRoom}  fanout kỳ vọng≈${expected}  tỉ lệ giao≈${ratio}%\n` +
    `p95 REST place=${p95r.toFixed(1)}ms  p95 delivery=${p95d.toFixed(1)}ms\n`;

  return { stdout: line, 'placement_summary.json': JSON.stringify(data, null, 2) };
}
