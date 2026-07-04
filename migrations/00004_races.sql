-- +goose Up
CREATE TABLE races (
    id         uuid        PRIMARY KEY,
    strapi_id  text        NOT NULL UNIQUE,
    nombre     text        NOT NULL,
    fecha      date        NOT NULL,
    capacidad  integer     NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE races;
