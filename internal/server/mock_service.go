package server

import (
	errors1 "chatflow/internal/app-errors"
	"chatflow/internal/model"
	"chatflow/internal/service"
	"context"
)

type MockService struct {
	db service.Repository
}

func NewMockService(db service.Repository) *MockService {

	return &MockService{db: db}
}

func (m *MockService) RegisterUser(ctx context.Context, name, login, password string) error {

	var client = model.Client{
		Name:     name,
		Login:    login,
		Password: password,
	}

	if err := m.db.RegisterUser(ctx, client); err != nil {
		return err
	}

	return nil
}

func (m *MockService) LoginUser(ctx context.Context, login, password string) (*model.Client, string, error) {

	user, err := m.db.FindUserByLogin(ctx, login)
	if err != nil {
		return nil, "", err
	}

	if user.Password != password {
		return nil, "", errors1.ErrIncorrectLoginData
	}

	return user, "", nil

}

func (m *MockService) CheckToken(ctx context.Context, token string) (int, error) {

	return 0, nil
}
