package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"backend/internal/module/realtime/room"
	"backend/internal/module/realtime/usecase"
	"backend/internal/security"

	"github.com/centrifugal/centrifuge"
)

const roomChannelPrefix = "room:"
const personalChannelPrefix = "personal:"

type CentrifugeTransport struct {
	node    *centrifuge.Node
	handler *centrifuge.WebsocketHandler
}

func NewCentrifugeTransport(jwtSecret string, allowedOrigins []string, roomUsecase *usecase.RoomUsecase) (*CentrifugeTransport, error) {
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		return nil, err
	}

	node.OnConnecting(func(ctx context.Context, event centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		claims, err := security.ParseToken(event.Token, jwtSecret)
		if err != nil {
			if err == security.ErrTokenExpired {
				return centrifuge.ConnectReply{}, centrifuge.ErrorTokenExpired
			}
			return centrifuge.ConnectReply{}, centrifuge.ErrorUnauthorized
		}

		info, _ := json.Marshal(map[string]string{"role": claims.Role})

		return centrifuge.ConnectReply{
			Context: ctx,
			Credentials: &centrifuge.Credentials{
				UserID: claims.UserID,
				Info:   info,
			},
			// Server-side subscription: mọi client tự động nhận publication trên channel riêng
			// của mình mà không cần tự gọi subscribe. Dùng để gửi player_position_correction —
			// xem docs/Realtime-Room-State-Decisions.md mục 6 (quyết định: personal channel theo
			// userID, không trả trong response RPC).
			Subscriptions: map[string]centrifuge.SubscribeOptions{
				personalChannelPrefix + claims.UserID: {},
			},
		}, nil
	})

	node.OnConnect(func(client *centrifuge.Client) {
		client.OnSubscribe(func(event centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			handleSubscribe(node, roomUsecase, client, event, cb)
		})

		client.OnUnsubscribe(func(event centrifuge.UnsubscribeEvent) {
			if !isRoomChannel(event.Channel) {
				return
			}
			roomID := strings.TrimPrefix(event.Channel, roomChannelPrefix)
			handleLeaveRoom(node, roomUsecase, roomID, client.UserID(), client.ID())
		})

		client.OnDisconnect(func(event centrifuge.DisconnectEvent) {
			// MVP chỉ có 1 room/map tại một thời điểm (xem docs/Architecture.md mục 9.1), nên
			// disconnect luôn leave đúng room mặc định mà không cần track channel client đã
			// subscribe. Cần sửa lại nếu sau này có nhiều room cùng lúc.
			roomID, err := roomUsecase.DefaultRoomID(context.Background())
			if err != nil {
				return
			}
			handleLeaveRoom(node, roomUsecase, roomID, client.UserID(), client.ID())
		})

		// Gameplay/chat event giờ đều đi qua HTTP hoặc RPC/command để backend validate và tự
		// publish (xem docs/Realtime-Room-State-Decisions.md mục 8-9). Client không còn publish
		// trực tiếp vào room channel nữa nên OnPublish luôn từ chối.
		client.OnPublish(func(event centrifuge.PublishEvent, cb centrifuge.PublishCallback) {
			cb(centrifuge.PublishReply{}, centrifuge.ErrorPermissionDenied)
		})

		client.OnRPC(func(event centrifuge.RPCEvent, cb centrifuge.RPCCallback) {
			if event.Method != "player_move" {
				cb(centrifuge.RPCReply{}, centrifuge.ErrorMethodNotFound)
				return
			}
			handlePlayerMove(node, roomUsecase, client, event, cb)
		})
	})

	if err := node.Run(); err != nil {
		return nil, err
	}

	return &CentrifugeTransport{
		node: node,
		handler: centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
			CheckOrigin: allowOrigin(allowedOrigins),
		}),
	}, nil
}

func (t *CentrifugeTransport) Handler() http.Handler {
	return t.handler
}

func (t *CentrifugeTransport) PublishRoom(ctx context.Context, roomID string, event any) error {
	_ = ctx

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = t.node.Publish(roomChannelPrefix+roomID, data)
	return err
}

func handleSubscribe(node *centrifuge.Node, roomUsecase *usecase.RoomUsecase, client *centrifuge.Client, event centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
	if !isRoomChannel(event.Channel) {
		cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
		return
	}

	roomID := strings.TrimPrefix(event.Channel, roomChannelPrefix)

	snapshot, joinedPlayer, isFirstConnection, err := roomUsecase.JoinRoom(context.Background(), roomID, client.UserID(), client.ID())
	if err != nil {
		cb(centrifuge.SubscribeReply{}, centrifuge.ErrorInternal)
		return
	}

	snapshotData, err := json.Marshal(roomSnapshotEvent{
		Type:    "room_snapshot",
		RoomID:  roomID,
		Players: toRoomPlayerDTOs(snapshot.Players),
	})
	if err != nil {
		cb(centrifuge.SubscribeReply{}, centrifuge.ErrorInternal)
		return
	}

	// room_snapshot gửi riêng cho client vừa join qua Data của chính subscribe reply — không
	// broadcast lại cho cả room (mỗi client chỉ cần snapshot lúc join của chính mình).
	cb(centrifuge.SubscribeReply{
		Options: centrifuge.SubscribeOptions{Data: snapshotData},
	}, nil)

	if isFirstConnection {
		publishRoomEvent(node, roomID, playerJoinedEvent{
			Type:   "player_joined",
			RoomID: roomID,
			Player: toRoomPlayerDTO(*joinedPlayer),
		})
	}
}

func handleLeaveRoom(node *centrifuge.Node, roomUsecase *usecase.RoomUsecase, roomID string, userID string, clientID string) {
	player, err := roomUsecase.LeaveRoom(context.Background(), roomID, userID, clientID)
	if err != nil || player == nil {
		return
	}

	publishRoomEvent(node, roomID, playerLeftEvent{
		Type:        "player_left",
		RoomID:      roomID,
		CharacterID: player.CharacterID,
	})
}

func handlePlayerMove(node *centrifuge.Node, roomUsecase *usecase.RoomUsecase, client *centrifuge.Client, event centrifuge.RPCEvent, cb centrifuge.RPCCallback) {
	var cmd playerMoveCommand
	if err := json.Unmarshal(event.Data, &cmd); err != nil {
		cb(centrifuge.RPCReply{}, centrifuge.ErrorBadRequest)
		return
	}

	roomID, err := roomUsecase.DefaultRoomID(context.Background())
	if err != nil {
		cb(centrifuge.RPCReply{}, centrifuge.ErrorInternal)
		return
	}

	movement := room.PlayerMovement{
		X:         cmd.X,
		Y:         cmd.Y,
		Direction: room.Direction(cmd.Direction),
		Moving:    cmd.Moving,
	}

	updated, rejection, err := roomUsecase.MovePlayer(context.Background(), roomID, client.UserID(), movement)
	if err != nil {
		cb(centrifuge.RPCReply{}, centrifuge.ErrorInternal)
		return
	}

	if rejection != nil {
		sendPersonalEvent(node, client.UserID(), positionCorrectionEvent{
			Type:        "player_position_correction",
			CharacterID: rejection.CharacterID,
			Reason:      rejection.Reason,
			X:           rejection.X,
			Y:           rejection.Y,
		})
		// RPC vẫn ack thành công (server đã xử lý xong request) — correction đi riêng qua
		// personal channel theo đúng quyết định đã chốt, không trả trong response RPC này.
		cb(centrifuge.RPCReply{}, nil)
		return
	}

	publishRoomEvent(node, roomID, playerMoveEvent{
		Type:        "player_move",
		CharacterID: updated.CharacterID,
		X:           updated.X,
		Y:           updated.Y,
		Direction:   string(updated.Direction),
		Moving:      updated.Moving,
	})

	cb(centrifuge.RPCReply{}, nil)
}

func publishRoomEvent(node *centrifuge.Node, roomID string, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = node.Publish(roomChannelPrefix+roomID, data)
}

func sendPersonalEvent(node *centrifuge.Node, userID string, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = node.Publish(personalChannelPrefix+userID, data)
}

func isRoomChannel(channel string) bool {
	return strings.HasPrefix(channel, roomChannelPrefix)
}

func allowOrigin(allowedOrigins []string) func(*http.Request) bool {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}

		_, ok := allowed[origin]
		return ok
	}
}
