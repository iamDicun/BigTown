package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"backend/internal/security"

	"github.com/centrifugal/centrifuge"
)

const roomChannelPrefix = "room:"

type CentrifugeTransport struct {
	node    *centrifuge.Node
	handler *centrifuge.WebsocketHandler
}

func NewCentrifugeTransport(jwtSecret string) (*CentrifugeTransport, error) {
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
		}, nil
	})

	node.OnConnect(func(client *centrifuge.Client) {
		client.OnSubscribe(func(event centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			if !isRoomChannel(event.Channel) {
				cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
				return
			}
			cb(centrifuge.SubscribeReply{}, nil)
		})

		client.OnPublish(func(event centrifuge.PublishEvent, cb centrifuge.PublishCallback) {
			if !isRoomChannel(event.Channel) {
				cb(centrifuge.PublishReply{}, centrifuge.ErrorPermissionDenied)
				return
			}
			cb(centrifuge.PublishReply{}, nil)
		})
	})

	if err := node.Run(); err != nil {
		return nil, err
	}

	return &CentrifugeTransport{
		node: node,
		handler: centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
			CheckOrigin: allowDevOrigin,
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

func isRoomChannel(channel string) bool {
	return strings.HasPrefix(channel, roomChannelPrefix)
}

func allowDevOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:")
}
