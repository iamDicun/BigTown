package chat

import (
	"database/sql"

	"backend/internal/module/chat/delivery"
	"backend/internal/module/chat/port"
	"backend/internal/module/chat/repository"
	"backend/internal/module/chat/usecase"
)

type Provider struct {
	db         *sql.DB
	publisher  port.RoomPublisher
	characters port.CharacterReader

	repo    port.ChatRepository
	usecase *usecase.ChatUsecase
	handler *delivery.ChatHandler
}

func NewProvider(db *sql.DB, publisher port.RoomPublisher, characters port.CharacterReader) *Provider {
	return &Provider{db: db, publisher: publisher, characters: characters}
}

func (p *Provider) Repository() port.ChatRepository {
	if p.repo == nil {
		p.repo = repository.NewChatRepository(p.db)
	}
	return p.repo
}

func (p *Provider) Usecase() *usecase.ChatUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewChatUsecase(p.Repository(), p.publisher, p.characters)
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.ChatHandler {
	if p.handler == nil {
		p.handler = delivery.NewChatHandler(p.Usecase())
	}
	return p.handler
}
