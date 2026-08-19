package memory

import (
	errors1 "chatflow/internal/app-errors"
	"chatflow/internal/model"
	"context"
	"sync"
)

type login string

type Repository struct {
	users map[login]*model.User
	mu    sync.Mutex
}

func NewRepository() *Repository {

	return &Repository{
		users: make(map[login]*model.User),
		mu:    sync.Mutex{},
	}
}

func (r *Repository) RegisterUser(ctx context.Context, client model.User) error {

	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[login(client.Login)] = &client

	return nil
}

func (r *Repository) LoginExists(ctx context.Context, userLogin string) (bool, error) {

	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.users[login(userLogin)]

	return ok, nil
}

func (r *Repository) FindUserByLogin(ctx context.Context, userLogin string) (*model.User, error) {

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[login(userLogin)]; !ok {
		return nil, errors1.ErrLoginNotFound
	}

	return r.users[login(userLogin)], nil
}

func (r *Repository) AddToken(ctx context.Context, userID string, token [32]byte) error {
	return nil
}

func (r *Repository) CheckToken(ctx context.Context, token []byte) (userID int, err error) {
	return 0, nil
}
