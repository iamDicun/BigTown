package room

import (
	"context"
	"sync"
	"time"
)

var _ RoomStore = (*ActorRoomStore)(nil)

type command struct {
	fn func(room *GameRoom, actor *roomActor)
}

type roomActor struct {
	id        string
	state     *GameRoom
	cmds      chan command
	quit      chan struct{}
	broadcast func(event any)
	dirty     map[string]struct{}
}

func newRoomActor(id string, broadcast func(event any)) *roomActor {
	a := &roomActor{
		id: id,
		state: &GameRoom{
			ID:            id,
			Players:       make(map[string]*RoomPlayer),
			Clients:       make(map[string]map[string]struct{}),
			PlayersByUser: make(map[string]string),
		},
		cmds:      make(chan command, 2048), // Tăng buffer để chứa lượng lớn RPC di chuyển
		quit:      make(chan struct{}),
		broadcast: broadcast,
		dirty:     make(map[string]struct{}),
	}
	go a.loop()
	return a
}

func (a *roomActor) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case cmd := <-a.cmds:
			cmd.fn(a.state, a)
		case <-ticker.C:
			a.flushDirty()
		case <-a.quit:
			return
		}
	}
}

func (a *roomActor) flushDirty() {
	if len(a.dirty) == 0 {
		return
	}

	type playerState struct {
		CharacterID string `json:"characterId"`
		X           int    `json:"x"`
		Y           int    `json:"y"`
		Direction   string `json:"direction"`
		Moving      bool   `json:"moving"`
	}
	type roomStateEvent struct {
		Type    string        `json:"type"`
		Players []playerState `json:"players"`
	}

	event := roomStateEvent{
		Type:    "room_state",
		Players: make([]playerState, 0, len(a.dirty)),
	}

	for cid := range a.dirty {
		p, ok := a.state.Players[cid]
		if ok {
			event.Players = append(event.Players, playerState{
				CharacterID: p.CharacterID,
				X:           p.X,
				Y:           p.Y,
				Direction:   string(p.Direction),
				Moving:      p.Moving,
			})
		}
	}

	// Xóa sạch map dirty
	a.dirty = make(map[string]struct{})

	if a.broadcast != nil {
		a.broadcast(event)
	}
}

func (a *roomActor) dispatch(ctx context.Context, fn func(*GameRoom, *roomActor)) error {
	select {
	case a.cmds <- command{fn: fn}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-a.quit:
		return ErrPlayerNotFound
	}
}

type ActorRoomStore struct {
	mu            sync.RWMutex
	actors        map[string]*roomActor
	broadcastFunc func(roomID string, event any)
}

func NewActorRoomStore() *ActorRoomStore {
	return &ActorRoomStore{actors: make(map[string]*roomActor)}
}

func (s *ActorRoomStore) SetBroadcastFunc(fn func(roomID string, event any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.broadcastFunc = fn
}

func (s *ActorRoomStore) actorFor(roomID string, create bool) *roomActor {
	s.mu.RLock()
	a, ok := s.actors[roomID]
	s.mu.RUnlock()
	if ok {
		return a
	}
	if !create {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok = s.actors[roomID]; ok {
		return a
	}

	// Đóng gói broadcastFunc động bảo vệ race-condition
	broadcastWrapper := func(event any) {
		s.mu.RLock()
		fn := s.broadcastFunc
		s.mu.RUnlock()
		if fn != nil {
			fn(roomID, event)
		}
	}

	a = newRoomActor(roomID, broadcastWrapper)
	s.actors[roomID] = a
	return a
}

func (s *ActorRoomStore) JoinRoom(ctx context.Context, roomID string, player RoomPlayer) (*RoomSnapshot, *RoomPlayer, bool, error) {
	a := s.actorFor(roomID, true)

	type result struct {
		snap    *RoomSnapshot
		joined  *RoomPlayer
		isFirst bool
	}
	reply := make(chan result, 1)

	err := a.dispatch(ctx, func(gr *GameRoom, actor *roomActor) {
		if existing, ok := gr.Players[player.CharacterID]; ok {
			existing.LastSeenAt = time.Now()
			addClientLocked(gr, player.CharacterID, player.ClientID)
			setUserIndexLocked(gr, existing.UserID, player.CharacterID)
			cp := *existing
			reply <- result{snapshotLocked(gr), &cp, false}
			return
		}
		p := player
		p.LastSeenAt = time.Now()
		gr.Players[p.CharacterID] = &p
		addClientLocked(gr, p.CharacterID, p.ClientID)
		setUserIndexLocked(gr, p.UserID, p.CharacterID)
		cp := p
		reply <- result{snapshotLocked(gr), &cp, true}
	})
	if err != nil {
		return nil, nil, false, err
	}

	select {
	case r := <-reply:
		return r.snap, r.joined, r.isFirst, nil
	case <-ctx.Done():
		return nil, nil, false, ctx.Err()
	}
}

func (s *ActorRoomStore) LeaveRoom(ctx context.Context, roomID string, characterID string, clientID string) (*RoomPlayer, bool, error) {
	a := s.actorFor(roomID, false)
	if a == nil {
		return nil, false, nil
	}

	type result struct {
		player  *RoomPlayer
		removed bool
	}
	reply := make(chan result, 1)

	err := a.dispatch(ctx, func(gr *GameRoom, actor *roomActor) {
		player, ok := gr.Players[characterID]
		if !ok {
			reply <- result{nil, false}
			return
		}
		if clients, ok := gr.Clients[characterID]; ok {
			delete(clients, clientID)
			if len(clients) > 0 {
				cp := *player
				reply <- result{&cp, false}
				return
			}
		}
		cp := *player
		delete(gr.Players, characterID)
		delete(gr.Clients, characterID)
		delete(gr.PlayersByUser, player.UserID)
		reply <- result{&cp, true}
	})
	if err != nil {
		return nil, false, err
	}

	select {
	case r := <-reply:
		return r.player, r.removed, nil
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}

func (s *ActorRoomStore) GetSnapshot(ctx context.Context, roomID string) (*RoomSnapshot, error) {
	a := s.actorFor(roomID, false)
	if a == nil {
		return &RoomSnapshot{RoomID: roomID}, nil
	}
	reply := make(chan *RoomSnapshot, 1)
	err := a.dispatch(ctx, func(gr *GameRoom, actor *roomActor) {
		reply <- snapshotLocked(gr)
	})
	if err != nil {
		return nil, err
	}
	select {
	case r := <-reply:
		return r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *ActorRoomStore) GetPlayer(ctx context.Context, roomID string, characterID string) (*RoomPlayer, error) {
	return s.getPlayerBy(ctx, roomID, func(gr *GameRoom, actor *roomActor) (*RoomPlayer, bool) {
		p, ok := gr.Players[characterID]
		return p, ok
	})
}

func (s *ActorRoomStore) GetPlayerByUserID(ctx context.Context, roomID string, userID string) (*RoomPlayer, error) {
	return s.getPlayerBy(ctx, roomID, func(gr *GameRoom, actor *roomActor) (*RoomPlayer, bool) {
		cid, ok := gr.PlayersByUser[userID]
		if !ok {
			return nil, false
		}
		p, ok := gr.Players[cid]
		return p, ok
	})
}

func (s *ActorRoomStore) getPlayerBy(ctx context.Context, roomID string, pick func(*GameRoom, *roomActor) (*RoomPlayer, bool)) (*RoomPlayer, error) {
	a := s.actorFor(roomID, false)
	if a == nil {
		return nil, ErrPlayerNotFound
	}
	reply := make(chan *RoomPlayer, 1)
	err := a.dispatch(ctx, func(gr *GameRoom, actor *roomActor) {
		if p, ok := pick(gr, actor); ok {
			cp := *p
			reply <- &cp
		} else {
			reply <- nil
		}
	})
	if err != nil {
		return nil, err
	}
	select {
	case p := <-reply:
		if p == nil {
			return nil, ErrPlayerNotFound
		}
		return p, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *ActorRoomStore) MovePlayer(ctx context.Context, roomID string, characterID string, m PlayerMovement) (*RoomPlayer, error) {
	a := s.actorFor(roomID, false)
	if a == nil {
		return nil, ErrPlayerNotFound
	}
	reply := make(chan *RoomPlayer, 1)
	err := a.dispatch(ctx, func(gr *GameRoom, actor *roomActor) {
		p, ok := gr.Players[characterID]
		if !ok {
			reply <- nil
			return
		}
		p.X, p.Y = m.X, m.Y
		p.Direction, p.Moving = m.Direction, m.Moving
		p.LastSeenAt = time.Now()
		
		// Đánh dấu dirty để phát tin theo nhịp (Tick Broadcast)
		actor.dirty[characterID] = struct{}{}
		
		cp := *p
		reply <- &cp
	})
	if err != nil {
		return nil, err
	}
	select {
	case p := <-reply:
		if p == nil {
			return nil, ErrPlayerNotFound
		}
		return p, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
