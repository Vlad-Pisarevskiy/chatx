package postgres

import (
	errors1 "chatflow/internal/app-errors"
	"chatflow/internal/model"
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

const zeroInt = 0

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

	row := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)", login)

	var exists bool

	err := row.Scan(&exists)
	if err != nil {
		return exists, err
	}

	return exists, nil
}

func (r *Repository) FindUserByLogin(ctx context.Context, login string) (*model.User, error) {

	var user model.User

	row := r.pool.QueryRow(ctx, "SELECT id, name, login, password FROM users WHERE login=($1)", login)
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Login,
		&user.Password); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors1.ErrIncorrectLoginData
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) FindUserByID(ctx context.Context, id int) (*model.UserFromDB, error) {

	var user model.UserFromDB

	row := r.pool.QueryRow(ctx, "SELECT id, name, login FROM users WHERE id=$1", id)
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Login); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors1.ErrIncorrectLoginData
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUsersExcept(ctx context.Context, id int) ([]*model.UserFromDB, error) {

	var users []*model.UserFromDB

	rows, err := r.pool.Query(ctx, "SELECT id, name, login FROM users WHERE id <> $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users, err = scanRows(rows, users)
	if err != nil {
		if users != nil {
			return users, nil
		}
		return nil, err
	}

	return users, nil
}

func (r *Repository) AddToken(ctx context.Context, userID int, token []byte) error {

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

func (r *Repository) GetUsers(ctx context.Context) ([]*model.UserFromDB, error) {

	var users []*model.UserFromDB

	rows, err := r.pool.Query(ctx, "SELECT id, name, login FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users, err = scanRows(rows, users)
	if err != nil {
		if users != nil {
			return users, nil
		}
		return nil, err
	}

	return users, nil
}

func (r *Repository) ChatExists(ctx context.Context, from int, to int) (int, bool, error) {

	row := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chats c 
    		JOIN users_chats u ON c.id = u.chat_id
       		WHERE u.user_id IN ($1, $2)), id FROM chats`, from, to)

	var exists bool
	var chatID int

	if err := row.Scan(&exists, &chatID); err != nil {
		return chatID, exists, err
	}

	return chatID, exists, nil
}

func (r *Repository) StartChat(ctx context.Context, from, to int) (int, error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return zeroInt, err
	}
	defer tx.Rollback(ctx)

	var chatID int
	if err = tx.QueryRow(ctx, `INSERT INTO chats DEFAULT VALUES RETURNING id`).Scan(&chatID); err != nil {
		return zeroInt, err
	}

	_, err = tx.Exec(ctx, `INSERT INTO users_chats (chat_id, user_id) VALUES ($1, $2), ($1, $3)`, chatID, from, to)
	if err != nil {
		return zeroInt, err
	}

	return chatID, tx.Commit(ctx)
}

func (r *Repository) SendMessage(ctx context.Context, chatID, from int, message string) error {

	_, err := r.pool.Exec(ctx, `INSERT INTO messages(chat_id, sender_id, data, created_at)
							VALUES($1, $2, $3, $4))`, chatID, from, message, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func scanRows(rows pgx.Rows, users []*model.UserFromDB) ([]*model.UserFromDB, error) {

	for rows.Next() {

		var user model.UserFromDB

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Login); err != nil {
			log.Println(err)
			return nil, err
		}

		users = append(users, &user)
	}

	if rows.Err() != nil {
		return users, rows.Err()
	}

	return users, nil
}
