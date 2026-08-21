-- +goose Up
ALTER TABLE users_chats ADD PRIMARY KEY (user_id, chat_id);
CREATE INDEX ON users_chats (chat_id);

-- +goose Down
DROP INDEX users_chats_chat_id_idx;
ALTER TABLE users_chats DROP CONSTRAINT users_chats_pkey;