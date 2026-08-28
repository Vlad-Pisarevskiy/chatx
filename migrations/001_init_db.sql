-- +goose Up
CREATE TABLE users
(
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name     TEXT NOT NULL ,
    login    TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL
);

CREATE TABLE chats
(
    id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

CREATE TABLE users_chats
(
    chat_id   INT REFERENCES chats (id),
    user_id   BIGINT REFERENCES users (id) ON DELETE CASCADE,
    last_read BIGINT,
);

CREATE TABLE messages
(
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    chat_id   INT REFERENCES chats (id) ON DELETE CASCADE NOT NULL ,
    sender_id BIGINT REFERENCES users(id) NOT NULL,
    data      TEXT,
    created_at timestamptz
);

CREATE TABLE tokens
(
    token_hash bytea PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz,
    created_at timestamptz
);

-- +goose Down
DROP TABLE tokens;
DROP TABLE users_chats;
DROP TABLE messages;
DROP TABLE chats;
DROP TABLE users;

