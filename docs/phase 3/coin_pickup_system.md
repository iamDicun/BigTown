# Thiết kế tính năng: Hệ thống nhặt coin rơi ngẫu nhiên trên bản đồ (Coin Pickup System)

Hệ thống này cho phép spawn các đồng xu lơ lửng ngẫu nhiên trên bản đồ. Khi người chơi chạm vào đồng xu, coin của họ sẽ tăng lên bất đồng bộ và đồng bộ tức thì sang các người chơi khác.

---

## 1. Tổng quan luồng hoạt động (Data Flow)

```mermaid
sequenceDiagram
    participant Player as Phaser Client (Player)
    participant API as Backend HTTP/WS
    participant Manager as RoomManager
    participant Actor as MapActor
    participant DB as Postgres DB

    Player->>Player: Overlap với Coin Sprite (Phaser Arcade Physics)
    Player->>Player: Chạy animation nhặt xu & hủy Coin object local
    Player->>API: HTTP POST /api/editor/coin-pickup (payload: coin_id, x, y)
    API->>Manager: CreditCoins(mapCode, charID, delta = +10)
    Manager->>Actor: SendCmd(CmdCredit { Coins: 10 })
    Actor->>Actor: Cộng coin trong wallets RAM & gửi outbound broadcast
    Actor->>DB: Gửi persistOp (Ghi nền write-behind)
    Actor-->>API: Trả về số coin mới (NewCoins)
    API-->>Player: HTTP JSON Response (NewCoins)
    Actor->>Player: Centrifuge Broadcast "coin_picked" (Xóa coin đối với các client khác)
```

---

## 2. Thiết kế chi tiết phía Client (Frontend)

### 2.1 Asset & Animation
*   **Asset path**: `frontend/public/assets/tiles/Coin_Gems` (spritesheet tỉ lệ `64x16`, chứa 4-5 frame lặp xoay của đồng xu).
*   Trong Phaser Scene `update` / `create`, nạp spritesheet và tạo animation xoay vòng:
    ```ts
    this.anims.create({
        key: 'coin_spin',
        frames: this.anims.generateFrameNumbers('coin_gems', { start: 0, end: 4 }),
        frameRate: 8,
        repeat: -1
    });
    ```

### 2.2 Collision & Overlap
*   Quản lý danh sách coin đang có trên map bằng một `Phaser.Physics.Arcade.StaticGroup` (hoặc Group thường nếu coin lơ lửng nhún nhảy).
*   Thiết lập overlap listener giữa Player Sprite và Coins Group:
    ```ts
    this.physics.add.overlap(this.player, this.coinsGroup, (player, coin) => {
        const coinObj = coin as Phaser.GameObjects.Sprite;
        const coinId = coinObj.getData('id');
        
        // 1. Chạy hiệu ứng nhặt coin local lập tức (Client prediction)
        coinObj.disableBody(true, true); // Ẩn và tắt va chạm
        this.playPickupSoundAndParticles(coinObj.x, coinObj.y);
        
        // 2. Gửi request lên server
        this.claimCoinOnServer(coinId);
    }, null, this);
    ```

---

## 3. Thiết kế chi tiết phía Server (Backend)

Để tránh tình trạng ghi đè số dư ví (desync coin) khi người chơi đang resident trong map, backend **bắt buộc** phải sử dụng kênh command của Map Actor để cộng tiền:

### 3.1 Endpoint `/api/editor/coin-pickup`
*   **Handler**:
    *   Nhận thông tin người chơi qua JWT middleware (`userID`).
    *   Nhận `map_code` và `coin_id` từ request.
    *   Kiểm tra tính hợp lệ của Coin (nếu server quản lý state spawn).
    *   Gọi `RoomManager.CreditCoins(ctx, mapCode, characterID, coinValue)`.

### 3.2 Tích hợp `RoomManager.CreditCoins`
Hàm này gửi một lệnh `CmdCredit` trực tiếp vào Actor quản lý phòng hiện tại:
```go
func (rm *RoomManager) CreditCoins(ctx context.Context, mapCode, characterID string, delta int) (int, error) {
	a, err := rm.Actor(mapCode)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, errors.New("không tìm thấy bản đồ")
	}

	reply := make(chan CmdResult, 1)
	if err := a.SendCmd(Cmd{
		Kind:   CmdCredit,
		CharID: characterID,
		Coins:  delta,
		Reply:  reply,
	}); err != nil {
		return 0, err
	}
	
	res := <-reply
	return res.NewCoins, res.Err
}
```

---

## 4. Lợi ích kiến trúc
1.  **Nhất quán dữ liệu**: Vì coin được cộng trực tiếp vào ví trong RAM của Actor, khi người chơi rời phòng, số coin nhặt được sẽ được flush đồng bộ xuống Postgres cùng với các hoạt động xây dựng bản đồ. Không bị mất mát/ghi đè.
2.  **Chống cheat cơ bản**: Do server kiểm soát `CmdCredit` và lưu trữ state đồng xu nào đã được nhặt, người chơi không thể gửi request khống để tự nhân số coin.
