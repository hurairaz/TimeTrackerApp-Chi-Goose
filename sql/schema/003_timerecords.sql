-- +goose Up
CREATE TABLE time_records (
    id UUID PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    stop_time TIMESTAMP,
    duration INTERVAL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE time_records;