package usecase

import "backend/internal/module/user/port"

type UserUsecase struct {
	repo port.UserRepository
}

func NewUserUsecase(repo port.UserRepository) *UserUsecase {
	return &UserUsecase{repo: repo}
}
