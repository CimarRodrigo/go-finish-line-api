-- +goose Up
ALTER TABLE races RENAME COLUMN strapi_id TO document_id;

-- +goose Down
ALTER TABLE races RENAME COLUMN document_id TO strapi_id;
