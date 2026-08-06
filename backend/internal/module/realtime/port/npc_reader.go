package port

import (
	"context"

	characterentity "backend/internal/module/character/entity"
)

type NPCReader interface {
	GetNPCSpawnsByMapCode(ctx context.Context, mapCode string) ([]characterentity.NPCSpawn, error)
}
