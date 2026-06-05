-- name: GetEventHandlerCheckpoint :one
SELECT commit_position::bigint, prepare_position::bigint
FROM projector_checkpoint
WHERE name = @name;

-- name: UpsertEventHandlerCheckpoint :exec
INSERT INTO projector_checkpoint (id, name, commit_position, prepare_position, updated_at)
VALUES (@id, @name, @commit_position, @prepare_position, now())
ON CONFLICT (name) DO UPDATE
SET commit_position = EXCLUDED.commit_position,
    prepare_position = EXCLUDED.prepare_position,
    updated_at = now();
