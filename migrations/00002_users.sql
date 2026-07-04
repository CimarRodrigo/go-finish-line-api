-- +goose Up
CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    nombre        text        NOT NULL,
    email         citext      NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL
);

-- +goose Down
DROP TABLE users;
