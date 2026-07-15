package example

import "database/sql"

type ExampleModule struct {
	provider *Provider
}

func NewExampleModule(db *sql.DB) *ExampleModule {
	return &ExampleModule{provider: NewProvider(db)}
}
