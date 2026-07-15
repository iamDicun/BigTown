package user

import "database/sql"

type UserModule struct {
	provider *Provider
}

func NewUserModule(db *sql.DB) *UserModule {
	return &UserModule{provider: NewProvider(db)}
}
