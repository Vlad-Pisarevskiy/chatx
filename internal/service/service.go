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

type RegisterInput struct {
	Name     string
	Login    string
	Password string
}

func New(repo *postgres.Repository) *Service {
	return &Service{db: repo}
}

func (s *Service) RegisterUser(ctx context.Context, input RegisterInput) error {

	if err := s.validateRegister(ctx, input); err != nil {
		return err
	}

	hash, err := hashPassword(input.Password)
	if err != nil {
		return err
	}

	input.Password = hash

	user := model.User{
		Name:     input.Name,
		Login:    input.Login,
		Password: input.Password,
	}

	if err = s.db.RegisterUser(ctx, user); err != nil {
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

func (s *Service) CleanupTokens(ctx context.Context) (int64, error) {

	return s.db.DeleteExpiredTokens(ctx)

}

func (s *Service) Logout(ctx context.Context, token string) error {

	tokenHash := sha256.Sum256([]byte(token))
	if err := s.db.RemoveToken(ctx, tokenHash[:]); err != nil {
		return err
	}

	return nil
}

func (s *Service) FindUserByID(ctx context.Context, id int) (*model.UserFromDB, error) {

	user, err := s.db.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) GetUsers(ctx context.Context) ([]*model.UserFromDB, error) {

	return s.db.GetUsers(ctx)

}

func (s *Service) GetUsersExcept(ctx context.Context, id int) ([]*model.UserFromDB, error) {

	return s.db.GetUsersExcept(ctx, id)
}

func (s *Service) GetOrCreateChat(ctx context.Context, peerID, sender int) (int, error) {

	chatID, ok, err := s.db.ChatExists(ctx, sender, peerID)
	if err != nil {
		return emptyID, err
	}

	if ok {
		return chatID, nil
	}

	chatID, err = s.db.StartChat(ctx, sender, peerID)
	if err != nil {
		return emptyID, err
	}

	return chatID, nil
}

func (s *Service) SendMessage(ctx context.Context, message protocol.Send, from int) error {

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

func (s *Service) LoadMessages(ctx context.Context, from, to int) ([]model.Message, error) {

	return s.db.LoadMessages(ctx, from, to)
}
