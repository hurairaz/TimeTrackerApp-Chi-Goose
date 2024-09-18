-- name: CreateTimeRecord :one
INSERT INTO time_records(id, start_time, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateTimeRecord :one
UPDATE time_records
SET stop_time = $1, duration = $1 - start_time
WHERE id = $2
RETURNING *;

-- name: GetUserTimeRecords :many
SELECT * FROM time_records WHERE user_id = $1;