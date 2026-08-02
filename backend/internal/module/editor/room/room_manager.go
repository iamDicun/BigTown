package room

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"backend/internal/module/editor/port"
	realtimePort "backend/internal/module/realtime/port"
)

var _ realtimePort.RoomEventListener = (*RoomManager)(nil)

type RoomManager struct {
	mu     sync.RWMutex
	actors map[string]*MapActor

	db          *sql.DB
	publisher   port.RoomPublisher
	repo        port.EditorRepository
	charReader  port.CharacterReader
	writer      *Writer
	onlineCoins map[string]int
	liveConns   map[string]int // <-- số kết nối đang online của mỗi char
}

func NewRoomManager(db *sql.DB, publisher port.RoomPublisher, repo port.EditorRepository, charReader port.CharacterReader) *RoomManager {
	rm := &RoomManager{
		actors:      make(map[string]*MapActor),
		db:          db,
		publisher:   publisher,
		repo:        repo,
		charReader:  charReader,
		onlineCoins: make(map[string]int),
		liveConns:   make(map[string]int),
	}
	rm.writer = NewWriter(db, rm)
	return rm
}

func (rm *RoomManager) GetCoins(ctx context.Context, characterID string) (int, error) {
	rm.mu.RLock()
	coins, ok := rm.onlineCoins[characterID]
	rm.mu.RUnlock()
	if ok {
		return coins, nil
	}
	return rm.charReader.GetCoins(ctx, characterID)
}

func (rm *RoomManager) SetOnlineCoins(characterID string, coins int) {
	rm.mu.Lock()
	rm.onlineCoins[characterID] = coins
	rm.mu.Unlock()
}

func (rm *RoomManager) EvictOnlineCoins(characterID string) {
	rm.mu.Lock()
	delete(rm.onlineCoins, characterID)
	rm.mu.Unlock()
}

func (rm *RoomManager) Actor(mapCode string) (*MapActor, error) {
	rm.mu.RLock()
	a, ok := rm.actors[mapCode]
	rm.mu.RUnlock()
	if ok {
		return a, nil
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// double check
	if a, ok = rm.actors[mapCode]; ok {
		return a, nil
	}

	// Load map info from DB
	mapInfo, err := rm.repo.GetMapInfoByCode(context.Background(), mapCode)
	if err != nil {
		return nil, err
	}
	if mapInfo == nil {
		return nil, nil // not found
	}

	a = NewMapActor(
		mapInfo.ID,
		mapCode,
		mapInfo.Width*mapInfo.TileSize,
		mapInfo.Height*mapInfo.TileSize,
		mapInfo.TileSize,
		rm.charReader,
		rm.repo,
		rm.writer.in,
		rm.publisher,
		rm.GetCoins, // <-- truyền resolver ví live
	)
	if err := a.loadFromDB(); err != nil {
		return nil, err
	}

	rm.actors[mapCode] = a
	return a, nil
}

func (rm *RoomManager) Shutdown() {
	rm.mu.Lock()
	actors := make([]*MapActor, 0, len(rm.actors))
	for _, a := range rm.actors {
		actors = append(actors, a)
	}
	rm.mu.Unlock()

	// 1) Stop receiving new commands
	for _, a := range actors {
		close(a.cmds)
	}

	// 2) Wait for all actors to finish draining (P1)
	for _, a := range actors {
		<-a.done
	}

	// 3) Now it is safe to close the writer and wait
	rm.writer.Close()
}

// CreditCoins cộng/trừ coin cho một character ĐANG resident trong map (P2)
func (rm *RoomManager) CreditCoins(ctx context.Context, mapCode, characterID string, delta int) (int, error) {
	a, err := rm.Actor(mapCode)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, errors.New("map not found")
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
	if res.Err == nil {
		rm.SetOnlineCoins(characterID, res.NewCoins)
	}
	return res.NewCoins, res.Err
}

func (rm *RoomManager) OnPlayerJoin(ctx context.Context, roomID string, characterID string, coins int) error {
	rm.mu.Lock()
	if _, exists := rm.onlineCoins[characterID]; !exists {
		rm.onlineCoins[characterID] = coins
	}
	rm.liveConns[characterID]++          // <-- đếm kết nối lên
	currentCoins := rm.onlineCoins[characterID]
	rm.mu.Unlock()

	a, err := rm.Actor(roomID)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	return a.SendCmd(Cmd{
		Kind:   CmdJoin,
		CharID: characterID,
		Coins:  currentCoins,
	})
}

func (rm *RoomManager) OnPlayerLeave(ctx context.Context, roomID string, characterID string) error {
	rm.mu.Lock()
	if rm.liveConns[characterID] > 0 {
		rm.liveConns[characterID]--
	}
	gone := rm.liveConns[characterID] <= 0
	if gone {
		delete(rm.liveConns, characterID)
	}
	rm.mu.Unlock()

	if gone {
		// Debounce: warp = leave→join, liveConns chạm 0 vài trăm ms rồi lại lên.
		// Chỉ evict nếu sau grace vẫn không có kết nối nào (offline thật).
		go rm.scheduleEvict(characterID)
	}

	a, err := rm.Actor(roomID)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	return a.SendCmd(Cmd{
		Kind:   CmdLeave,
		CharID: characterID,
	})
}

func (rm *RoomManager) scheduleEvict(characterID string) {
	time.Sleep(3 * time.Second) // grace cho warp/reconnect
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.liveConns[characterID] == 0 { // vẫn offline sau grace
		delete(rm.onlineCoins, characterID)
	}
}

func (rm *RoomManager) GetSpawnedCoins(mapCode string) []SpawnedCoin {
	a, err := rm.Actor(mapCode)
	if err != nil || a == nil {
		return nil
	}
	return a.GetSpawnedCoins()
}

func (rm *RoomManager) ClaimCoin(ctx context.Context, mapCode, characterID, coinID string) (int, error) {
	a, err := rm.Actor(mapCode)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, errors.New("map not found")
	}

	reply := make(chan CmdResult, 1)
	if err := a.SendCmd(Cmd{
		Kind:   CmdClaimCoin,
		CharID: characterID,
		CoinID: coinID,
		Reply:  reply,
	}); err != nil {
		return 0, err
	}

	res := <-reply
	if res.Err == nil {
		rm.SetOnlineCoins(characterID, res.NewCoins)
	}
	return res.NewCoins, res.Err
}
