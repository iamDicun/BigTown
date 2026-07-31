package editor

import (
	"context"
	"database/sql"

	charPort "backend/internal/module/character/port"
	"backend/internal/module/editor/delivery"
	"backend/internal/module/editor/port"
	"backend/internal/module/editor/repository"
	"backend/internal/module/editor/usecase"
)

type characterReaderWrapper struct {
	charRepo charPort.CharacterRepository
}

func (w *characterReaderWrapper) GetByUserID(ctx context.Context, userID string) (*port.CharacterInfo, error) {
	c, err := w.charRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &port.CharacterInfo{
		ID:    c.ID,
		Coins: c.Coins,
	}, nil
}

type Provider struct {
	db       *sql.DB
	charRepo charPort.CharacterRepository
	publisher port.RoomPublisher

	repo          port.EditorRepository
	charReader    port.CharacterReader
	editorUsecase *usecase.EditorUsecase
	handler       *delivery.EditorHandler
}

func NewProvider(db *sql.DB, charRepo charPort.CharacterRepository, publisher port.RoomPublisher) *Provider {
	return &Provider{db: db, charRepo: charRepo, publisher: publisher}
}

func (p *Provider) Repository() port.EditorRepository {
	if p.repo == nil {
		p.repo = repository.NewEditorRepository(p.db)
	}
	return p.repo
}

func (p *Provider) CharacterReader() port.CharacterReader {
	if p.charReader == nil {
		p.charReader = &characterReaderWrapper{charRepo: p.charRepo}
	}
	return p.charReader
}

func (p *Provider) Usecase() *usecase.EditorUsecase {
	if p.editorUsecase == nil {
		p.editorUsecase = usecase.NewEditorUsecase(p.db, p.Repository(), p.CharacterReader(), p.publisher)
	}
	return p.editorUsecase
}

func (p *Provider) Handler() *delivery.EditorHandler {
	if p.handler == nil {
		p.handler = delivery.NewEditorHandler(p.Usecase())
	}
	return p.handler
}
