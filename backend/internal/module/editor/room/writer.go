package room

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
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

	// 2. Placements: INSERT batch idempotent + DELETE batch
	var placeOps []persistOp
	var deleteOps []persistOp
	for _, op := range batch {
		switch op.Kind {
		case opPlace:
			placeOps = append(placeOps, op)
		case opDelete:
			deleteOps = append(deleteOps, op)
		}
	}

	if len(placeOps) > 0 {
		if err := w.batchInsertPlacements(tx, placeOps); err != nil {
			return
		}
	}

	if len(deleteOps) > 0 {
		placeholders := make([]string, len(deleteOps))
		args := make([]interface{}, len(deleteOps))
		for i, op := range deleteOps {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = op.P.ID
		}
		query := "DELETE FROM map_placements WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		if _, err := tx.Exec(query, args...); err != nil {
			log.Printf("[writer] failed to batch delete %d placements: %v", len(deleteOps), err)
			return
		}
	}

	// 3. Reward events: batch append logs
	var evOps []persistOp
	for _, op := range batch {
		if op.EventType != "" {
			evOps = append(evOps, op)
		}
	}
	if len(evOps) > 0 {
		rows := make([]string, 0, len(evOps))
		args := make([]interface{}, 0, len(evOps)*3)
		for i, op := range evOps {
			b := i * 3
			rows = append(rows, fmt.Sprintf("($%d,$%d,$%d)", b+1, b+2, b+3))
			args = append(args, op.CharID, op.EventType, op.CoinDelta)
		}
		query := "INSERT INTO reward_events (character_id, event_type, coin_delta) VALUES " + strings.Join(rows, ",")
		if _, err := tx.Exec(query, args...); err != nil {
			log.Printf("[writer] failed to batch insert %d reward events: %v", len(evOps), err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[writer] failed to commit batch transaction: %v", err)
		return
	}
}

func (w *Writer) Close() {
	close(w.in) // trigger loop shutdown
	<-w.done    // wait for loop to clean up and flush
}

func (w *Writer) QueueLen() int { return len(w.in) }

func (w *Writer) batchInsertPlacements(tx *sql.Tx, ops []persistOp) error {
	if len(ops) == 0 {
		return nil
	}

	valueRows := make([]string, 0, len(ops))
	args := make([]interface{}, 0, len(ops)*7)
	for i, op := range ops {
		base := i * 7
		valueRows = append(valueRows,
			fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7))
		args = append(args, op.P.ID, op.P.MapID, op.P.CharacterID, op.P.ItemID, op.P.X, op.P.Y, op.P.Rotation)
	}

	query := fmt.Sprintf(
		"INSERT INTO map_placements (id, map_id, character_id, item_id, x, y, rotation) VALUES %s ON CONFLICT (id) DO NOTHING",
		strings.Join(valueRows, ","))

	_, err := tx.Exec(query, args...)
	if err != nil {
		log.Printf("[writer] failed to batch insert %d placements: %v", len(ops), err)
	}
	return err
}
