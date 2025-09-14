package schema
import (
	"database/sql"
	"fmt"
	utils "app/pkg/utils"
	"strings"
)
// CreateUsersTable creates the users table with production-ready schema
func CreateUsersTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(30) UNIQUE NOT NULL,
			email VARCHAR(254) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			avatar_url VARCHAR(2048),
			bio TEXT,
			is_active BOOLEAN DEFAULT true,
			is_verified BOOLEAN DEFAULT false,
			last_login TIMESTAMP WITH TIME ZONE,
			login_count INTEGER DEFAULT 0,
			failed_login_attempts INTEGER DEFAULT 0,
			locked_until TIMESTAMP WITH TIME ZONE,
			email_verified_at TIMESTAMP WITH TIME ZONE,
			password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create indexes for performance
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE is_active = true;
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE is_active = true;
		CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
		CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login);
		CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

		-- Create partial index for active users
		CREATE INDEX IF NOT EXISTS idx_users_active ON users(id) WHERE is_active = true;

		-- Add constraints
		ALTER TABLE users ADD CONSTRAINT chk_username_length CHECK (length(username) >= 3 AND length(username) <= 30);
		ALTER TABLE users ADD CONSTRAINT chk_email_length CHECK (length(email) >= 5 AND length(email) <= 254);
		ALTER TABLE users ADD CONSTRAINT chk_password_hash_length CHECK (length(password_hash) >= 60);
		ALTER TABLE users ADD CONSTRAINT chk_bio_length CHECK (length(bio) <= 500);
		ALTER TABLE users ADD CONSTRAINT chk_login_count CHECK (login_count >= 0);
		ALTER TABLE users ADD CONSTRAINT chk_failed_login_attempts CHECK (failed_login_attempts >= 0);

		-- Create updated_at trigger
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ language 'plpgsql';

		CREATE TRIGGER update_users_updated_at 
			BEFORE UPDATE ON users 
			FOR EACH ROW 
			EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		// Ignore errors about existing constraints, indexes, or tables
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value") || strings.Contains(dbErrStr, "already defined")) {
			utils.Logger.Errorf("Failed to create users table: %v", err)
			return fmt.Errorf("failed to create users table: %w", err)
		}
	}

	utils.Logger.Info("Users table created successfully")
	return nil
}

// CreateUserSessionsTable creates the user sessions table
func CreateUserSessionsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			refresh_token_hash VARCHAR(255),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			is_active BOOLEAN DEFAULT true,
			user_agent TEXT,
			ip_address INET,
			last_used TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create indexes
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_token_hash ON user_sessions(token_hash);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_last_used ON user_sessions(last_used);

		-- Create partial index for active sessions
		CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(id) WHERE is_active = true;

		-- Add constraints
		ALTER TABLE user_sessions ADD CONSTRAINT chk_token_hash_length CHECK (length(token_hash) >= 32);
		ALTER TABLE user_sessions ADD CONSTRAINT chk_expires_at_future CHECK (expires_at > created_at);

		-- Create updated_at trigger
		CREATE TRIGGER update_user_sessions_updated_at 
			BEFORE UPDATE ON user_sessions 
			FOR EACH ROW 
			EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		// Ignore errors about existing constraints, indexes, or tables
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value") || strings.Contains(dbErrStr, "already defined")) {
			utils.Logger.Errorf("Failed to create user_sessions table: %v", err)
			return fmt.Errorf("failed to create user_sessions table: %w", err)
		}
	}

	utils.Logger.Info("User sessions table created successfully")
	return nil
}

// CreateAuditLogsTable creates the audit logs table for security compliance
func CreateAuditLogsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			event_type VARCHAR(50) NOT NULL,
			description TEXT,
			ip_address INET,
			user_agent TEXT,
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create indexes
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_metadata ON audit_logs USING GIN(metadata);

		-- Add constraints
		ALTER TABLE audit_logs ADD CONSTRAINT chk_event_type_length CHECK (length(event_type) <= 50);
		ALTER TABLE audit_logs ADD CONSTRAINT chk_description_length CHECK (length(description) <= 1000);

		-- Create partition by month for better performance (optional)
		-- This requires PostgreSQL 10+ and proper partitioning setup
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		// Ignore errors about existing constraints, indexes, or tables
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value") || strings.Contains(dbErrStr, "already defined")) {
			utils.Logger.Errorf("Failed to create audit_logs table: %v", err)
			return fmt.Errorf("failed to create audit_logs table: %w", err)
		}
	}

	utils.Logger.Info("Audit logs table created successfully")
	return nil
}

// CreatePasswordResetTokensTable creates the password reset tokens table
func CreatePasswordResetTokensTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			used_at TIMESTAMP WITH TIME ZONE,
			ip_address INET,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create indexes
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

		-- Add constraints
		ALTER TABLE password_reset_tokens ADD CONSTRAINT chk_token_hash_length CHECK (length(token_hash) >= 32);
		ALTER TABLE password_reset_tokens ADD CONSTRAINT chk_expires_at_future CHECK (expires_at > created_at);
		ALTER TABLE password_reset_tokens ADD CONSTRAINT chk_used_at_after_created CHECK (used_at IS NULL OR used_at >= created_at);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		// Ignore errors about existing constraints, indexes, or tables
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value") || strings.Contains(dbErrStr, "already defined")) {
			utils.Logger.Errorf("Failed to create password_reset_tokens table: %v", err)
			return fmt.Errorf("failed to create password_reset_tokens table: %w", err)
		}
	}

	utils.Logger.Info("Password reset tokens table created successfully")
	return nil
}

// CreateEmailVerificationTokensTable creates the email verification tokens table
func CreateEmailVerificationTokensTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS email_verification_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			used_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create indexes
		CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
		CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_token_hash ON email_verification_tokens(token_hash);
		CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_expires_at ON email_verification_tokens(expires_at);

		-- Add constraints
		ALTER TABLE email_verification_tokens ADD CONSTRAINT chk_email_verification_token_hash_length CHECK (length(token_hash) >= 32);
		ALTER TABLE email_verification_tokens ADD CONSTRAINT chk_email_verification_expires_at_future CHECK (expires_at > created_at);
		ALTER TABLE email_verification_tokens ADD CONSTRAINT chk_email_verification_used_at_after_created CHECK (used_at IS NULL OR used_at >= created_at);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		// Ignore errors about existing constraints, indexes, or tables
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value") || strings.Contains(dbErrStr, "already defined")) {
			utils.Logger.Errorf("Failed to create email_verification_tokens table: %v", err)
			return fmt.Errorf("failed to create email_verification_tokens table: %w", err)
		}
	}

	utils.Logger.Info("Email verification tokens table created successfully")
	return nil
}

// CreateTokenBlacklistTable creates the token blacklist table for revoked tokens
func CreateTokenBlacklistTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS token_blacklist (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			token_hash VARCHAR(255) UNIQUE NOT NULL,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			session_id UUID,
			token_type VARCHAR(20) NOT NULL, -- 'access' or 'refresh'
			blacklisted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			reason VARCHAR(100), -- 'logout', 'refresh', 'security', etc.
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create indexes for performance
		CREATE INDEX IF NOT EXISTS idx_token_blacklist_token_hash ON token_blacklist(token_hash);
		CREATE INDEX IF NOT EXISTS idx_token_blacklist_user_id ON token_blacklist(user_id);
		CREATE INDEX IF NOT EXISTS idx_token_blacklist_session_id ON token_blacklist(session_id);
		CREATE INDEX IF NOT EXISTS idx_token_blacklist_expires_at ON token_blacklist(expires_at);
		CREATE INDEX IF NOT EXISTS idx_token_blacklist_blacklisted_at ON token_blacklist(blacklisted_at);
		CREATE INDEX IF NOT EXISTS idx_token_blacklist_token_type ON token_blacklist(token_type);

		-- Add constraints
		ALTER TABLE token_blacklist ADD CONSTRAINT chk_token_blacklist_token_hash_length CHECK (length(token_hash) >= 32);
		ALTER TABLE token_blacklist ADD CONSTRAINT chk_token_blacklist_expires_at_future CHECK (expires_at > blacklisted_at);
		ALTER TABLE token_blacklist ADD CONSTRAINT chk_token_blacklist_token_type CHECK (token_type IN ('access', 'refresh'));
		ALTER TABLE token_blacklist ADD CONSTRAINT chk_token_blacklist_reason_length CHECK (length(reason) <= 100);

		-- Create a function to clean up expired blacklisted tokens
		CREATE OR REPLACE FUNCTION cleanup_expired_blacklisted_tokens()
		RETURNS INTEGER AS $$
		DECLARE
			deleted_count INTEGER;
		BEGIN
			DELETE FROM token_blacklist WHERE expires_at < NOW();
			GET DIAGNOSTICS deleted_count = ROW_COUNT;
			RETURN deleted_count;
		END;
		$$ LANGUAGE plpgsql;
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		// Ignore errors about existing constraints, indexes, or tables
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value") || strings.Contains(dbErrStr, "already defined")) {
			utils.Logger.Errorf("Failed to create token_blacklist table: %v", err)
			return fmt.Errorf("failed to create token_blacklist table: %w", err)
		}
	}

	utils.Logger.Info("Token blacklist table created successfully")
	return nil
}

// CreateAllTables creates all authentication-related tables
func CreateAllTables(db *sql.DB) error {
	tables := []struct {
		name string
		fn   func(*sql.DB) error
	}{
		{"users", CreateUsersTable},
		{"user_sessions", CreateUserSessionsTable},
		{"audit_logs", CreateAuditLogsTable},
		{"password_reset_tokens", CreatePasswordResetTokensTable},
		{"email_verification_tokens", CreateEmailVerificationTokensTable},
		{"token_blacklist", CreateTokenBlacklistTable},
	}

	for _, table := range tables {
		utils.Logger.Infof("Creating %s table...", table.name)
		if err := table.fn(db); err != nil {
			return fmt.Errorf("failed to create %s table: %w", table.name, err)
		}
	}

	utils.Logger.Info("All authentication tables created successfully")
	return nil
}

// CleanupExpiredTokens removes expired tokens from the database
func CleanupExpiredTokens(db *sql.DB) error {
	queries := []string{
		`DELETE FROM user_sessions WHERE expires_at < NOW()`,
		`DELETE FROM password_reset_tokens WHERE expires_at < NOW()`,
		`DELETE FROM email_verification_tokens WHERE expires_at < NOW()`,
	}

	for _, query := range queries {
		result, err := db.Exec(query)
		if err != nil {
			utils.Logger.Errorf("Failed to cleanup expired tokens: %v", err)
			return fmt.Errorf("failed to cleanup expired tokens: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			utils.Logger.Infof("Cleaned up %d expired tokens", rowsAffected)
		}
	}

	return nil
}

// CreateMaintenanceFunctions creates database maintenance functions
func CreateMaintenanceFunctions(db *sql.DB) error {
	query := `
		-- Function to cleanup old audit logs (keep last 90 days)
		CREATE OR REPLACE FUNCTION cleanup_old_audit_logs()
		RETURNS INTEGER AS $$
		DECLARE
			deleted_count INTEGER;
		BEGIN
			DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days';
			GET DIAGNOSTICS deleted_count = ROW_COUNT;
			RETURN deleted_count;
		END;
		$$ LANGUAGE plpgsql;

		-- Function to get user statistics
		CREATE OR REPLACE FUNCTION get_user_statistics()
		RETURNS TABLE(
			total_users BIGINT,
			active_users BIGINT,
			verified_users BIGINT,
			users_this_month BIGINT,
			users_this_week BIGINT
		) AS $$
		BEGIN
			RETURN QUERY
			SELECT 
				COUNT(*) as total_users,
				COUNT(*) FILTER (WHERE is_active = true) as active_users,
				COUNT(*) FILTER (WHERE is_verified = true) as verified_users,
				COUNT(*) FILTER (WHERE created_at >= date_trunc('month', NOW())) as users_this_month,
				COUNT(*) FILTER (WHERE created_at >= date_trunc('week', NOW())) as users_this_week
			FROM users;
		END;
		$$ LANGUAGE plpgsql;
	`

	_, err := db.Exec(query)
	if err != nil {
		utils.Logger.Errorf("Failed to create maintenance functions: %v", err)
		return fmt.Errorf("failed to create maintenance functions: %w", err)
	}

	utils.Logger.Info("Maintenance functions created successfully")
	return nil
}
