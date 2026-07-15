package usecase

import "backend/internal/module/example/port"

type ExampleUsecase struct {
	repo port.ItemRepository
}

func NewExampleUsecase(repo port.ItemRepository) *ExampleUsecase {
	return &ExampleUsecase{repo: repo}
}
