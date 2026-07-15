package repository

import (
	"context"
	"database/sql"
	"sync"

	"backend/internal/module/example/entity"
	"backend/internal/module/example/port"

	"github.com/google/uuid"
)

// MemoryItemRepository là bản implement in-memory, không đụng Postgres, để module
// boilerplate này chạy được ngay mà không cần migrate schema. Repository thật của một
// module nghiệp vụ phải dùng *sql.DB giống backend/internal/module/device/repository/device_repo.go.
type MemoryItemRepository struct {
	mu    sync.RWMutex
	items map[string]entity.Item
}

func NewMemoryItemRepository() *MemoryItemRepository {
	seed := []entity.Item{
		{ID: uuid.NewString(), Name: "Sample laptop", Status: "Available"},
		{ID: uuid.NewString(), Name: "Sample monitor", Status: "Available"},
	}

	items := make(map[string]entity.Item, len(seed))
	for _, it := range seed {
		items[it.ID] = it
	}

	return &MemoryItemRepository{items: items}
}

var _ port.ItemRepository = (*MemoryItemRepository)(nil)

func (r *MemoryItemRepository) FindAll(ctx context.Context) ([]entity.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]entity.Item, 0, len(r.items))
	for _, it := range r.items {
		items = append(items, it)
	}
	return items, nil
}

func (r *MemoryItemRepository) FindByID(ctx context.Context, id string) (*entity.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	it, ok := r.items[id]
	if !ok {
		// Trả sentinel error giống repository thật (sql.ErrNoRows) để usecase map
		// sang apperror.NotFound đúng theo luồng error handling chuẩn (mục 7 ARCHITECTURE_GUIDE.md).
		return nil, sql.ErrNoRows
	}
	return &it, nil
}

func (r *MemoryItemRepository) Create(ctx context.Context, name string) (*entity.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	it := entity.Item{ID: uuid.NewString(), Name: name, Status: "Available"}
	r.items[it.ID] = it
	return &it, nil
}
