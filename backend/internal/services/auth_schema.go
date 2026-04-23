package services

import (
	"fmt"

	"indian-transit-backend/internal/database"
)

// EnsureAuthSchema makes the Google-auth additions available even if the
// local database was created before the auth migration was added.
func EnsureAuthSchema(db *database.DB) error {
	statements := []string{
		`ALTER TABLE users ALTER COLUMN phone_number DROP NOT NULL`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS google_sub VARCHAR(255) UNIQUE`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_provider VARCHAR(50) NOT NULL DEFAULT 'phone'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP`,
		`UPDATE users
		 SET auth_provider = CASE
		 	WHEN google_sub IS NOT NULL THEN 'google'
		 	ELSE 'phone'
		 END
		 WHERE auth_provider IS NULL OR auth_provider = ''`,
		`CREATE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub)`,
		`CREATE INDEX IF NOT EXISTS idx_users_auth_provider ON users(auth_provider)`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			token_hash VARCHAR(255) UNIQUE NOT NULL,
			provider VARCHAR(50) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			revoked_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("ensure auth schema failed for %q: %w", stmt, err)
		}
	}

	return nil
}
