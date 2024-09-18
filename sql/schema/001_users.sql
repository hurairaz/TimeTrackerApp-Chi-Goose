-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL
);

-- +goose Down
DROP TABLE users;