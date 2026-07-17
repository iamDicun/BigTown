package usecase

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/apperror"
	"backend/internal/module/character/entity"
	"backend/internal/module/character/port"
)

type CharacterUsecase struct {
	db             *sql.DB
	repo           port.CharacterRepository
	defaultMapCode string
}

func NewCharacterUsecase(db *sql.DB, repo port.CharacterRepository, defaultMapCode string) *CharacterUsecase {
	return &CharacterUsecase{db: db, repo: repo, defaultMapCode: defaultMapCode}
}

func (u *CharacterUsecase) GetByUserID(ctx context.Context, userID string) (*entity.Character, error) {
	character, err := u.repo.FindByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("Chưa có nhân vật cho user này", err)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return character, nil
}

// GetOrCreateForUser là safety net cho các user được tạo trước khi Register/TeamsLogin tự tạo
// character mặc định (xem auth/usecase/register.go, teams_login.go). Các module khác (chat,
// realtime) nên gọi hàm này thay vì GetByUserID để không bao giờ bị chặn bởi lỗi NotFound.
func (u *CharacterUsecase) GetOrCreateForUser(ctx context.Context, userID string, defaultName string) (*entity.Character, error) {
	character, err := u.repo.FindByUserID(ctx, userID)
	if err == nil {
		return u.syncMap(ctx, character)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.Internal(err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	created, err := u.repo.CreateDefaultWithTx(ctx, tx, userID, defaultName)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, apperror.Internal(err)
	}

	return created, nil
}

// syncMap đồng bộ map_id của character theo GAME_DEFAULT_MAP_CODE mỗi lần được load (login/bootstrap).
// Nhờ vậy đổi map mặc định áp dụng cho cả user cũ lẫn mới ngay lần login kế tiếp, không cần migrate
// dữ liệu riêng — xem docs/Architecture.md mục 9.1.
func (u *CharacterUsecase) syncMap(ctx context.Context, character *entity.Character) (*entity.Character, error) {
	mapID, err := u.repo.SyncMapID(ctx, character.ID, character.MapID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	character.MapID = mapID
	return character, nil
}

// GetDefaultMap trả metadata map mặc định hiện hành (GAME_DEFAULT_MAP_CODE) — dùng bởi
// realtime bootstrap để trả tilemap/tileset/spawn point thật cho frontend, không hardcode.
func (u *CharacterUsecase) GetDefaultMap(ctx context.Context) (*entity.MapInfo, error) {
	mapInfo, err := u.repo.FindMapByCode(ctx, u.defaultMapCode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("Chưa seed map mặc định: "+u.defaultMapCode, err)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return mapInfo, nil
}
