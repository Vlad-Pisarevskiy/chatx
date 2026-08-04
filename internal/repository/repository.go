package repository

import (
	"chatflow/internal/model"
	"context"

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
		client.Name, client.Login, client.Email, client.Password)
	if err != nil {
		return err
	}

	return nil
}
