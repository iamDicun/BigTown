# BigTown — Proximity Voice Chat: Thiết kế & Hướng tích hợp

> Tính năng: một player phát giọng, **mọi** player đủ gần đều nghe thấy (không phải call 1‑1 hay group cố định). Nghe được nhiều người cùng lúc. Có bật/tắt mic. Người tắt mic vẫn **nghe** được người khác.
> Bám kiến trúc hiện có: Centrifuge (`room:<mapCode>` + `personal:<userID>`), mọi lệnh qua RPC, vị trí player authoritative trong `GameRoom` (RAM).

---

## 0. Quyết định lớn nhất (đọc trước)

**Âm thanh KHÔNG đi qua Centrifuge/WebSocket.** WebSocket không có jitter buffer, không kiểm soát nghẽn, không PLC (packet loss concealment) — nhét Opus frame qua đó nghe sẽ giật và trễ. Media đi trên **WebRTC**; Centrifuge chỉ làm **signaling** (trao đổi SDP/ICE) và **điều phối proximity** (ai gần ai). Đây là ranh giới then chốt của toàn bộ thiết kế.

Hai topology cho WebRTC:

| | Mesh P2P (khuyến nghị làm trước) | SFU (LiveKit/mediasoup — nâng cấp sau) |
|---|---|---|
| Cách hoạt động | Mỗi cặp player gần nhau có 1 `RTCPeerConnection` trực tiếp | Mỗi player publish 1 track lên server, server forward cho người gần |
| Hạ tầng thêm | Chỉ cần STUN + **TURN** (coturn). Không có media server | Cả một media server + ops + scaling |
| Chi phí băng thông client | Mỗi người **upload N lần** (N = số peer gần) | Upload **1 lần** duy nhất |
| Trần chịu tải | Tốt tới ~6–8 người cùng nói/nghe quanh bạn | Đám đông dày (quảng trường, event) |
| Proximity | Client tự attenuate theo khoảng cách | Server quản subscribe + volume theo khoảng cách |

**Vì sao mesh trước:** proximity **tự giới hạn** fan‑out (bạn chỉ kết nối với người trong bán kính R), khớp gần như miễn phí với Centrifuge signaling sẵn có, không thêm service phải vận hành. Chuyển sang SFU khi có cảnh **> ~8 người vừa nói vừa nghe chen nhau một chỗ** (lúc đó upload N lần của mesh bắt đầu nghẽn). Ngưỡng chuyển ghi rõ ở mục 9.

> Nếu biết chắc sẽ có quảng trường đông ngay từ đầu → cân nhắc LiveKit luôn (nó có sẵn ví dụ spatial audio, SDK Go + JS). Kiểm tra tài liệu LiveKit hiện tại trước khi chọn version/API cụ thể. Phần còn lại của doc này tập trung vào mesh vì nó tái dùng đúng hạ tầng bạn đang có.

---

## 1. Ai quyết định "đủ gần" — Server, không phải client

Bạn **đã** có sẵn thứ đắt nhất: `GameRoom.Players[characterID]` giữ `X, Y, ClientID, UserID` của mọi player trong map, cập nhật mỗi lần `player_move`. Đây chính là **proximity authority**. Client tuyệt đối không được tự khai "tôi gần người kia" (spam/nghe lén) — server chốt.

### Thuật toán neighbor set

Với mỗi player, tập "nghe được" = những player khác trong bán kính `R`. Recompute mỗi khi có move (đã throttle sẵn ở movement pipeline). `handlePlayerMove` hiện không broadcast tức thời (có tick 100ms); recompute theo mỗi move (đã throttle 10/s) là ổn, không cần thêm timer riêng.

Ba chi tiết bắt buộc để không giật/không thrash:

1. **Spatial hash** thay vì so sánh O(n²): bucket theo ô lưới cạnh ≈ `R` (`cell = (x/R, y/R)`). Chỉ so player trong 9 ô lân cận → gần O(1) mỗi move. `map_actor` của bạn đã có khái niệm lưới occupancy, dùng lại tư duy đó cho một grid riêng của voice.
2. **Hysteresis 2 bán kính**: kết nối khi khoảng cách < `R_in`, chỉ ngắt khi > `R_out` (với `R_out > R_in`, ví dụ 256px / 320px). Nếu chỉ 1 ngưỡng, đứng đúng ranh giới sẽ connect/disconnect liên tục (nghe "ọc ọc"). Để ra được diff `add`/`remove`, cần nhớ audible‑set trước đó của mỗi player — state này sống trong room actor (per‑room), cập nhật dưới lock của actor.
3. **Cap fan‑out**: nếu quanh bạn > `MAX_PEERS` (vd 8), chỉ giữ `MAX_PEERS` người **gần nhất**. Chặn mesh nổ ở chỗ đông.

Khi tập thay đổi → server bắn cho **riêng** player đó (qua `personal:<userID>`) một diff:

```jsonc
// type: "voice_peers"
{ "type": "voice_peers",
  "add":    [ { "characterId": "B", "userId": "ub" } ],
  "remove": [ "C" ] }
```

`add` → client dựng PeerConnection. `remove` → client đóng PeerConnection + gỡ audio node.

---

## 2. Signaling — thêm vài RPC, tái dùng personal channel

Client không được `publish` (bạn đã chặn ở `OnPublish`). Nên signaling cũng đi qua **RPC** như `player_move`, và server **relay** gói signaling tới đúng người nhận qua `personal:<targetUserID>`.

Luồng (deterministic để tránh "glare" — hai bên cùng offer):

```
Server → A (personal): voice_peers.add = [B]
A: nếu A.characterId < B.characterId  → A là bên tạo offer
A → server (RPC "voice_signal"): { to: "B", kind: "offer", sdp }
Server: kiểm B có THỰC SỰ nằm trong audible-set của A không → nếu ok, relay
Server → B (personal): { type:"voice_signal", from:"A", kind:"offer", sdp }
B → server (RPC "voice_signal"): { to:"A", kind:"answer", sdp }
Server → A (personal): { ... kind:"answer" ... }
(hai bên trao đổi tiếp kind:"ice" cho từng ICE candidate)
```

Hai luật an toàn ở server khi relay:
- **Chỉ relay giữa hai người đang trong audible‑set của nhau.** Nếu không, ai đó có thể dùng signaling để gọi tới user bất kỳ. Server có sẵn dữ kiện để kiểm.
- **Không nhận `from` từ client** — server tự gắn từ UserID đã xác thực (giống hệt cách `player_move` không tin `characterId` client gửi).

---

## 3. Backend — điểm hook cụ thể

### 3.1 DTO — `realtime/transport/events.go`
Thêm command + event (đặt cạnh `playerMoveCommand`):

```go
// Client gửi lên qua RPC "voice_signal". Không nhận "from" từ client.
type voiceSignalCommand struct {
	To   string          `json:"to"`   // characterId người nhận
	Kind string          `json:"kind"` // "offer" | "answer" | "ice"
	Data json.RawMessage `json:"data"` // SDP hoặc ICE candidate, server không cần đọc
}

// Server relay xuống personal channel của người nhận.
type voiceSignalEvent struct {
	Type string          `json:"type"` // "voice_signal"
	From string          `json:"from"` // characterId người gửi (server gắn)
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// Server báo cho từng client danh sách peer nên kết nối / ngắt.
type voicePeersEvent struct {
	Type   string        `json:"type"` // "voice_peers"
	Add    []voicePeerRef `json:"add"`
	Remove []string       `json:"remove"` // characterId
}
type voicePeerRef struct {
	CharacterID string `json:"characterId"`
	UserID      string `json:"userId"`
}
```

### 3.2 Đăng ký RPC — `realtime/transport/centrifuge.go` (trong `OnRPC`)
Thêm nhánh method mới cạnh `player_move` / `player_warp`:

```go
switch event.Method {
case "player_move":
	handlePlayerMove(node, roomUsecase, client, event, cb)
case "player_warp":
	handlePlayerWarp(node, roomUsecase, client, event, cb)
case "voice_signal":
	handleVoiceSignal(node, roomUsecase, client, event, cb)
case "mic_state":
	handleMicState(node, roomUsecase, client, event, cb)
default:
	cb(centrifuge.RPCReply{}, centrifuge.ErrorMethodNotFound)
}
```

`handleVoiceSignal` (skeleton — validate proximity rồi relay qua personal channel bằng đúng helper `sendPersonalEvent` bạn đang có ~dòng 268):

```go
func handleVoiceSignal(node *centrifuge.Node, ru *usecase.RoomUsecase, client *centrifuge.Client, event centrifuge.RPCEvent, cb centrifuge.RPCCallback) {
	var cmd voiceSignalCommand
	if err := json.Unmarshal(event.Data, &cmd); err != nil {
		cb(centrifuge.RPCReply{}, centrifuge.ErrorBadRequest); return
	}
	fromUserID := client.UserID() // như player_move đang làm
	// Server tự resolve fromCharID + kiểm 'to' có nằm trong audible-set của from.
	fromCharID, toUserID, ok, err := ru.ResolveVoiceRelay(context.Background(), fromUserID, cmd.To)
	if err != nil { cb(centrifuge.RPCReply{}, centrifuge.ErrorInternal); return }
	if !ok {
		// 'to' không ở gần → từ chối relay
		cb(centrifuge.RPCReply{}, centrifuge.ErrorPermissionDenied); return
	}
	ev := voiceSignalEvent{Type: "voice_signal", From: fromCharID, Kind: cmd.Kind, Data: cmd.Data}
	data, _ := json.Marshal(ev)
	_, _ = node.Publish(personalChannelPrefix+toUserID, data)
	cb(centrifuge.RPCReply{}, nil)
}
```

### 3.3 Proximity recompute — `realtime/usecase/room_usecase.go`
Trong `MovePlayer` (sau khi cập nhật X/Y và trước/song song broadcast `playerMoveEvent`), gọi thêm bước tính neighbor diff cho **người vừa di chuyển** và cho **những người mà quan hệ gần/xa với họ vừa đổi**:

```go
// Sau khi apply movement vào GameRoom:
added, removed := ru.recomputeVoiceNeighbors(room, mover) // dùng spatial hash + hysteresis + cap
for target, diff := range added {   // target = mỗi player bị ảnh hưởng
	ru.publishVoicePeers(target.UserID, diff.add, diff.remove)
}
```

`recomputeVoiceNeighbors` cần dữ liệu vị trí từ `GameRoom`, nhưng store là `ActorRoomStore` (đóng gói trong actor, chỉ vào qua `dispatch`) — không truy cập thẳng `GameRoom.Players` được. Phải hoặc tính neighbor **ngay trong room actor** (có lock), hoặc tính trong usecase từ `store.GetSnapshot`. Tin tốt: `MovePlayer` **đã gọi `GetSnapshot` sẵn** cho check minDistance → tái dùng đúng snapshot đó, khỏi tốn thêm round‑trip. Hàm thuần RAM, không chạm DB. Nhớ: quan hệ gần là **đối xứng**, nên khi A vào vùng B thì cả A và B đều nhận diff.

`ResolveVoiceRelay` cũng dùng snapshot tương tự (from → charID, kiểm `to` thuộc audible‑set hiện tại của from), không tra `GameRoom` trực tiếp.

### 3.4 Dọn khi rời đi
Player `LeaveRoom` / disconnect / warp sang map khác → server bắn `voice_peers.remove` chứa họ tới mọi người từng nghe được họ, để client đóng PeerConnection. Tận dụng đúng chỗ bạn đang xử lý `playerLeftEvent`.

---

## 4. Frontend — một `VoiceSystem` song song `EditorSystem`

Tạo class `VoiceSystem` (ví dụ `frontend/src/features/game/systems/voiceSystem.ts`) quản lý:

- `peers: Map<characterId, { pc: RTCPeerConnection, audioEl, gain: GainNode }>`
- một `AudioContext` dùng chung
- `localStream: MediaStream | null` (chỉ có khi mic bật)
- lắng nghe các event realtime `voice_peers` / `voice_signal` (bridge từ `gameSocket`/`GameScene` như bạn đang làm với `decoration_placed`)
- gọi RPC `centrifuge.rpc('voice_signal', …)` (mở thêm hàm trong `gameSocket.ts` cạnh `player_move`)

### 4.1 Dựng PeerConnection cho một peer mới

```ts
private async addPeer(peerId: string, isOfferer: boolean) {
  const pc = new RTCPeerConnection({
    iceServers: [
      { urls: 'stun:stun.l.google.com:19302' },
      { urls: 'turn:YOUR_TURN_HOST:3478', username: '...', credential: '...' }, // BẮT BUỘC (mục 6)
    ],
  })

  // Luôn có 1 transceiver sendrecv; track thật gắn sau qua replaceTrack (không cần renegotiate)
  const sender = pc.addTransceiver('audio', { direction: 'sendrecv' }).sender
  if (this.localStream) {
    await sender.replaceTrack(this.localStream.getAudioTracks()[0])
  }

  // Nhận audio của peer → đưa vào Web Audio graph để chỉnh volume theo khoảng cách
  pc.ontrack = (e) => this.attachRemoteAudio(peerId, e.streams[0])

  pc.onicecandidate = (e) => {
    if (e.candidate) this.sendSignal(peerId, 'ice', e.candidate)
  }

  this.peers.set(peerId, { pc, gain: null as any, audioEl: null as any })

  if (isOfferer) {
    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)
    this.sendSignal(peerId, 'offer', offer)
  }
}
```

`isOfferer = myCharacterId < peerId` (so sánh chuỗi) — khớp luật deterministic ở server.

### 4.2 Audio graph theo khoảng cách (đây là chỗ tạo cảm giác "xa nhỏ, gần to")

```ts
private attachRemoteAudio(peerId: string, stream: MediaStream) {
  const src = this.audioCtx.createMediaStreamSource(stream)
  const gain = this.audioCtx.createGain()
  gain.gain.value = 0 // sẽ set theo khoảng cách ở updateVolumes()
  src.connect(gain).connect(this.audioCtx.destination)
  // Lưu ý Chrome: MediaStream từ WebRTC đôi khi cần một <audio> ẩn (muted) để "mồi" luồng
  const el = new Audio(); el.srcObject = stream; el.muted = true; el.play().catch(() => {})
  const p = this.peers.get(peerId)!; p.gain = gain; p.audioEl = el
}

// Gọi mỗi ~150–250ms (hoặc mỗi khi có playerMove của peer):
private updateVolumes(myX: number, myY: number, positions: Map<string, {x:number,y:number}>) {
  for (const [peerId, p] of this.peers) {
    const pos = positions.get(peerId); if (!pos || !p.gain) continue
    const d = Math.hypot(pos.x - myX, pos.y - myY)
    // Falloff tuyến tính đơn giản: 1.0 ở sát, 0 ở R_out. Có thể đổi sang inverse-distance.
    const g = Math.max(0, Math.min(1, 1 - d / this.R_OUT))
    p.gain.gain.setTargetAtTime(g, this.audioCtx.currentTime, 0.08) // ramp mượt, tránh "cụp"
  }
}
```

> Nâng cấp tùy chọn: thay `GainNode` bằng `PannerNode` (HRTF) để nghe **hướng** (trái/phải) chứ không chỉ to/nhỏ. MVP dùng gain là đủ cho "đủ gần thì nghe".

### 4.3 Xử lý `voice_signal` đến
Chuẩn WebRTC: `offer` → `setRemoteDescription` → `createAnswer` → gửi `answer`; `answer` → `setRemoteDescription`; `ice` → `addIceCandidate`. Bọc try/catch, và với ICE đến sớm hơn remote description thì đệm lại (candidate queue).

---

## 5. Bật/tắt mic (và "nghe mà không cần mic")

Tách bạch **nghe** (không cần quyền gì) khỏi **nói** (cần `getUserMedia`):

- PeerConnection **luôn** được tạo khi ở gần, kể cả khi bạn chưa bật mic → bạn vẫn **nghe** được người khác ngay.
- **Bật mic lần đầu**: `getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true } })`. Nút toggle chính là "user gesture" cần thiết để mở AudioContext (chính sách autoplay của trình duyệt).
- Sau khi có stream: với **mọi** peer hiện tại, `sender.replaceTrack(micTrack)` — **không cần renegotiate**, đây là mẹo quan trọng để bật/tắt mượt.
- **Tắt mic**: `micTrack.enabled = false` (peer nghe im lặng, kết nối vẫn nguyên → bật lại tức thì). Muốn triệt để hơn thì `replaceTrack(null)`.
- **KHÔNG** nối `localStream` vào `audioCtx.destination` — nếu không bạn sẽ **nghe lại giọng chính mình** (voice loopback). Chỉ remote stream mới nối tới destination.

Trạng thái mic nên broadcast nhẹ để UI hiện biểu tượng 🔇/🎙️ trên đầu nhân vật:

```ts
// RPC "mic_state" → server relay qua room channel (payload bé, tần suất thấp)
centrifuge.rpc('mic_state', { on: true })
```

### "Ai đang nói" (tùy chọn, rất đáng làm)
Dùng `AnalyserNode` trên **local** mic để phát hiện đang nói (RMS vượt ngưỡng), rồi bắn `speaking:true/false` (debounce ~300ms) qua room channel → hiện vòng sáng quanh avatar người đang nói. Không gửi âm lượng liên tục, chỉ đổi trạng thái.

---

## 6. Hạ tầng bắt buộc — đừng bỏ qua

- **TURN server (coturn) là bắt buộc, không phải tùy chọn.** STUN chỉ xử được NAT dễ; nhiều người dùng 4G/CGNAT/symmetric NAT sẽ **không kết nối được** nếu thiếu TURN relay. Không có TURN = "có người nghe được, có người không" và bạn sẽ debug mù. Tự host coturn hoặc dùng dịch vụ TURN.
- **echoCancellation / noiseSuppression / autoGainControl**: bật ngay ở `getUserMedia`, nếu không sẽ vọng và hú khi hai người ngồi gần loa.
- **Autoplay policy**: `AudioContext` phải `resume()` sau một tương tác người dùng — cột nó vào lần bấm nút mic/nút vào game.
- **HTTPS**: `getUserMedia` chỉ chạy trên secure context (bạn deploy qua nginx/vercel nên OK, chỉ lưu ý localhost khi dev).

---

## 7. Bảo mật, chi phí, giới hạn

- Server chỉ relay signaling giữa hai người **thật sự đang gần** (mục 2) → không lộ đường gọi tới user tùy ý; vị trí luôn do server chốt (client không tự khai gần).
- Mesh: mỗi người **upload N lần**. Proximity + `MAX_PEERS` (vd 8) chặn N. Trên mạng gia đình thường, ~6–8 luồng upstream Opus (mỗi ~24–40kbps) là thoải mái; vượt xa hơn ở một chỗ đông → dấu hiệu cần SFU.
- Không có "vật thể ma âm thanh": khi rời map/disconnect, server chủ động phát `voice_peers.remove` → client đóng PC, gỡ audio node (tránh rò `RTCPeerConnection`).

---

## 8. Không đụng gì tới placement / coin
Voice là kênh media riêng, **không** chạm map actor, coin, hay write‑behind. Nó chỉ đọc thêm vị trí từ `GameRoom` (đã có) và thêm RPC/relay ở tầng realtime. Đây là lý do nên đặt toàn bộ trong `module/realtime`, không nhét vào `editor`.

---

## 9. Kế hoạch theo phase

**Phase A — MVP mesh (nghe được là chính):**
- Server: `voice_peers` (spatial hash + hysteresis + cap), relay `voice_signal`, dọn khi leave.
- Client: `VoiceSystem`, PeerConnection theo diff, audio graph gain‑theo‑distance, toggle mic bằng `replaceTrack`.
- STUN + TURN. Test 2–3 tab.
- *Bỏ qua* panner, speaking‑indicator, mic_state UI ở phase này.

**Phase B — hardening:**
- `mic_state` + speaking indicator, biểu tượng mute trên avatar.
- Hysteresis tinh chỉnh, `MAX_PEERS`, ramp gain mượt.
- Đo: số PC trung bình/người, packet loss, CPU client ở cảnh 6–8 người.

**Phase C — SFU khi cần:**
- Khi có cảnh > ~8 người vừa nói vừa nghe một chỗ, hoặc client than nóng máy/nghẽn upstream → chuyển media sang LiveKit/mediasoup; **giữ nguyên** lớp proximity authority ở server (giờ nó điều khiển subscribe/volume thay vì offer/answer). Signaling proximity không phải làm lại.

---

## 10. Checklist test

- [ ] Hai player đi lại gần nhau (< `R_in`) → tự nghe thấy nhau trong ~1–2s; đi ra xa (> `R_out`) → im, PC đóng.
- [ ] Ba+ player quây một chỗ → mỗi người nghe được **tất cả** người còn lại, âm lượng theo khoảng cách.
- [ ] Player tắt mic → vẫn **nghe** được người khác; người khác nghe im lặng chỗ họ.
- [ ] Bật lại mic → nghe lại được ngay (không phải reload, không renegotiate lỗi).
- [ ] Không nghe thấy giọng chính mình (no loopback).
- [ ] Một client sau symmetric NAT/4G vẫn kết nối (chứng minh TURN hoạt động).
- [ ] Đứng đúng ranh giới bán kính không bị connect/disconnect liên tục (hysteresis OK).
- [ ] Warp sang map khác / đóng tab → biến mất khỏi voice của mọi người, không rò kết nối.
- [ ] Thử relay `voice_signal` tới một user **không** ở gần → server từ chối (PermissionDenied).
