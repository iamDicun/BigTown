// BigTown — k6 smoke test: kiểm tra nhanh kết nối trước khi chạy load test chính.
//
// Chạy:
//   k6 run smoke_test.js -e WS_URL=wss://bigtown-1.onrender.com/connection/websocket -e BASE_URL=https://bigtown-1.onrender.com
//
// Pass hết checks = hệ thống sẵn sàng. Fail = token hết hạn / sai secret / backend lỗi.

import ws from 'k6/ws';
import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

const WS_URL   = __ENV.WS_URL   || 'ws://localhost:8080/connection/websocket';
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ORIGIN   = __ENV.ORIGIN || '';

const tokens = new SharedArray('tokens', () => JSON.parse(open('./tokens.json')));

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    checks: ['rate==1'],
  },
};

export default function () {
  const t = tokens[0];
  const room = 'loadtest-map-00';
  const channel = 'room:' + room;

  const params = { headers: {} };
  if (ORIGIN) params.headers['Origin'] = ORIGIN;

  // ---- 1. Bootstrap (HTTP) ----
  console.log('1/4  Bootstrap...');
  const bRes = http.get(`${BASE_URL}/api/realtime/bootstrap?map_code=${room}`, {
    headers: { Authorization: `Bearer ${t.token}` },
  });
  let bOk = false;
  try {
    const d = bRes.json('data') || {};
    bOk = bRes.status === 200 && d.spawn_x !== undefined && d.tick_rate_ms !== undefined;
  } catch (e) { bOk = false; }
  check(bRes, { 'bootstrap 200 + payload hop le': () => bOk });
  console.log(`     bootstrap: ${bRes.status} (${bRes.timings.duration}ms)`);

  // ---- 2. WebSocket connect + subscribe ----
  console.log('2/4  WebSocket connect...');
  let wsOk = false;
  let subOk = false;

  const res = ws.connect(WS_URL, params, function (socket) {
    socket.on('open', function () {
      wsOk = true;
      console.log('     WS connected');
      socket.send(JSON.stringify({ id: 1, connect: { token: t.token } }));
    });

    socket.on('message', function (raw) {
      const parts = String(raw).split('\n');
      for (const p of parts) {
        if (!p) continue;
        let obj;
        try { obj = JSON.parse(p); } catch (e) { continue; }

        if (Object.keys(obj).length === 0) { socket.send('{}'); continue; }

        if (obj.id === 1 && obj.connect) {
          console.log('     Centrifuge connected');
          socket.send(JSON.stringify({ id: 2, subscribe: { channel: channel } }));
          continue;
        }

        if (obj.id === 2 && obj.subscribe) {
          subOk = true;
          console.log('     subscribed to ' + channel);

          // ---- 3. Chat POST ----
          console.log('3/4  POST chat...');
          const cRes = http.post(`${BASE_URL}/api/rooms/${room}/chat/messages`,
            JSON.stringify({ message: 'smoke test' }),
            { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${t.token}` } },
          );
          console.log(`     chat POST: ${cRes.status} (${cRes.timings.duration}ms)`);
          check(cRes, { 'chat POST 201': (r) => r.status === 201 });

          // ---- 4. Move RPC ----
          console.log('4/4  Player move RPC...');
          socket.send(JSON.stringify({
            id: 3,
            rpc: { method: 'player_move', data: { x: 100, y: 100, direction: 'down', moving: true } },
          }));
        }

        if (obj.id === 3 && obj.rpc !== undefined) {
          console.log('     move RPC ack received');
          socket.close();
        }
      }
    });

    socket.on('error', function (e) {
      console.log(`     WS error: ${JSON.stringify(e)}`);
    });

    socket.setTimeout(function () {
      socket.close();
    }, 10000);
  });

  check(true, { 'WS upgrade 101': () => res && res.status === 101 });
  check(true, { 'WS opened': () => wsOk });
  check(true, { 'subscribed to room': () => subOk });

  sleep(0.5);
}

export function handleSummary(data) {
  const failed = data.metrics.checks.values.fails || 0;
  const passed = data.metrics.checks.values.passes || 0;
  const allPass = failed === 0;

  const line =
    `\n===== BigTown SMOKE TEST =====\n` +
    `checks: ${passed} pass, ${failed} fail\n` +
    (allPass ? '=> PASS: san sang chay load test chinh\n' : '=> FAIL: kiem tra token/secret/backend\n');

  return { stdout: line };
}
