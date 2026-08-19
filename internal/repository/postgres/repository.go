package postgres

import (
	errors1 "chatflow/internal/app-errors"
	"chatflow/internal/model"
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(ctx context.Context, pool *pgxpool.Pool) *Repository {

	return &Repository{pool: pool}
}

func (r *Repository) RegisterUser(ctx context.Context, client model.User) error {

	_, err := r.pool.Exec(ctx, "INSERT INTO users(name, login, password) VALUES ($1, $2, $3)",
		client.Name, client.Login, client.Password)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) LoginExists(ctx context.Context, login string) (bool, error) {

	row := r.pool.QueryRow(ctx, "SELECT login FROM users WHERE login = $1", login)

	var dbLogin string

	err := row.Scan(&dbLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *Repository) FindUserByLogin(ctx context.Context, login string) (*model.User, error) {

	var client model.User

	row := r.pool.QueryRow(ctx, "SELECT id, name, login, password FROM users WHERE login=($1)", login)
	err := row.Scan(
		&client.ID,
		&client.Name,
		&client.Login,
		&client.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors1.ErrIncorrectLoginData
		}
		return nil, err
	}

	return &client, nil
}

func (r *Repository) AddToken(ctx context.Context, userID string, token []byte) error {

	_, err := r.pool.Exec(ctx, "INSERT INTO tokens(user_id, token_hash) VALUES ($1, $2)", userID, token)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) CheckToken(ctx context.Context, token []byte) (userID int, err error) {

	row := r.pool.QueryRow(ctx, "SELECT user_id FROM tokens WHERE token_hash=($1)", token)

	err = row.Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors1.ErrInvalidToken
		}
		return userID, err
	}

	return userID, nil
}

func (r *Repository) GetUsers(ctx context.Context) ([]model.UserFromDB, error) {

	var user model.UserFromDB
	var users []model.UserFromDB

	rows, err := r.pool.Query(ctx, "SELECT id, name, login FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		if err = rows.Scan(&user.ID, &user.Name, &user.Login); err != nil {
			log.Println(err)
		}
		users = append(users, user)
	}

	if rows.Err() != nil {
		return users, rows.Err()
	}

	return users, nil
}
