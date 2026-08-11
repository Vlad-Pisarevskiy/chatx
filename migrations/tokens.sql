--goose up
CREATE TABLE sessions(
    token_hash bytea,
    user_id bigint,
    expires_at TIMESTAMP,
    created_at TIMESTAMP
);