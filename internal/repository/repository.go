package repository

import (
	errors1 "chatflow/internal/app-errors"
	"chatflow/internal/model"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(ctx context.Context, pool *pgxpool.Pool) *Repository {

	return &Repository{pool: pool}
}

func (r *Repository) RegisterUser(ctx context.Context, client model.Client) error {

	_, err := r.pool.Exec(ctx, "INSERT INTO users(name, login, email, password) VALUES ($1, $2, $3, $4)",
		client.Name, client.Login, client.Password)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) LoginExists(ctx context.Context, login string) (bool, error) {

	row := r.pool.QueryRow(ctx, "SELECT login FROM users WHERE login=$1", login)

	var exists bool

	err := row.Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *Repository) FindUserByLogin(ctx context.Context, login string) (*model.Client, error) {

	var client model.Client

	row := r.pool.QueryRow(ctx, "SELECT id, name, login, password FROM users WHERE login=$1", login)

	err := row.Scan(&client)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors1.ErrLoginNotFound
		}
		return nil, err
	}

	return &client, nil
}

func (r *Repository) AddToken(ctx context.Context, userID string, token [32]byte) error {

	_, err := r.pool.Exec(ctx, "INSERT INTO tokens(user_id, token) VALUES $1, $2", userID, token)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) CheckToken(ctx context.Context, token [32]byte) (userID int, err error) {

	row := r.pool.QueryRow(ctx, "SELECT user_id FROM sessions WHERE token_hash=$1", token)

	err = row.Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors1.ErrInvalidToken
		}
		return userID, err
	}

	return userID, nil
}
