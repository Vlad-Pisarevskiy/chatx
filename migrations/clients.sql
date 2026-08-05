--goose up
CREATE TABLE users(
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text,
    login text UNIQUE,
    email text UNIQUE,
    password text
);

--goose down
DROP TABLE users;