-- +goose Up
CREATE TABLE participants (
    id               uuid        PRIMARY KEY,
    nombres          text        NOT NULL,
    apellidos        text        NOT NULL,
    email            citext      NOT NULL UNIQUE,
    telefono         text        NOT NULL,
    fecha_nacimiento date        NOT NULL,
    genero           text        NOT NULL,
    created_at       timestamptz NOT NULL
);

CREATE TABLE registrations (
    id                uuid        PRIMARY KEY,
    participant_id    uuid        NOT NULL,
    race_id           uuid        NOT NULL,
    como_te_enteraste text        NOT NULL,
    estado            text        NOT NULL,
    dorsal            integer,
    created_at        timestamptz NOT NULL,
    confirmed_at      timestamptz,
    CONSTRAINT uq_registration_race_participant UNIQUE (race_id, participant_id),
    CONSTRAINT uq_registration_race_dorsal      UNIQUE (race_id, dorsal),
    CONSTRAINT fk_registrations_participant
        FOREIGN KEY (participant_id) REFERENCES participants (id) ON DELETE RESTRICT,
    CONSTRAINT fk_registrations_race
        FOREIGN KEY (race_id) REFERENCES races (id) ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE registrations;
DROP TABLE participants;
