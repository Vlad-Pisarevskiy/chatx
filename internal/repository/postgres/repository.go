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

const (
	null   = 0
	nullID = 0
	nullChat
	tokenTTL       = time.Hour * 24
	tokenThreshold = time.Hour * 12
)

func New(pool *pgxpool.Pool) *Repository {

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

func (r *Repository) LoginExists(ctx context.Context, login string) error {

	row := r.pool.QueryRow(ctx, "SELECT login FROM users WHERE login = $1", login)

	var dbLogin string

	err := row.Scan(&dbLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	return errors1.ErrExistsLogin
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

	_, err := r.pool.Exec(ctx, `INSERT INTO tokens(user_id, token_hash, created_at, expires_at) 
									VALUES ($1, $2, $3, $4)`, userID, token, time.Now(), time.Now().Add(tokenTTL))
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) CheckToken(ctx context.Context, token []byte) (userID int, err error) {

	row := r.pool.QueryRow(ctx, "SELECT user_id FROM tokens WHERE token_hash=($1) AND expires_at>$2", token, time.Now())

	err = row.Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nullID, errors1.ErrInvalidToken
		}
		return userID, err
	}

	_, err = r.pool.Exec(ctx, `UPDATE tokens SET expires_at = $1 WHERE token_hash = $2 AND expires_at < $3`,
		time.Now().Add(tokenTTL), token, time.Now().Add(tokenThreshold))
	if err != nil {
		log.Println(err)
	}

	return userID, nil
}

func (r *Repository) RemoveToken(ctx context.Context, token []byte) error {

	_, err := r.pool.Exec(ctx, `DELETE FROM tokens WHERE token_hash = $1`, token)
	if err != nil {
		return err
	}

	return nil
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
		`SELECT chat_id                                                                                                                                                                      
  			 FROM users_chats                                                                                                                                                                    
			 WHERE user_id IN ($1, $2)                                                                                                                                                           
  			 GROUP BY chat_id                                                                                                                                                                    
  			 HAVING count(*) = 2`, from, to)

	var chatID int

	if err := row.Scan(&chatID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nullChat, false, nil
		}
		return chatID, true, err
	}

	return chatID, true, nil
}

func (r *Repository) StartChat(ctx context.Context, from, to int) (int, error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nullChat, err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		_ = tx.Rollback(ctx)
	}(tx, ctx)

	var chatID int
	if err = tx.QueryRow(ctx, `INSERT INTO chats DEFAULT VALUES RETURNING id`).Scan(&chatID); err != nil {
		return nullChat, err
	}

	_, err = tx.Exec(ctx, `INSERT INTO users_chats (chat_id, user_id) VALUES ($1, $2), ($1, $3)`, chatID, from, to)
	if err != nil {
		return nullChat, err
	}

	return chatID, tx.Commit(ctx)
}

func (r *Repository) SendMessage(ctx context.Context, chatID, from int, message string) error {

	_, err := r.pool.Exec(ctx, `INSERT INTO messages(chat_id, sender_id, data, created_at)
							VALUES($1, $2, $3, $4)`, chatID, from, message, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) LoadMessages(ctx context.Context, from, to int) ([]model.Message, error) {

	var messages []model.Message
	var chatID int
	//TODO: Запрос на получение чата пока работает но при групповых чатах поломается
	if err := r.pool.QueryRow(ctx,
		`SELECT chat_id                                                                                                                                                                      
  			 FROM users_chats                                                                                                                                                                    
			 WHERE user_id IN ($1, $2)                                                                                                                                                           
  			 GROUP BY chat_id                                                                                                                                                                    
  			 HAVING COUNT(*) = 2`, from, to).Scan(&chatID); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT
				m.sender_id,
				m.data,
				m.created_at
			 FROM messages m
			 JOIN users_chats uc ON uc.chat_id = m.chat_id
			 WHERE uc.user_id = $1 AND m.chat_id = $2
			 ORDER BY m.created_at;
			`, from, chatID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message model.Message
		if err = rows.Scan(
			&message.From,
			&message.Data,
			&message.CreatedAt); err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *Repository) DeleteExpiredTokens(ctx context.Context) (int64, error) {

	tag, err := r.pool.Exec(ctx, `DELETE FROM tokens WHERE expires_at < $1`, time.Now())
	if err != nil {
		return null, err
	}

	return tag.RowsAffected(), nil
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
