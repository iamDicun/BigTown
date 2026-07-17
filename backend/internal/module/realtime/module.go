package realtime

import (
	"backend/internal/module/realtime/port"
	"backend/internal/module/realtime/transport"
)

type RealtimeModule struct {
	provider *Provider
}

func NewRealtimeModule(jwtSecret string, mapReader port.MapReader, characters port.CharacterResolver) *RealtimeModule {
	return &RealtimeModule{provider: NewProvider(jwtSecret, mapReader, characters)}
}

// Transport() cho phép module khác (chat, sau này realtime/room) tái dùng cùng
// CentrifugeTransport để publish vào room channel qua node.Publish thay vì client publish.
func (m *RealtimeModule) Transport() *transport.CentrifugeTransport {
	return m.provider.Transport()
}
