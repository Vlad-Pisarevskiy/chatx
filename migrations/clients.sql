--goose up
CREATE TABLE users(
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text,
    login text,
    email text,
    password text
);

--goose down
DROP TABLE users;