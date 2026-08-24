package service

import (
	errors1 "chatflow/internal/app-errors"
	"chatflow/internal/model"
	"chatflow/internal/protocol"
	"chatflow/internal/repository/postgres"
	"context"
	rand "crypto/rand"
	"crypto/sha256"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *postgres.Repository
}

func NewService(repo *postgres.Repository) *Service {
	return &Service{db: repo}
}

func (s *Service) RegisterUser(ctx context.Context, name, login, password string) error {

	var user = model.User{
		Name:     name,
		Login:    login,
		Password: password,
	}

	if err := s.validateUserRegister(ctx, user); err != nil {
		return err
	}

	hash, err := s.hashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hash

	if err = s.db.RegisterUser(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *Service) validateUserRegister(ctx context.Context, user model.User) error {

	if err := correctLogin(user.Login); err != nil {
		return err
	}

	if err := s.db.LoginExists(ctx, user.Login); err != nil {
		return err
	}

	if err := correctName(user.Name); err != nil {
		return err
	}

	if err := correctPassword(user.Password); err != nil {
		return err
	}

	return nil
}

func (s *Service) LoginUser(ctx context.Context, login, password string) (*model.User, string, error) {

	if err := correctLogin(login); err != nil {
		return nil, emptyToken, err
	}

	user, err := s.db.FindUserByLogin(ctx, login)
	if err != nil {
		return nil, emptyToken, err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", err
	}

	token := rand.Text()
	tokenHash := sha256.Sum256([]byte(token))

	if err = s.db.AddToken(ctx, user.ID, tokenHash[:]); err != nil {
		return nil, emptyToken, err
	}

	return user, token, err
}

func (s *Service) CheckToken(ctx context.Context, token string) (int, error) {

	tokenHash := sha256.Sum256([]byte(token))
	id, err := s.db.CheckToken(ctx, tokenHash[:])
	if err != nil {
		return emptyID, err
	}

	return id, nil
}

func (s *Service) FindUserByID(ctx context.Context, id int) (*model.UserFromDB, error) {

	user, err := s.db.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) GetUsers(ctx context.Context) ([]*model.UserFromDB, error) {

	users, err := s.db.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *Service) GetUsersExcept(ctx context.Context, id int) ([]*model.UserFromDB, error) {

	return s.db.GetUsersExcept(ctx, id)
}

func (s *Service) SendMessage(ctx context.Context, message protocol.SentMessage, from int) error {

	chatID, ok, err := s.db.ChatExists(ctx, from, message.To)
	if err != nil {
		return err
	}

	if !ok {
		chatID, err = s.db.StartChat(ctx, from, message.To)
		if err != nil {
			return err
		}
	}

	if err = s.db.SendMessage(ctx, chatID, from, message.Message); err != nil {
		return err
	}

	return nil
}

func (s *Service) hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return emptyHash, err
	}
	return string(hashedPassword), nil
}

func correctName(name string) error {

	if name == emptyName {
		return errors1.ErrEmptyName
	}

	if len(name) > maxNameLength {
		return errors1.ErrLongName
	}

	return nil
}

func correctLogin(login string) error {

	if login == emptyLogin {
		return errors1.ErrEmptyLogin
	}

	if len(login) < minLoginLength {
		return errors1.ErrShortLogin
	}

	if len(login) > maxLoginLength {
		return errors1.ErrLongLogin
	}

	return nil
}

func correctPassword(password string) error {

	if password == emptyPassword {
		return errors1.ErrEmptyPassword
	}

	if len(password) < minPasswordLength {
		return errors1.ErrShortPassword
	}

	if len(password) > maxPasswordLength {
		return errors1.ErrLongPassword
	}

	return nil
}
