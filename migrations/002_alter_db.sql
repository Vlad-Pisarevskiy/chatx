-- +goose Up
ALTER TABLE users_chats ADD PRIMARY KEY (user_id, chat_id);
CREATE INDEX ON users_chats (chat_id);