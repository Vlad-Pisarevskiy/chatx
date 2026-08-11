package service

import (
	"chatflow/internal/model"
	"chatflow/internal/repository"
	rand "crypto/rand"
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *repository.Repository
}

func (s *Service) RegisterUser(ctx context.Context, name, login, password string) error {

	var client = model.Client{
		Name:     name,
		Login:    login,
		Password: password,
	}

	if !s.correctLogin(client.Login) {
		return fmt.Errorf("incorrect login")
	}

	exists, err := s.db.LoginExists(ctx, client.Login)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("login is already exists")
	}

	if !s.correctName(client.Name) {
		return fmt.Errorf("incorrect name")
	}

	if !s.correctPassword(client.Password) {
		return fmt.Errorf("incorrect password")
	}

	hash, err := s.hashPassword(client.Password)
	if err != nil {
		return err
	}

	client.Password = hash

	if err = s.db.RegisterUser(ctx, client); err != nil {
		return err
	}

	return nil
}

func (s *Service) LoginUser(ctx context.Context, login, password string) (*model.Client, string, error) {

	if !s.correctLogin(login) {
		return nil, "", fmt.Errorf("incorrect login")
	}

	user, err := s.db.FindUserByLogin(ctx, login)
	if err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), hash); err != nil {
		return nil, "", err
	}

	rand.Text()

	return user, "", err
}

func (s *Service) hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (s *Service) correctName(name string) bool {
	return name != "" && len(name) > 2
}

func (s *Service) correctLogin(login string) bool {
	return login != "" && len(login) > 2
}

func (s *Service) correctPassword(password string) bool {
	return password != "" && len(password) > 5
}
