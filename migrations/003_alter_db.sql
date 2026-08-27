-- +goose Up
ALTER TABLE messages ADD CONSTRAINT messages_data_len CHECK (char_length(data) BETWEEN 1 AND 500) NOT VALID;
ALTER TABLE messages VALIDATE CONSTRAINT messages_data_len;

-- +goose Down
DROP CONSTRAINT messages_data_len