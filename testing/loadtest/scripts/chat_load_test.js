// BigTown — k6 load test: chat/room realtime.
//
// Mỗi VU: mở 1 WebSocket (Centrifuge) để NHẬN, và gọi REST định kỳ để GỬI chat.
// 100 VU chia đều vào N room. Cùng room phải nhận được tin của nhau; khác room
// KHÔNG được nhận (kiểm bằng metric cross_room_leak phải = 0).
//
// Chạy:
//   k6 run \
//     -e WS_URL=ws://localhost:8080/connection/websocket \
//     -e BASE_URL=http://localhost:8080 \
//     -e ROOMS=10 \
//     chat_load_test.js
//
// (deploy: WS_URL=wss://.../connection/websocket BASE_URL=https://...)
//
// Cần file tokens.json cùng thư mục (sinh bởi cmd/loadtest-gen).

import ws from 'k6/ws';
import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// ---- cấu hình ----
const WS_URL   = __ENV.WS_URL   || 'ws://localhost:8080/connection/websocket';
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const NUM_ROOMS = parseInt(__ENV.ROOMS || '10', 10);
const HOLD_MS   = parseInt(__ENV.HOLD_MS || '290000', 10); // giữ WS ~4m50s
const SEND_EVERY_MS = parseInt(__ENV.SEND_EVERY_MS || '2000', 10);
const ORIGIN = __ENV.ORIGIN || ''; // set nếu backend CheckOrigin chặn (allowed origin)

// tokens.json: [{ userId, token, room }, ...] — nạp 1 lần, chia sẻ giữa mọi VU.
const tokens = new SharedArray('tokens', () => JSON.parse(open('./tokens.json')));

// ---- custom metrics ----
const chatSent        = new Counter('chat_sent');        // số POST chat 201
const chatReceived    = new Counter('chat_received');    // số player_chat nhận qua WS
const crossRoomLeak   = new Counter('cross_room_leak');  // PHẢI = 0
const deliveryMs      = new Trend('chat_delivery_ms', true); // độ trễ gửi->nhận
const wsConnectErrors = new Counter('ws_connect_errors');

export const options = {
  scenarios: {
    soak_100vu: {
      executor: 'constant-vus',
      vus: 100,
      duration: '5m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    cross_room_leak:  ['count==0'],       // tuyệt đối không rò tin sang room khác
    chat_delivery_ms: ['p(95)<1000'],     // p95 độ trễ nhận < 1s (tùy chỉnh)
    http_req_failed:  ['rate<0.01'],      // REST chat lỗi < 1%
    checks:           ['rate>0.99'],
    ws_connect_errors:['count<5'],
  },
};

// Mỗi VU cố định 1 token + 1 room, suy ra từ chỉ số VU (ổn định qua các iteration).
function assign() {
  const idx = (__VU - 1) % tokens.length;
  const t = tokens[idx];
  // room tính lại độc lập để chắc chắn khớp phân bố, không phụ thuộc field 'room'
  const room = 'loadtest-map-' + String(idx % NUM_ROOMS).padStart(2, '0');
  return { token: t.token, room };
}

export default function () {
  const { token, room } = assign();
  const channel = 'room:' + room;

  // jitter nhỏ để 100 VU không connect cùng một thời điểm (thundering herd)
  const params = { tags: { room }, headers: {} };
  if (ORIGIN) params.headers['Origin'] = ORIGIN;

  const res = ws.connect(WS_URL, params, function (socket) {
    let subscribed = false;
    let sendTimer = null;
    let seq = 0;

    socket.on('open', function () {
      // Centrifuge protocol v2 (JSON): connect kèm token.
      socket.send(JSON.stringify({ id: 1, connect: { token: token } }));
    });

    socket.on('message', function (raw) {
      // Có thể là batch nhiều frame ngăn cách bằng '\n'.
      const parts = String(raw).split('\n');
      for (const p of parts) {
        if (!p) continue;
        let obj;
        try { obj = JSON.parse(p); } catch (e) { continue; }

        // Ping của server = object rỗng {} -> trả pong rỗng để giữ kết nối.
        if (Object.keys(obj).length === 0) {
          socket.send('{}');
          continue;
        }

        // Reply cho connect (id=1) -> subscribe vào room channel.
        if (obj.id === 1 && obj.connect) {
          socket.send(JSON.stringify({ id: 2, subscribe: { channel: channel } }));
          continue;
        }

        // Reply cho subscribe (id=2) -> bắt đầu gửi chat định kỳ.
        if (obj.id === 2 && obj.subscribe) {
          subscribed = true;
          check(true, { 'subscribed to room': () => true });
          sendTimer = socket.setInterval(function () {
            sendChatViaREST(room, seq++);
          }, SEND_EVERY_MS);
          continue;
        }

        // Push publication trên channel: obj.push.pub.data = event của ta.
        if (obj.push && obj.push.pub && obj.push.pub.data) {
          const data = obj.push.pub.data; // JSON protocol nhúng thẳng object
          if (data.type !== 'player_chat') continue; // bỏ qua join/leave/move

          chatReceived.add(1);

          // Kiểm CÁCH LY room: server đóng dấu roomId; phải trùng room của VU này.
          const isolated = data.roomId === room;
          if (!isolated) crossRoomLeak.add(1);
          check(isolated, { 'chat chỉ đến từ đúng room': () => isolated });

          // Độ trễ: message có dạng "LT|<room>|<vu>|<sentMs>|<seq>"
          const m = String(data.message || '').split('|');
          if (m[0] === 'LT' && m[3]) {
            const lat = Date.now() - Number(m[3]);
            if (lat >= 0 && lat < 60000) deliveryMs.add(lat);
          }
        }
      }
    });

    socket.on('error', function (e) {
      // 1000/1001 là đóng bình thường; còn lại tính là lỗi kết nối.
      if (e && e.error && !String(e.error).includes('1000') && !String(e.error).includes('1001')) {
        ws_connect_errors_add();
      }
    });

    // Đóng WS gần cuối test để iteration kết thúc gọn.
    socket.setTimeout(function () {
      if (sendTimer) socket.clearInterval(sendTimer);
      socket.close();
    }, HOLD_MS);
  });

  if (!res || res.status !== 101) {
    wsConnectErrors.add(1);
  }
}

function ws_connect_errors_add() { wsConnectErrors.add(1); }

// Gửi chat qua REST; server sẽ broadcast player_chat xuống room channel.
function sendChatViaREST(room, seq) {
  const url = `${BASE_URL}/api/rooms/${room}/chat/messages`;
  // Không có token trong header REST -> AuthMiddleware chặn. Dùng lại token của VU.
  const { token } = assign();
  const payload = JSON.stringify({
    message: `LT|${room}|${__VU}|${Date.now()}|${seq}`,
  });
  const res = http.post(url, payload, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    tags: { room, name: 'POST /rooms/:room/chat' }, // gộp URL để metric không nổ cardinality
  });
  const ok = res.status === 201;
  if (ok) chatSent.add(1);
  check(res, { 'chat POST 201': (r) => r.status === 201 });
}

// Tóm tắt cuối test: in tỉ lệ giao/nhận để đọc nhanh.
export function handleSummary(data) {
  const sent = (data.metrics.chat_sent && data.metrics.chat_sent.values.count) || 0;
  const recv = (data.metrics.chat_received && data.metrics.chat_received.values.count) || 0;
  const leak = (data.metrics.cross_room_leak && data.metrics.cross_room_leak.values.count) || 0;
  const membersPerRoom = Math.floor(100 / NUM_ROOMS);
  const expected = sent * membersPerRoom; // mỗi tin đến ~ mọi thành viên (kể cả người gửi)
  const ratio = expected ? (recv / expected) : 0;

  const line =
    `\n===== BigTown chat load summary =====\n` +
    `chat_sent=${sent}  chat_received=${recv}\n` +
    `members/room≈${membersPerRoom}  fanout kỳ vọng≈${expected}  tỉ lệ giao≈${(ratio * 100).toFixed(1)}%\n` +
    `cross_room_leak=${leak} (phải = 0)\n`;

  return {
    stdout: line,
    'chat_summary.json': JSON.stringify(data, null, 2),
  };
}
