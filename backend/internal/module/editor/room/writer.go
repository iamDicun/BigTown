package room

import (
	"context"
	"database/sql"
	"log"
	"time"

	"backend/internal/module/editor/entity"
)

type opKind int

const (
	opPlace opKind = iota
	opDelete
	opFlushWallet
)

type persistOp struct {
	Kind      opKind
	P         *entity.Placement
	CharID    string
	NewCoins  int
	CoinDelta int
	EventType string
}

type Writer struct {
	db    *sql.DB
	rm    *RoomManager
	in    chan persistOp
	done  chan struct{}
	every time.Duration
	max   int
}

func NewWriter(db *sql.DB, rm *RoomManager) *Writer {
	w := &Writer{
		db:    db,
		rm:    rm,
		in:    make(chan persistOp, 10000),
		done:  make(chan struct{}),
		every: 1 * time.Second,
		max:   512,
	}
	go w.loop()
	return w
}

func (w *Writer) loop() {
	t := time.NewTicker(w.every)
	defer t.Stop()
	batch := make([]persistOp, 0, w.max)

	for {
		select {
		case op, ok := <-w.in:
			if !ok {
				// in channel closed
				w.flush(batch)
				close(w.done)
				return
			}
			batch = append(batch, op)
			if len(batch) >= w.max {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-t.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (w *Writer) flush(batch []persistOp) {
	if len(batch) == 0 {
		return
	}

	tx, err := w.db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Printf("[writer] failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// 1. Coalesce wallet coins (last-write-wins)
	latestCoins := make(map[string]int)
	for _, op := range batch {
		if op.CharID != "" {
			latestCoins[op.CharID] = op.NewCoins
		}
	}

	for charID, coins := range latestCoins {
		query := `UPDATE characters SET coins = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
		_, err := tx.Exec(query, coins, charID)
		if err != nil {
			log.Printf("[writer] failed to update character %s coins to %d: %v", charID, coins, err)
			return
		}
	}

	// 2. Placements: INSERT idempotent + DELETE
	for _, op := range batch {
		switch op.Kind {
		case opPlace:
			query := `INSERT INTO map_placements (id, map_id, character_id, item_id, x, y)
			          VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`
			_, err = tx.Exec(query, op.P.ID, op.P.MapID, op.P.CharacterID, op.P.ItemID, op.P.X, op.P.Y)
		case opDelete:
			query := `DELETE FROM map_placements WHERE id = $1`
			_, err = tx.Exec(query, op.P.ID)
		}
		if err != nil {
			log.Printf("[writer] failed to execute placement operation (kind: %v, id: %v): %v", op.Kind, op.P.ID, err)
			return
		}
	}

	// 3. Reward events: append logs
	for _, op := range batch {
		if op.EventType == "" {
			continue
		}
		query := `INSERT INTO reward_events (character_id, event_type, coin_delta) VALUES ($1, $2, $3)`
		_, err = tx.Exec(query, op.CharID, op.EventType, op.CoinDelta)
		if err != nil {
			log.Printf("[writer] failed to insert reward event for %s (delta: %d): %v", op.CharID, op.CoinDelta, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[writer] failed to commit batch transaction: %v", err)
		return
	}

	// Evict online cache after database changes are fully committed
	for charID := range latestCoins {
		w.rm.EvictOnlineCoins(charID)
	}
}

func (w *Writer) Close() {
	close(w.in) // trigger loop shutdown
	<-w.done    // wait for loop to clean up and flush
}
