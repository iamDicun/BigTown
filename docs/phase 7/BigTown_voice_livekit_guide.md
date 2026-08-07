# BigTown — Voice chat theo khoảng cách qua LiveKit Cloud (hợp với Render free)

Mục tiêu: người chơi bật mic nói, **chỉ ai đủ gần mới nghe được**, âm nhỏ dần theo khoảng cách. Media chạy trên hạ tầng LiveKit; **backend Render chỉ ký token** nên free tier vẫn chạy tốt (không giọt media nào qua box của bạn).

---

## 0. Kiến trúc: vì sao hợp Render free

```
Client  ──WebRTC (audio)──►  LiveKit Cloud (SFU + TURN + signaling)  ◄──WebRTC──  Client khác
   │                                                                      
   │ 1) xin token (HTTPS)                                                 
   ▼                                                                      
Backend Go trên Render  ── chỉ ký JWT bằng API key/secret ──             
   │                                                                      
   └─ vẫn giữ vai "proximity authority": actor biết X,Y mọi player  ──► quyết ai nghe ai
```

- Backend **không** đụng media → không cần UDP, không sợ spin-down. Endpoint token chỉ là REST thường.
- LiveKit Cloud lo luôn **TURN/relay** — thứ mà mesh tự-host trên Render free không làm được.
- Proximity authority (biết ai gần ai) **tái dùng** đúng dữ liệu vị trí actor đã có (`GameRoom.Players[characterID].X/Y`, cập nhật mỗi `player_move`). Giống hệt tinh thần `proximity_voice_design.md`, chỉ khác: thay vì tự làm mesh SDP/ICE, ta điều khiển **subscription** của LiveKit.

---

## 1. Đăng ký LiveKit Cloud & lấy credentials

1. Vào `cloud.livekit.io`, đăng ký (GitHub/Google/email) — bản **Build miễn phí, không cần thẻ**.
2. Tạo một **Project**. Sau khi tạo bạn có:
   - **Project URL** dạng `wss://<project>.livekit.cloud` (công khai — dùng ở frontend).
   - **API Key** và **API Secret** (bí mật — chỉ để ở backend), lấy trong **Settings → Keys**.
3. Hạn mức Build: tính theo **participant-minute** + băng thông, reset hàng tháng, **không cho vượt** (hard cap). Đủ cho MVP/thử nghiệm; xem trang pricing hiện tại để biết con số chính xác trước khi lên production.

> Quy tắc vàng: **API Secret không bao giờ ra tới client.** Token luôn ký ở backend.

---

## 2. Cấu hình biến môi trường (Render → Environment)

```env
LIVEKIT_URL=wss://<project>.livekit.cloud
LIVEKIT_API_KEY=API xxxxxxxx
LIVEKIT_API_SECRET=secret_xxxxxxxxxxxxxxxx
```

Frontend chỉ cần biết URL công khai:

```env
VITE_LIVEKIT_URL=wss://<project>.livekit.cloud
```

---

## 3. Backend Go — endpoint ký token

Cài SDK:

```bash
go get github.com/livekit/server-sdk-go/v2
go get github.com/livekit/protocol/auth
```

`internal/module/voice/token.go`:

```go
package voice

import (
	"time"

	"github.com/livekit/protocol/auth"
)

type TokenIssuer struct {
	apiKey    string
	apiSecret string
}

func NewTokenIssuer(apiKey, apiSecret string) *TokenIssuer {
	return &TokenIssuer{apiKey: apiKey, apiSecret: apiSecret}
}

// Mint: room = mapCode (mỗi map một voice room), identity = characterID.
// canPublish/canSubscribe = true; nhưng "nghe ai" sẽ do proximity chốt (mục 6).
func (t *TokenIssuer) Mint(room, characterID, displayName string) (string, error) {
	canPub, canSub := true, true
	at := auth.NewAccessToken(t.apiKey, t.apiSecret)
	at.SetVideoGrant(&auth.VideoGrant{
		RoomJoin:     true,
		Room:         room,
		CanPublish:   &canPub,
		CanSubscribe: &canSub,
	})
	at.SetIdentity(characterID). // identity phải unique & ổn định để map về player
		SetName(displayName).
		SetValidFor(time.Hour)
	return at.ToJWT()
}
```

Gin handler (dùng lại auth middleware sẵn có để lấy characterID từ JWT của bạn — client **không** tự khai identity):

```go
// POST /voice/token   body: { "map_code": "winter" }
func (h *VoiceHandler) IssueToken(c *gin.Context) {
	charID := middleware.CharacterIDFromCtx(c) // lấy từ access token của BigTown
	name := middleware.DisplayNameFromCtx(c)

	var body struct{ MapCode string `json:"map_code" binding:"required"` }
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "map_code required"})
		return
	}

	tok, err := h.issuer.Mint(body.MapCode, charID, name)
	if err != nil {
		c.JSON(500, gin.H{"error": "cannot mint token"})
		return
	}
	c.JSON(200, gin.H{"token": tok, "url": h.livekitURL})
}
```

Đăng ký route trong `registerModules` (protected). Xong phần backend bắt buộc — nhẹ đúng một endpoint.

---

## 4. Frontend — kết nối, publish mic, phát audio

Cài: `npm i livekit-client`

```ts
import {
  Room, RoomEvent, Track,
  type RemoteTrack, type RemoteTrackPublication, type RemoteParticipant,
} from 'livekit-client'

let room: Room | null = null

export async function joinVoice(mapCode: string) {
  // 1) xin token từ backend
  const { token, url } = await http.post('/voice/token', { map_code: mapCode })

  // 2) tạo room, TẮT autoSubscribe để tự chốt ai nghe ai (mục 6)
  room = new Room({ adaptiveStream: true, dynacast: true })

  room
    .on(RoomEvent.TrackSubscribed, onTrackSubscribed)
    .on(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed)
    .on(RoomEvent.AudioPlaybackStatusChanged, () => {
      // trình duyệt chặn autoplay → cần 1 cú click của user
      if (room && !room.canPlayAudio) showEnableAudioButton()
    })

  await room.connect(url, token, { autoSubscribe: false })
}

// Push-to-talk / toggle mic
export async function setMic(on: boolean) {
  await room?.localParticipant.setMicrophoneEnabled(on)
}

function onTrackSubscribed(track: RemoteTrack) {
  if (track.kind === Track.Kind.Audio) {
    const el = track.attach()      // tạo <audio> ẩn
    el.style.display = 'none'
    document.body.appendChild(el)
  }
}
function onTrackUnsubscribed(track: RemoteTrack) {
  track.detach().forEach((el) => el.remove())
}

// gắn vào onclick của nút "Bật âm thanh"
export async function enableAudio() { await room?.startAudio() }

export async function leaveVoice() {
  await room?.disconnect(); room = null
}
```

> **Autoplay:** trình duyệt (nhất là Safari/iOS) chặn phát audio nếu chưa có tương tác. Luôn có một nút để gọi `room.startAudio()` trong sự kiện click. WebRTC cũng bắt buộc HTTPS/WSS — bạn đã có TLS qua Vercel/nginx nên ổn.

---

## 5. Bài toán cốt lõi: "đủ gần mới nghe được"

Ý tưởng: **mọi người trong một map vào chung một LiveKit room** (`room = mapCode`). Proximity không phải là "ai vào room" mà là **ai subscribe track của ai** + **volume theo khoảng cách**. Có hai tầng, làm tầng nào cũng được, kết hợp thì tốt nhất.

Bạn **đã có sẵn luồng vị trí**: server broadcast `room_state` mỗi tick 100ms với X,Y mọi player. Dùng chính nó làm đầu vào proximity — không cần thêm kênh.

### 5A. Hàm khoảng cách → volume (client, luôn cần)

```ts
const R_FULL = 96    // trong bán kính này: nghe rõ 100%
const R_MAX  = 320   // ngoài bán kính này: câm (unsubscribe)

function volumeForDistance(d: number): number {
  if (d <= R_FULL) return 1
  if (d >= R_MAX)  return 0
  return 1 - (d - R_FULL) / (R_MAX - R_FULL) // giảm tuyến tính; đổi sang mũ nếu muốn "ấm" hơn
}
```

### 5B. Cách 1 — Client tự chốt (đơn giản, làm trước)

Mỗi khi nhận `room_state` (đã có sẵn), tính khoảng cách tới từng participant rồi bật/tắt subscribe + chỉnh volume. `identity` của LiveKit chính là `characterID` (đã set ở token) nên map thẳng về vị trí.

```ts
// gọi mỗi lần cập nhật vị trí (throttle ~5–10/s là đủ)
export function updateVoiceProximity(me: {x:number;y:number},
                                     positions: Map<string,{x:number;y:number}>) {
  if (!room) return
  for (const [identity, p] of room.remoteParticipants) {
    const pos = positions.get(identity)
    if (!pos) continue
    const d = Math.hypot(pos.x - me.x, pos.y - me.y)
    const vol = volumeForDistance(d)

    // gate nghe/không: subscribe khi trong R_MAX, unsubscribe khi ra ngoài
    for (const pub of p.trackPublications.values()) {
      if (pub.kind !== Track.Kind.Audio) continue
      const want = vol > 0
      if (want && !pub.isSubscribed) pub.setSubscribed(true)
      else if (!want && pub.isSubscribed) pub.setSubscribed(false)
    }
    // spatial: chỉnh volume trên track mic của participant
    p.setVolume(vol)
  }
}
```

> **Hysteresis** (chống nhấp nháy ở ranh giới): subscribe khi `d < R_MAX_IN` (vd 300), chỉ unsubscribe khi `d > R_MAX_OUT` (vd 340). Nhớ trạng thái subscribe trước đó để chỉ gọi khi đổi — tránh spam SFU.

**Ưu:** ~30 dòng, tái dùng luồng vị trí, không đụng backend. **Nhược:** client tự quyết subscribe → client bị hack có thể subscribe *tất cả* người trong cùng LiveKit room để nghe lén. Với game thị trấn thường chấp nhận được; giảm rủi ro bằng cách chia room nhỏ (mục 7).

### 5C. Cách 2 — Server chốt (chống nghe lén, khớp `proximity_voice_design.md`)

Nếu cần đúng tinh thần "client không được tự khai gần ai", để **backend** quản subscription qua `RoomServiceClient`. Actor của bạn đã có spatial hash + audible-set (hysteresis, cap fan-out) trong doc voice — nay nó không bắn `voice_peers` cho mesh nữa, mà gọi thẳng LiveKit:

```go
import (
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/livekit/protocol/livekit"
)

rs := lksdk.NewRoomServiceClient(livekitURL, apiKey, apiSecret)

// Khi audible-set của "listener" đổi (tính trong actor):
// subscribe thêm / hủy subscribe track mic của những người vừa vào/ra bán kính.
_, err := rs.UpdateSubscriptions(ctx, &livekit.UpdateSubscriptionsRequest{
	Room:      mapCode,
	Identity:  listenerCharID,   // người NGHE
	TrackSids: audioTrackSids,   // track mic của những người trong bán kính
	Subscribe: true,             // false để hủy khi ra khỏi bán kính
})
```

SFU chỉ forward track đã được server subscribe → client hack cũng không nghe được người ngoài bán kính. Lấy `TrackSids`: đăng ký **webhook** `track_published` của LiveKit (SDK Go có verify webhook) để map `identity → trackSid`, hoặc gọi `rs.ListParticipants`. Volume theo khoảng cách vẫn để client làm (5A) — volume không làm lộ audio vì việc *nghe được hay không* đã bị server gate.

**Khuyến nghị:** ship 5B trước cho nhanh; nâng lên 5C khi cần chống nghe lén nghiêm túc (hoặc khi có "phòng riêng" cần bảo mật).

---

## 6. Chi phí & mẹo tiết kiệm participant-minute

LiveKit tính tiền theo **thời gian mỗi participant ở trong room**, bất kể có nghe ai không. Nên:

1. **Chỉ join room khi bật voice** (nút toggle / push-to-talk), **disconnect khi tắt** — đừng auto-join lúc vào map. Đây là đòn tiết kiệm lớn nhất.
2. **Chia room theo zone** thay vì cả map một room: `room = mapCode + ":" + zoneId`. Vừa giảm participant-minute (ít người/room), vừa **thu nhỏ bán kính nghe lén** ở Cách 1.
3. Tự động rời room nếu quanh mình không có ai trong `R_MAX` một lúc lâu.
4. Theo dõi usage trên dashboard LiveKit; đặt cảnh báo trước khi chạm hạn mức Build.

---

## 7. Lưu ý Teams & trình duyệt

- App nhúng trong Microsoft Teams muốn xin mic phải khai `devicePermissions: ["media"]` trong manifest; test kỹ trong Teams client (không chỉ browser).
- Người dùng có thể đang ở sẵn trong cuộc gọi Teams — cân nhắc UX để voice game không đè tiếng họp (mặc định tắt, chỉ bật khi user chủ động).
- Echo/noise: LiveKit bật sẵn echo cancellation/noise suppression của WebRTC; cho phép user chọn thiết bị mic.

---

## Checklist

- [ ] Đăng ký LiveKit Cloud (Build free), tạo project, lấy URL + API key/secret
- [ ] Set `LIVEKIT_URL/API_KEY/API_SECRET` ở Render; `VITE_LIVEKIT_URL` ở frontend
- [ ] Backend: `voice.TokenIssuer` + route `POST /voice/token` (identity = characterID từ auth middleware)
- [ ] Frontend: join room (`autoSubscribe:false`), push-to-talk `setMicrophoneEnabled`, attach audio, nút `startAudio`
- [ ] Proximity 5B: `volumeForDistance` + `setSubscribed`/`setVolume` theo `room_state` (có hysteresis)
- [ ] Tiết kiệm: chỉ connect khi bật voice; cân nhắc chia room theo zone
- [ ] (Sau) Proximity 5C server-authoritative qua `UpdateSubscriptions` + webhook `track_published`
- [ ] (Teams) khai `devicePermissions: ["media"]`, test trong Teams client
```
