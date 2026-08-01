package room

import (
	"context"
	"database/sql"
	"sync"

	"backend/internal/module/editor/port"
	realtimePort "backend/internal/module/realtime/port"
)

var _ realtimePort.RoomEventListener = (*RoomManager)(nil)

type RoomManager struct {
	mu     sync.RWMutex
	actors map[string]*MapActor

	db         *sql.DB
	publisher  port.RoomPublisher
	repo       port.EditorRepository
	charReader port.CharacterReader
	writer     *Writer
}

func NewRoomManager(db *sql.DB, publisher port.RoomPublisher, repo port.EditorRepository, charReader port.CharacterReader) *RoomManager {
	return &RoomManager{
		actors:     make(map[string]*MapActor),
		db:         db,
		publisher:  publisher,
		repo:       repo,
		charReader: charReader,
		writer:     NewWriter(db),
	}
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

	a = NewMapActor(mapInfo.ID, mapCode, mapInfo.Width, mapInfo.Height, mapInfo.TileSize, rm.charReader, rm.repo, rm.writer.in, rm.publisher)
	if err := a.loadFromDB(); err != nil {
		return nil, err
	}

	rm.actors[mapCode] = a
	return a, nil
}

func (rm *RoomManager) Shutdown() {
	rm.mu.Lock()
	for _, a := range rm.actors {
		close(a.cmds) // close commands channel
	}
	rm.mu.Unlock()

	rm.writer.Close() // close done flusher and wait
}

func (rm *RoomManager) OnPlayerJoin(ctx context.Context, roomID string, characterID string, coins int) error {
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
		Coins:  coins,
	})
}

func (rm *RoomManager) OnPlayerLeave(ctx context.Context, roomID string, characterID string) error {
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
