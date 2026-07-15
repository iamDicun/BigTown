package app

import (
	"database/sql"

	"backend/internal/platform/config"
)

type Container struct {
	Config *config.Config
	DB     *sql.DB
}
