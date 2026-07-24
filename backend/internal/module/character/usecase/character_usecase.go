package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"backend/internal/apperror"
	"backend/internal/module/character/entity"
	"backend/internal/module/character/port"
)

type CharacterUsecase struct {
	db             *sql.DB
	repo           port.CharacterRepository
	users          port.UserReader
	defaultMapCode string

	mapCacheMu sync.RWMutex
	mapCache   *entity.MapInfo
	mapByCode  map[string]*entity.MapInfo
}

type SpritesheetConfig struct {
	FrameWidth    int `json:"frame_width"`
	FrameHeight   int `json:"frame_height"`
	Columns       int `json:"columns"`
	RowIdleDown   int `json:"row_idle_down"`
	RowWalkDown   int `json:"row_walk_down"`
	RowIdleUp     int `json:"row_idle_up"`
	RowWalkUp     int `json:"row_walk_up"`
	RowWalkSide   int `json:"row_walk_side"`
	WalkFrameRate int `json:"walk_frame_rate"`
	IdleFrameRate int `json:"idle_frame_rate"`
}

type CharacterOption struct {
	Name         string
	BaseAssetKey string
	PreviewURL   string
	Spritesheet  SpritesheetConfig
}

var explorerConfig = SpritesheetConfig{
	FrameWidth: 32, FrameHeight: 32,
	Columns:     6,
	RowIdleDown: 0, RowWalkDown: 1,
	RowIdleUp: 2, RowWalkUp: 3,
	RowWalkSide:   4,	
	WalkFrameRate: 8, IdleFrameRate: 4,
}

var knightConfig = SpritesheetConfig{
	FrameWidth: 110, FrameHeight: 110,
	Columns:     4,
	RowIdleDown: 0, RowWalkDown: 1,
	RowIdleUp: 2, RowWalkUp: 3,
	RowWalkSide:   4,
	WalkFrameRate: 8, IdleFrameRate: 4,
}

var characterOptions = []CharacterOption{
	{Name: "Nhà thám hiểm", BaseAssetKey: "player", PreviewURL: "/assets/player/Player.png", Spritesheet: explorerConfig},
	{Name: "Hiệp sĩ", BaseAssetKey: "knight", PreviewURL: "/assets/player/knight.png", Spritesheet: knightConfig},
}

func NewCharacterUsecase(db *sql.DB, repo port.CharacterRepository, users port.UserReader, defaultMapCode string) *CharacterUsecase {
	return &CharacterUsecase{db: db, repo: repo, users: users, defaultMapCode: defaultMapCode}
}

func (u *CharacterUsecase) GetByUserID(ctx context.Context, userID string) (*entity.Character, error) {
	character, err := u.repo.FindByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("Chưa có nhân vật cho user này", err)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return u.syncMap(ctx, character)
}

func (u *CharacterUsecase) ListOptions() []CharacterOption {
	return characterOptions
}

func (u *CharacterUsecase) CreateForUser(ctx context.Context, userID string, name string, baseAssetKey string) (*entity.Character, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperror.BadRequest("Tên nhân vật không được để trống", nil)
	}
	if len(name) > 80 {
		return nil, apperror.BadRequest("Tên nhân vật quá dài", nil)
	}

	baseAssetKey = strings.TrimSpace(baseAssetKey)
	if !isAllowedBaseAssetKey(baseAssetKey) {
		return nil, apperror.BadRequest("Nhân vật đã chọn không hợp lệ", fmt.Errorf("invalid base_asset_key: %s", baseAssetKey))
	}

	if _, err := u.repo.FindByUserID(ctx, userID); err == nil {
		return nil, apperror.BadRequest("User đã có nhân vật", nil)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.Internal(err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	created, err := u.repo.CreateWithTx(ctx, tx, userID, name, baseAssetKey)
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
//
// Cache trong process: map mặc định không đổi lúc server đang chạy (chỉ đổi qua deploy lại với
// GAME_DEFAULT_MAP_CODE/DB khác), nhưng hàm này bị gọi lại trên mỗi player_move RPC (10 lần/giây/
// player đang di chuyển) để lấy bounds validate — không cache sẽ khiến mỗi tick di chuyển tốn thêm
// 1 round-trip DB, gây trễ nặng khi FE/BE lệch region hoặc DB chậm (xem RoomUsecase.MovePlayer).
func (u *CharacterUsecase) GetDefaultMap(ctx context.Context) (*entity.MapInfo, error) {
	u.mapCacheMu.RLock()
	cached := u.mapCache
	u.mapCacheMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	mapInfo, err := u.repo.FindMapByCode(ctx, u.defaultMapCode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("Chưa seed map mặc định: "+u.defaultMapCode, err)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	u.mapCacheMu.Lock()
	u.mapCache = mapInfo
	u.mapCacheMu.Unlock()

	return mapInfo, nil
}

func (u *CharacterUsecase) GetMapByCode(ctx context.Context, code string) (*entity.MapInfo, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return u.GetDefaultMap(ctx)
	}

	u.mapCacheMu.RLock()
	if u.mapByCode != nil {
		if cached, ok := u.mapByCode[code]; ok {
			u.mapCacheMu.RUnlock()
			return cached, nil
		}
	}
	u.mapCacheMu.RUnlock()

	mapInfo, err := u.repo.FindMapByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("Không tìm thấy map: "+code, err)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	u.mapCacheMu.Lock()
	if u.mapByCode == nil {
		u.mapByCode = make(map[string]*entity.MapInfo)
	}
	u.mapByCode[code] = mapInfo
	u.mapCacheMu.Unlock()

	return mapInfo, nil
}

func isAllowedBaseAssetKey(baseAssetKey string) bool {
	for _, option := range characterOptions {
		if option.BaseAssetKey == baseAssetKey {
			return true
		}
	}
	return false
}
