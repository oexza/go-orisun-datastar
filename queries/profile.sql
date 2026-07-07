-- name: ProfileUser :one
SELECT user_id,
       coalesce(name, '') AS name,
       coalesce(username, '') AS username,
       coalesce(email, '') AS email,
       coalesce(image, '') AS image,
       coalesce(bio, '') AS bio,
       coalesce(header_image_url, '') AS header_image_url
FROM profile_stats
WHERE user_id = @user_id;

-- name: UpsertRegisteredProfileUser :exec
INSERT INTO profile_stats (user_id, name, username, email, last_event_commit_position, last_event_prepare_position)
VALUES (@user_id, @name, @username, @email, @last_event_commit_position, @last_event_prepare_position)
ON CONFLICT (user_id) DO UPDATE SET
    name = EXCLUDED.name,
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    last_event_commit_position = EXCLUDED.last_event_commit_position,
    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertProfileName :exec
INSERT INTO profile_stats (user_id, name, last_event_commit_position, last_event_prepare_position)
VALUES (@user_id, @name, @last_event_commit_position, @last_event_prepare_position)
ON CONFLICT (user_id) DO UPDATE SET
    name = EXCLUDED.name,
    last_event_commit_position = EXCLUDED.last_event_commit_position,
    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertProfileBio :exec
INSERT INTO profile_stats (user_id, bio, last_event_commit_position, last_event_prepare_position)
VALUES (@user_id, @bio, @last_event_commit_position, @last_event_prepare_position)
ON CONFLICT (user_id) DO UPDATE SET
    bio = EXCLUDED.bio,
    last_event_commit_position = EXCLUDED.last_event_commit_position,
    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertProfileImage :exec
INSERT INTO profile_stats (user_id, image, last_event_commit_position, last_event_prepare_position)
VALUES (@user_id, @image, @last_event_commit_position, @last_event_prepare_position)
ON CONFLICT (user_id) DO UPDATE SET
    image = EXCLUDED.image,
    last_event_commit_position = EXCLUDED.last_event_commit_position,
    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertProfileHeaderImage :exec
INSERT INTO profile_stats (user_id, header_image_url, last_event_commit_position, last_event_prepare_position)
VALUES (@user_id, @header_image_url, @last_event_commit_position, @last_event_prepare_position)
ON CONFLICT (user_id) DO UPDATE SET
    header_image_url = EXCLUDED.header_image_url,
    last_event_commit_position = EXCLUDED.last_event_commit_position,
    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteProfileByRegisteredID :exec
DELETE FROM profile_stats
WHERE user_id = @user_registered_id;
