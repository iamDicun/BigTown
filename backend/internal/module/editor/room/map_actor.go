package room

import (
	"context"
	"errors"
	"log"
	"time"

	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
)

type MapActor struct {
	mapID    string
	mapCode  string
	tileSize int
	mapW     int
	mapH     int

	occupied  map[[2]int]*entity.Placement
	byID      map[string]*entity.Placement
	wallets   map[string]int
	residents map[string]int

	cmds     chan Cmd
	outbound chan any
	dirty    chan persistOp

	charReader port.CharacterReader
	repo       port.EditorRepository
}

func NewMapActor(mapID, mapCode string, mapW, mapH, tileSize int, charReader port.CharacterReader, repo port.EditorRepository, dirty chan persistOp, publisher port.RoomPublisher) *MapActor {
	m := &MapActor{
		mapID:      mapID,
		mapCode:    mapCode,
		tileSize:   tileSize,
		mapW:       mapW,
		mapH:       mapH,
		occupied:   make(map[[2]int]*entity.Placement),
		byID:       make(map[string]*entity.Placement),
		wallets:    make(map[string]int),
		residents:  make(map[string]int),
		cmds:       make(chan Cmd, 4096),
		outbound:   make(chan any, 1024),
		dirty:      dirty,
		charReader: charReader,
		repo:       repo,
	}

	go m.run()
	go m.broadcastLoop(publisher)

	return m
}

func (m *MapActor) loadFromDB() error {
	placements, err := m.repo.GetPlacementsByMap(context.Background(), m.mapID)
	if err != nil {
		return err
	}
	for _, p := range placements {
		pCopy := p
		key := [2]int{p.X, p.Y}
		m.occupied[key] = &pCopy
		m.byID[p.ID] = &pCopy
	}
	return nil
}

func (m *MapActor) run() {
	for c := range m.cmds {
		switch c.Kind {
		case CmdPlace:
			m.handlePlace(c)
		case CmdDelete:
			m.handleDelete(c)
		case CmdJoin:
			m.wallets[c.CharID] = c.Coins
			m.residents[c.CharID]++
		case CmdLeave:
			m.residents[c.CharID]--
			if m.residents[c.CharID] <= 0 {
				m.dirty <- persistOp{
					Kind:     opFlushWallet,
					CharID:   c.CharID,
					NewCoins: m.wallets[c.CharID],
				}
				delete(m.residents, c.CharID)
				delete(m.wallets, c.CharID)
			}
		}
	}
	close(m.outbound) // stop broadcastLoop
}

func (m *MapActor) handlePlace(c Cmd) {
	// 1. Validate coordinates (bounds + grid)
	if m.tileSize <= 0 {
		c.Reply <- CmdResult{Err: errors.New("tileSize must be positive")}
		return
	}
	if c.X%m.tileSize != 0 || c.Y%m.tileSize != 0 {
		c.Reply <- CmdResult{Err: errors.New("toạ độ không khớp snap grid")}
		return
	}
	if c.X < 0 || c.X >= m.mapW || c.Y < 0 || c.Y >= m.mapH {
		c.Reply <- CmdResult{Err: errors.New("toạ độ vượt quá giới hạn bản đồ")}
		return
	}

	key := [2]int{c.X, c.Y}
	if _, taken := m.occupied[key]; taken {
		c.Reply <- CmdResult{Err: ErrOccupied}
		return
	}

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
		CreatedAt:   time.Now(),
	}
	m.occupied[key] = p
	m.byID[p.ID] = p

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

	price := c.Item.Price
	coins, err := m.getOrLoadWallet(c.CharID)
	if err != nil {
		c.Reply <- CmdResult{Err: err}
		return
	}

	// MUTATE RAM = Final Decision
	newCoins := coins + price
	m.wallets[c.CharID] = newCoins

	delete(m.occupied, [2]int{p.X, p.Y})
	delete(m.byID, p.ID)

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
	coins, ok := m.wallets[charID]
	if ok {
		return coins, nil
	}
	// lazy load from DB
	dbCoins, err := m.charReader.GetCoins(context.Background(), charID)
	if err != nil {
		return 0, err
	}
	m.wallets[charID] = dbCoins
	return dbCoins, nil
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
