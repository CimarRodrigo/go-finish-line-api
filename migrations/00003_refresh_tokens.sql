-- +goose Up
CREATE TABLE refresh_tokens (
    id         uuid        PRIMARY KEY,
    user_id    uuid        NOT NULL,
    family_id  uuid        NOT NULL,
    token_hash text        NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    rotated_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL,
    CONSTRAINT fk_refresh_tokens_user
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens (family_id);

-- +goose Down
DROP TABLE refresh_tokens;
