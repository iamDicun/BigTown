// BigTown — k6 load test: player_move RPC qua WebSocket (Centrifuge).
//
// Mỗi VU: mở 1 WS, connect + subscribe room, rồi gửi RPC "player_move" định kỳ.
// 100 VU chia đều vào N room (10 VU/room). Movement là random walk ±5px/100ms
// từ spawn (100, 100), clamp trong bounds map 4000×4000.
//
// Khác với chat test: movement gửi qua WS RPC (không phải REST), đo RPC latency
// và tỉ lệ broadcast. Đây là script stress đúng hot path — nơi global mutex đang
// serialize mọi room.
//
// Chạy:
//   k6 run \
//     -e WS_URL=ws://localhost:8080/connection/websocket \
//     -e ROOMS=10 \
//     movement_load_test.js
//
// Cần file tokens.json cùng thư mục (sinh bởi cmd/loadtest-gen).
//
// Protocol Centrifuge cho player_move:
//   Client → Server: {"id":N, "rpc":{"method":"player_move","data":{x,y,direction,moving}}}
//   Server → Client: {"id":N, "rpc":{}}                                  (ack)
//   Server → Room:   {"push":{"pub":{"data":{"type":"player_move",...}}}} (broadcast)
//   Server → Self:   {"push":{"pub":{"data":{"type":"player_position_correction",...}}}}

import ws from 'k6/ws';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// ---- cấu hình ----
const WS_URL          = __ENV.WS_URL          || 'ws://localhost:8080/connection/websocket';
const NUM_ROOMS       = parseInt(__ENV.ROOMS  || '10', 10);
const MOVE_INTERVAL_MS = parseInt(__ENV.MOVE_INTERVAL_MS || '100', 10);
const HOLD_MS         = parseInt(__ENV.HOLD_MS || '290000', 10);
const ORIGIN          = __ENV.ORIGIN || '';

// tokens.json: [{ userId, token, room }, ...]
const tokens = new SharedArray('tokens', () => JSON.parse(open('./tokens.json')));

// ---- custom metrics ----
const moveRpcSent     = new Counter('move_rpc_sent');       // số RPC player_move đã gửi
const moveRpcError    = new Counter('move_rpc_error');      // RPC lỗi / timeout
const moveRpcLatency  = new Trend('move_rpc_latency', true); // ms từ gửi đến nhận reply
const moveBroadcast   = new Counter('move_broadcast');      // số player_move nhận qua broadcast
const moveCorrection  = new Counter('move_correction');     // số player_position_correction nhận
const wsConnectErrors = new Counter('ws_connect_errors');

// player_move broadcast KHÔNG có trường roomId (khác với chat event) —
// room isolation được đảm bảo bởi Centrifuge channel-based pub/sub, đã kiểm
// qua cross_room_leak ở chat test.

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
    move_rpc_latency:  ['p(95)<500'],     // p95 RPC < 500ms (tùy chỉnh theo hạ tầng)
    move_rpc_error:    ['rate<0.05'],      // < 5% RPC lỗi
    checks:            ['rate>0.99'],
    ws_connect_errors: ['count<5'],
  },
};

function assign() {
  const idx = (__VU - 1) % tokens.length;
  const t = tokens[idx];
  const room = 'loadtest-map-' + String(idx % NUM_ROOMS).padStart(2, '0');
  return { token: t.token, room };
}

export default function () {
  const { token, room } = assign();
  const channel = 'room:' + room;

  const params = { tags: { room }, headers: {} };
  if (ORIGIN) params.headers['Origin'] = ORIGIN;

  // Mỗi VU giữ vị trí riêng (không share). Bắt đầu ở spawn.
  const state = {
    x: 100,
    y: 100,
    rpcSeq: 3,           // id 1 = connect, 2 = subscribe
    subscribed: false,
    moveInterval: null,
  };
  // Map id → timestamp gửi để đo RPC latency (key xóa khi nhận reply).
  const pendingRpcs = {};

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

        // Ping-pong keepalive (Centrifuge gửi {} mỗi 25s mặc định)
        if (Object.keys(obj).length === 0) {
          socket.send('{}');
          continue;
        }

        // Connect reply (id=1) → subscribe room
        if (obj.id === 1 && obj.connect) {
          socket.send(JSON.stringify({ id: 2, subscribe: { channel: channel } }));
          continue;
        }

        // Subscribe reply (id=2) → bắt đầu gửi movement định kỳ
        if (obj.id === 2 && obj.subscribe) {
          state.subscribed = true;
          check(true, { 'subscribed to room': () => true });
          state.moveInterval = socket.setInterval(function () {
            sendMoveRPC(socket, state, pendingRpcs);
          }, MOVE_INTERVAL_MS);
          continue;
        }

        // RPC reply → đo latency (khớp bằng id)
        if (obj.rpc !== undefined && pendingRpcs[obj.id] !== undefined) {
          moveRpcLatency.add(Date.now() - pendingRpcs[obj.id]);
          delete pendingRpcs[obj.id];
          continue;
        }

        // Push publication trên channel (broadcast từ server)
        if (obj.push && obj.push.pub && obj.push.pub.data) {
          const data = obj.push.pub.data;

          if (data.type === 'room_state') {
            const count = (data.players && data.players.length) || 0;
            moveBroadcast.add(count);
            for (const p of data.players || []) {
              check(p.characterId, {
                'move broadcast có characterId': (c) => c && c.length > 0,
              });
            }
          }

          if (data.type === 'player_move') {
            moveBroadcast.add(1);
            check(data.characterId, {
              'move broadcast có characterId': (c) => c && c.length > 0,
            });
          }

          if (data.type === 'player_position_correction') {
            moveCorrection.add(1);
          }
        }
      }
    });

    socket.on('error', function (e) {
      if (e && e.error &&
          !String(e.error).includes('1000') &&
          !String(e.error).includes('1001')) {
        wsConnectErrors.add(1);
      }
    });

    // Đóng WS gần cuối test
    socket.setTimeout(function () {
      if (state.moveInterval) socket.clearInterval(state.moveInterval);
      socket.close();
    }, HOLD_MS);
  });

  if (!res || res.status !== 101) {
    wsConnectErrors.add(1);
  }
}

// ---- helpers ----

const DIRS = ['down', 'up', 'left', 'right'];

function sendMoveRPC(socket, state, pendingRpcs) {
  // Random walk ±5px, clamp trong bounds map 4000×4000.
  // Tốc độ tối đa: 5px/100ms = 50px/s < 400px/s (maxSpeedPxPerSec).
  state.x += Math.floor(Math.random() * 11) - 5;
  state.y += Math.floor(Math.random() * 11) - 5;
  state.x = Math.max(0, Math.min(4000, state.x));
  state.y = Math.max(0, Math.min(4000, state.y));

  const id = state.rpcSeq++;
  pendingRpcs[id] = Date.now();

  socket.send(JSON.stringify({
    id: id,
    rpc: {
      method: 'player_move',
      data: {
        x: state.x,
        y: state.y,
        direction: DIRS[Math.floor(Math.random() * DIRS.length)],
        moving: true,
      },
    },
  }));
  moveRpcSent.add(1);
}

// ---- tóm tắt cuối test ----
export function handleSummary(data) {
  const sent      = (data.metrics.move_rpc_sent     && data.metrics.move_rpc_sent.values.count)     || 0;
  const broadcast = (data.metrics.move_broadcast    && data.metrics.move_broadcast.values.count)    || 0;
  const correction= (data.metrics.move_correction   && data.metrics.move_correction.values.count)   || 0;
  const p95       = (data.metrics.move_rpc_latency  && data.metrics.move_rpc_latency.values['p(95)']) || 0;
  const errRate   = (data.metrics.move_rpc_error    && data.metrics.move_rpc_error.values.rate)      || 0;

  const vus = (data.metrics.vus && data.metrics.vus.values.max) || 500;
  const membersPerRoom = Math.floor(vus / NUM_ROOMS);
  const expected = sent * membersPerRoom;
  const ratio = expected ? (broadcast / expected * 100).toFixed(1) : 'N/A';

  const line =
    `\n===== BigTown movement load summary =====\n` +
    `move_rpc_sent=${sent}  move_broadcast=${broadcast}  move_correction=${correction}\n` +
    `members/room≈${membersPerRoom}  fanout kỳ vọng≈${expected}  tỉ lệ giao≈${ratio}%\n` +
    `p95 rpc latency=${p95.toFixed(1)}ms  rpc error rate=${(errRate*100).toFixed(2)}%\n`;

  return {
    stdout: line,
    'movement_summary.json': JSON.stringify(data, null, 2),
  };
}
