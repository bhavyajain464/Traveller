DROP INDEX IF EXISTS idx_auth_sessions_expires_at;
DROP INDEX IF EXISTS idx_auth_sessions_user_id;
DROP TABLE IF EXISTS auth_sessions;

DROP INDEX IF EXISTS idx_users_auth_provider;
DROP INDEX IF EXISTS idx_users_google_sub;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS auth_provider,
    DROP COLUMN IF EXISTS google_sub,
    DROP COLUMN IF EXISTS avatar_url;

ALTER TABLE users
    ALTER COLUMN phone_number SET NOT NULL;
