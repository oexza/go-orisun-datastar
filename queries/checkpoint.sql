-- name: GetEventHandlerCheckpoint :one
SELECT commit_position, prepare_position
FROM projector_checkpoint
WHERE name = @name;

-- name: UpsertEventHandlerCheckpoint :exec
INSERT INTO projector_checkpoint (id, name, commit_position, prepare_position, updated_at)
VALUES (@id, @name, @commit_position, @prepare_position, CURRENT_TIMESTAMP)
ON CONFLICT (name) DO UPDATE
SET commit_position = EXCLUDED.commit_position,
    prepare_position = EXCLUDED.prepare_position,
    updated_at = CURRENT_TIMESTAMP;
