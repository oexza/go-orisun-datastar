-- name: ResetReadModelAuthSessions :exec
DELETE FROM auth_session;

-- name: ResetReadModelAuthAccounts :exec
DELETE FROM auth_account;

-- name: ResetReadModelAuthUsers :exec
DELETE FROM auth_user;

-- name: ResetReadModelAuthVerifications :exec
DELETE FROM auth_verification;

-- name: ResetReadModelProfiles :exec
DELETE FROM profile_stats;

-- name: ResetReadModelTodos :exec
DELETE FROM todo_items;

-- name: ResetEventHandlerCheckpoints :exec
DELETE FROM projector_checkpoint;
