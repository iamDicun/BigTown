package realtime

import (
	"backend/internal/module/realtime/delivery"
	"backend/internal/module/realtime/transport"
	"backend/internal/module/realtime/usecase"
)

type Provider struct {
	jwtSecret string

	usecase   *usecase.RealtimeUsecase
	transport *transport.CentrifugeTransport
	handler   *delivery.RealtimeHandler
}

func NewProvider(jwtSecret string) *Provider {
	return &Provider{jwtSecret: jwtSecret}
}

func (p *Provider) Usecase() *usecase.RealtimeUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewRealtimeUsecase()
	}
	return p.usecase
}

func (p *Provider) Transport() *transport.CentrifugeTransport {
	if p.transport == nil {
		realtimeTransport, err := transport.NewCentrifugeTransport(p.jwtSecret)
		if err != nil {
			panic(err)
		}
		p.transport = realtimeTransport
	}
	return p.transport
}

func (p *Provider) Handler() *delivery.RealtimeHandler {
	if p.handler == nil {
		p.handler = delivery.NewRealtimeHandler(p.Usecase())
	}
	return p.handler
}
