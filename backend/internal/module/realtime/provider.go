package realtime

import (
	"context"

	"backend/internal/module/realtime/delivery"
	"backend/internal/module/realtime/port"
	"backend/internal/module/realtime/room"
	"backend/internal/module/realtime/transport"
	"backend/internal/module/realtime/usecase"
)

type Provider struct {
	jwtSecret      string
	allowedOrigins []string
	mapReader      port.MapReader
	characters     port.CharacterResolver

	usecase     *usecase.RealtimeUsecase
	roomStore   room.RoomStore
	roomUsecase *usecase.RoomUsecase
	transport   *transport.CentrifugeTransport
	handler     *delivery.RealtimeHandler
}

func NewProvider(jwtSecret string, allowedOrigins []string, mapReader port.MapReader, characters port.CharacterResolver) *Provider {
	return &Provider{jwtSecret: jwtSecret, allowedOrigins: allowedOrigins, mapReader: mapReader, characters: characters}
}

func (p *Provider) Usecase() *usecase.RealtimeUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewRealtimeUsecase(p.mapReader)
	}
	return p.usecase
}

func (p *Provider) RoomStore() room.RoomStore {
	if p.roomStore == nil {
		p.roomStore = room.NewActorRoomStore()
	}
	return p.roomStore
}

func (p *Provider) RoomUsecase() *usecase.RoomUsecase {
	if p.roomUsecase == nil {
		p.roomUsecase = usecase.NewRoomUsecase(p.RoomStore(), p.characters, p.mapReader)
	}
	return p.roomUsecase
}

func (p *Provider) Transport() *transport.CentrifugeTransport {
	if p.transport == nil {
		realtimeTransport, err := transport.NewCentrifugeTransport(p.jwtSecret, p.allowedOrigins, p.RoomUsecase())
		if err != nil {
			panic(err)
		}
		p.transport = realtimeTransport

		// Liên kết callback phát tin theo nhịp (Server-side Tick Broadcast) cho ActorRoomStore
		if actorStore, ok := p.RoomStore().(*room.ActorRoomStore); ok {
			actorStore.SetBroadcastFunc(func(roomID string, event any) {
				_ = p.transport.PublishRoom(context.Background(), roomID, event)
			})
		}
	}
	return p.transport
}

func (p *Provider) Handler() *delivery.RealtimeHandler {
	if p.handler == nil {
		p.handler = delivery.NewRealtimeHandler(p.Usecase())
	}
	return p.handler
}

func (p *Provider) AddEventListener(l port.RoomEventListener) {
	p.RoomUsecase().AddEventListener(l)
}
