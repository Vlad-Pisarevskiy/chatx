package service

import (
	"chatflow/internal/model"
	"chatflow/internal/repository"
	"context"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *repository.Repository
}

func (s *Service) RegisterUser(ctx context.Context, client model.Client) error {

	var err error
	client.Password, err = s.hashPassword(client.Password)
	if err != nil {
		return err
	}

	if err = s.db.RegisterUser(ctx, client); err != nil {
		return err
	}

	return nil
}

func (s *Service) hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
