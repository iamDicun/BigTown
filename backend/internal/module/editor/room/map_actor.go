package room

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"

	"github.com/google/uuid"
)

// CoinResolver trả về số coin LIVE của character (ưu tiên onlineCoins, fallback DB).
type CoinResolver func(ctx context.Context, charID string) (int, error)

type MapActor struct {
	mapID    string
	mapCode  string
	tileSize int
	mapW     int
	mapH     int

	placementsMu sync.RWMutex
	occupied     map[[2]int][]*entity.Placement
	byID         map[string]*entity.Placement
	hasCollision map[[2]int]bool // có item collision nào trong ô này không
	wallets      map[string]int
	residents    map[string]int
	prices       map[string]int  // itemID -> price (P0 cache)
	collides     map[string]bool // itemID -> has collision (P2 cache)

	coinsOnMap map[string]SpawnedCoin // <-- spawned coins registry
	coinsMu    sync.RWMutex           // <-- reader/writer mutex for spawned coins

	cmds     chan Cmd
	outbound chan any
	dirty    chan persistOp
	done     chan struct{} // channel to block Shutdown until actor loop ends (P1)

	charReader port.CharacterReader
	repo       port.EditorRepository
	coins      CoinResolver // <-- live coin resolver
}

func NewMapActor(
	mapID, mapCode string, mapW, mapH, tileSize int,
	charReader port.CharacterReader, repo port.EditorRepository,
	dirty chan persistOp, publisher port.RoomPublisher,
	coins CoinResolver, // <-- live coin resolver parameter
) *MapActor {
	m := &MapActor{
		mapID:        mapID,
		mapCode:      mapCode,
		tileSize:     tileSize,
		mapW:         mapW,
		mapH:         mapH,
		occupied:     make(map[[2]int][]*entity.Placement),
		byID:         make(map[string]*entity.Placement),
		hasCollision: make(map[[2]int]bool),
		wallets:      make(map[string]int),
		residents:    make(map[string]int),
		prices:       make(map[string]int),
		collides:     make(map[string]bool),
		coinsOnMap:   make(map[string]SpawnedCoin),
		cmds:         make(chan Cmd, 4096),
		outbound:     make(chan any, 1024),
		dirty:        dirty,
		done:         make(chan struct{}),
		charReader:   charReader,
		repo:         repo,
		coins:        coins,
	}

	go m.run()
	go m.broadcastLoop(publisher)

	return m
}

func (m *MapActor) loadFromDB() error {
	// Pre-load all item prices for offline validation (P0)
	items, err := m.repo.GetDecorationItems(context.Background())
	if err != nil {
		return err
	}
	for _, it := range items {
		m.prices[it.ID] = it.Price
		m.collides[it.ID] = parseMetadataCollides(it.MetadataJSON)
	}

	placements, err := m.repo.GetPlacementsByMap(context.Background(), m.mapID)
	if err != nil {
		return err
	}
	for _, p := range placements {
		pCopy := p
		key := [2]int{p.X, p.Y}
		m.occupied[key] = append(m.occupied[key], &pCopy)
		m.byID[p.ID] = &pCopy
		// Track collision per cell
		if m.itemCollides(p.ItemID) {
			m.hasCollision[key] = true
		}
	}

	return nil
}

func (m *MapActor) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case c, ok := <-m.cmds:
			if !ok {
				close(m.outbound)
				close(m.done)
				return
			}
			switch c.Kind {
			case CmdPlace:
				m.handlePlace(c)
			case CmdDelete:
				m.handleDelete(c)
			case CmdJoin:
				hadResidents := len(m.residents) > 0
				if _, ok := m.wallets[c.CharID]; !ok {
					m.wallets[c.CharID] = c.Coins
				}
				m.residents[c.CharID]++
				if !hadResidents {
					m.spawnInitialCoins()
				}
			case CmdLeave:
				if m.residents[c.CharID] > 0 {
					m.residents[c.CharID]--
				}
				if m.residents[c.CharID] <= 0 {
					if coins, ok := m.wallets[c.CharID]; ok {
						m.dirty <- persistOp{
							Kind:     opFlushWallet,
							CharID:   c.CharID,
							NewCoins: coins,
						}
						delete(m.wallets, c.CharID)
					}
					delete(m.residents, c.CharID)
				}
			case CmdCredit:
				coins, err := m.getOrLoadWallet(c.CharID)
				if err != nil {
					c.Reply <- CmdResult{Err: err}
				} else {
					newCoins := coins + c.Coins // c.Coins represents the delta change
					m.wallets[c.CharID] = newCoins
					c.Reply <- CmdResult{NewCoins: newCoins}

					m.dirty <- persistOp{
						Kind:     opFlushWallet,
						CharID:   c.CharID,
						NewCoins: newCoins,
					}
				}
			case CmdClaimCoin:
				m.coinsMu.Lock()
				sc, ok := m.coinsOnMap[c.CoinID]
				if ok {
					delete(m.coinsOnMap, c.CoinID)
				}
				m.coinsMu.Unlock()

				if !ok {
					c.Reply <- CmdResult{Err: ErrNotFound}
					break
				}

				delta := coinValue(sc.Type)
				coins, err := m.getOrLoadWallet(c.CharID)
				if err != nil {
					c.Reply <- CmdResult{Err: err}
					break
				}
				newCoins := coins + delta
				m.wallets[c.CharID] = newCoins
				c.Reply <- CmdResult{NewCoins: newCoins}

				// Broadcast coin_picked to all players in real-time
				m.outbound <- map[string]any{
					"type":   "coin_picked",
					"coinId": c.CoinID,
				}

				m.dirty <- persistOp{
					Kind:      opFlushWallet,
					CharID:    c.CharID,
					NewCoins:  newCoins,
					CoinDelta: delta,
					EventType: "coin_pickup",
				}
			}
		case <-ticker.C:
			m.tickCoins()
		}
	}
}

func (m *MapActor) handlePlace(c Cmd) {
	// 1. Validate coordinates (bounds + grid)
	if m.tileSize <= 0 {
		c.Reply <- CmdResult{Err: errors.New("tileSize must be positive")}
		return
	}
	if c.X%m.tileSize != 0 || c.Y%m.tileSize != 0 {
		log.Printf("DEBUG: placement coordinate is not matching snap grid: X=%d, Y=%d, tileSize=%d", c.X, c.Y, m.tileSize)
		c.Reply <- CmdResult{Err: errors.New("toạ độ không khớp snap grid")}
		return
	}
	if c.X < 0 || c.X >= m.mapW || c.Y < 0 || c.Y >= m.mapH {
		c.Reply <- CmdResult{Err: errors.New("toạ độ vượt quá giới hạn bản đồ")}
		return
	}

	key := [2]int{c.X, c.Y}

	// Giới hạn tối đa 2 item trên cùng 1 tọa độ tile
	if len(m.occupied[key]) >= 2 {
		c.Reply <- CmdResult{Err: ErrOccupied}
		return
	}

	// Chỉ chặn nếu ô đã có item collision (không cho đè lên item collision)
	if m.hasCollision[key] {
		// Kiểm tra xem item mới có collision không — nếu có thì chặn
		newCollides := parseMetadataCollides(c.Item.MetadataJSON)
		if newCollides || len(m.occupied[key]) > 0 {
			c.Reply <- CmdResult{Err: ErrOccupied}
			return
		}
	}
	// Nếu item mới không collision và ô có sẵn item không collision → cho phép stacking
	// Nếu ô có sẵn item collision → đã bị chặn ở trên
	// Nếu ô trống → cho phép như cũ

	coins, err := m.getOrLoadWallet(c.CharID)
	if err != nil {
		c.Reply <- CmdResult{Err: err}
		return
	}

	if coins < c.Item.Price {
		c.Reply <- CmdResult{Err: ErrInsufficientCoins}
		return
	}

	// MUTATE RAM = Final Authority Decision
	newCoins := coins - c.Item.Price
	m.wallets[c.CharID] = newCoins

	p := &entity.Placement{
		ID:          c.PlaceID,
		MapID:       m.mapID,
		CharacterID: c.CharID,
		ItemID:      c.Item.ID,
		X:           c.X,
		Y:           c.Y,
		Rotation:    c.Rotation,
		CreatedAt:   time.Now(),
	}
	m.placementsMu.Lock()
	m.occupied[key] = append(m.occupied[key], p)
	m.byID[p.ID] = p
	m.placementsMu.Unlock()
	if parseMetadataCollides(c.Item.MetadataJSON) {
		m.hasCollision[key] = true
	}

	// Reply to HTTP handler immediately
	c.Reply <- CmdResult{Placement: p, NewCoins: newCoins}

	// Broadcast asynchronously
	m.outbound <- map[string]any{
		"type":      "decoration_placed",
		"placement": p,
	}

	// Send to write-behind background pipeline
	m.dirty <- persistOp{
		Kind:      opPlace,
		P:         p,
		CharID:    c.CharID,
		NewCoins:  newCoins,
		CoinDelta: -c.Item.Price,
		EventType: "decoration_place",
	}
}

func (m *MapActor) handleDelete(c Cmd) {
	p, ok := m.byID[c.TargetID]
	if !ok {
		c.Reply <- CmdResult{Err: ErrNotFound}
		return
	}

	if p.CharacterID != c.CharID {
		c.Reply <- CmdResult{Err: ErrNotOwner}
		return
	}

	price, ok := m.prices[p.ItemID]
	if !ok {
		// Fallback for newly added items during runtime (P0)
		item, err := m.repo.GetItemByID(context.Background(), p.ItemID)
		if err != nil || item == nil {
			c.Reply <- CmdResult{Err: ErrNotFound}
			return
		}
		price = item.Price
		m.prices[p.ItemID] = price
	}

	coins, err := m.getOrLoadWallet(c.CharID)
	if err != nil {
		c.Reply <- CmdResult{Err: err}
		return
	}

	// MUTATE RAM = Final Decision
	newCoins := coins + price
	m.wallets[c.CharID] = newCoins

	key := [2]int{p.X, p.Y}
	m.placementsMu.Lock()
	delete(m.byID, p.ID)
	if list, ok := m.occupied[key]; ok {
		filtered := list[:0]
		for _, pp := range list {
			if pp.ID != p.ID {
				filtered = append(filtered, pp)
			}
		}
		if len(filtered) == 0 {
			delete(m.occupied, key)
			delete(m.hasCollision, key)
		} else {
			m.occupied[key] = filtered
			hasCol := false
			for _, pp := range filtered {
				if m.itemCollides(pp.ItemID) {
					hasCol = true
					break
				}
			}
			if !hasCol {
				delete(m.hasCollision, key)
			}
		}
	}
	m.placementsMu.Unlock()

	// Reply immediately
	c.Reply <- CmdResult{NewCoins: newCoins}

	// Broadcast asynchronously
	m.outbound <- map[string]any{
		"type":        "decoration_deleted",
		"placementId": p.ID,
	}

	// Write-behind
	m.dirty <- persistOp{
		Kind:      opDelete,
		P:         p,
		CharID:    c.CharID,
		NewCoins:  newCoins,
		CoinDelta: price,
		EventType: "decoration_refund",
	}
}

func (m *MapActor) getOrLoadWallet(charID string) (int, error) {
	if coins, ok := m.wallets[charID]; ok {
		return coins, nil // ví của actor này đã seed → là nguồn chuẩn tại chỗ
	}
	// Chưa seed: hỏi ví LIVE (onlineCoins trước, DB sau) — không bao giờ đọc DB "trần"
	live, err := m.coins(context.Background(), charID)
	if err != nil {
		return 0, err
	}
	m.wallets[charID] = live
	return live, nil
}

func (m *MapActor) broadcastLoop(pub port.RoomPublisher) {
	for ev := range m.outbound {
		// Centrifuge publishing is isolated so net latency won't block the actor tick loop
		if err := pub.PublishRoom(context.Background(), m.mapCode, ev); err != nil {
			log.Printf("[room %s] Centrifuge broadcast failed: %v", m.mapCode, err)
		}
	}
}

func (m *MapActor) SendCmd(cmd Cmd) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrBusy
		}
	}()

	select {
	case m.cmds <- cmd:
		return nil
	default:
		return ErrBusy
	}
}

func (m *MapActor) GetSpawnedCoins() []SpawnedCoin {
	m.coinsMu.RLock()
	defer m.coinsMu.RUnlock()

	coins := make([]SpawnedCoin, 0, len(m.coinsOnMap))
	for _, sc := range m.coinsOnMap {
		coins = append(coins, sc)
	}
	return coins
}

func (m *MapActor) isTileFreeForCoin(x, y int) bool {
	// 1. Check bounds
	if x < 0 || x >= m.mapW || y < 0 || y >= m.mapH {
		return false
	}
	// 2. Check occupied placements
	key := [2]int{x, y}
	if _, taken := m.occupied[key]; taken {
		return false
	}
	// 3. Check existing spawned coins
	// No lock needed here as this is executed sequentially within actor or protected by coinsMu at call site
	for _, sc := range m.coinsOnMap {
		if sc.X == x && sc.Y == y {
			return false
		}
	}
	return true
}

func (m *MapActor) findFreeTileForCoin() (int, int, bool) {
	cols := m.mapW / m.tileSize
	rows := m.mapH / m.tileSize
	if cols <= 0 || rows <= 0 {
		return 0, 0, false
	}

	for i := 0; i < 100; i++ {
		c := rand.Intn(cols)
		r := rand.Intn(rows)
		x := c * m.tileSize
		y := r * m.tileSize
		if m.isTileFreeForCoin(x, y) {
			return x, y, true
		}
	}
	return 0, 0, false
}

func (m *MapActor) spawnInitialCoins() {
	if m.mapCode != "winter" && m.mapCode != "dark_village" {
		return
	}
	if len(m.residents) == 0 {
		return
	}
	m.coinsMu.Lock()
	defer m.coinsMu.Unlock()

	types := []string{"gri", "ama", "azu", "roj", "gold"}
	for _, t := range types {
		for i := 0; i < 20; i++ { // spawn 20 initial coins of each type
			x, y, ok := m.findFreeTileForCoin()
			if !ok {
				break
			}
			id := uuid.NewString()
			m.coinsOnMap[id] = SpawnedCoin{
				ID:   id,
				Type: t,
				X:    x,
				Y:    y,
			}
		}
	}
}

func (m *MapActor) tickCoins() {
	if m.mapCode != "winter" && m.mapCode != "dark_village" {
		return
	}
	if len(m.residents) == 0 {
		return
	}

	var toBroadcast []SpawnedCoin

	m.coinsMu.Lock()
	types := []string{"gri", "ama", "azu", "roj", "gold"}
	for _, t := range types {
		count := 0
		for _, sc := range m.coinsOnMap {
			if sc.Type == t {
				count++
			}
		}
		if count < 30 { // Limit maximum of 30 coins per type
			spawnCount := 1
			if count < 20 { // If it drops below 20, spawn 2
				spawnCount = 2
			}
			for k := 0; k < spawnCount; k++ {
				if count >= 30 {
					break
				}
				x, y, ok := m.findFreeTileForCoin()
				if ok {
					id := uuid.NewString()
					sc := SpawnedCoin{
						ID:   id,
						Type: t,
						X:    x,
						Y:    y,
					}
					m.coinsOnMap[id] = sc
					toBroadcast = append(toBroadcast, sc)
					count++
				}
			}
		}
	}
	m.coinsMu.Unlock()

	// Broadcast events safely without holding coinsMu lock
	for _, sc := range toBroadcast {
		m.outbound <- map[string]any{
			"type": "coin_spawned",
			"coin": sc,
		}
	}
}

func coinValue(t string) int {
	switch t {
	case "gri":
		return 5
	case "ama":
		return 10
	case "azu":
		return 25
	case "roj":
		return 50
	case "gold":
		return 100
	default:
		return 10
	}
}

func parseMetadataCollides(metadataJSON string) bool {
	if metadataJSON == "" || metadataJSON == "{}" {
		return false
	}
	var meta struct {
		Collides bool `json:"collides"`
	}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return false
	}
	return meta.Collides
}

func (m *MapActor) itemCollides(itemID string) bool {
	if c, ok := m.collides[itemID]; ok {
		return c
	}
	item, err := m.repo.GetItemByID(context.Background(), itemID)
	if err != nil || item == nil {
		return false
	}
	c := parseMetadataCollides(item.MetadataJSON)
	m.collides[itemID] = c
	return c
}

func (m *MapActor) GetPlacements() []entity.Placement {
	m.placementsMu.RLock()
	defer m.placementsMu.RUnlock()
	result := make([]entity.Placement, 0, len(m.byID))
	for _, p := range m.byID {
		result = append(result, *p)
	}
	return result
}

func (m *MapActor) CmdQueueLen() int { return len(m.cmds) }
