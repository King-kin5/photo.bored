package Auth

import (
	"context"
	"database/sql"
	"fmt"
	utils "app/pkg/utils"
	"time"

	"github.com/google/uuid"
)

type UserStoreImpl struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStoreImpl {
	return &UserStoreImpl{db: db}
}

func (us *UserStoreImpl) CreateUser(user *User) error {
	tx, err := us.db.Begin()
	if err != nil {
		utils.Logger.Errorf("Failed to begin transaction: %v", err)
		return fmt.Errorf("database error")
	}
	defer tx.Rollback()

	// Set default values
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.IsActive = true
	user.IsVerified = false
	user.LoginCount = 0

	query := `
		INSERT INTO users (id, username, email, password_hash, bio, is_active, is_verified, login_count, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.Exec(query,
		user.ID, user.Username, user.Email, user.Password, user.Bio,
		user.IsActive, user.IsVerified, user.LoginCount, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		utils.Logger.Errorf("Error creating user: %v", err)
		return fmt.Errorf("failed to create user")
	}

	return tx.Commit()
}

func (us *UserStoreImpl) CheckDBConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := us.db.PingContext(ctx)
	if err != nil {
		utils.Logger.Errorf("Database connection failed: %v", err)
		return fmt.Errorf("database connection failed")
	}
	return nil
}

func (us *UserStoreImpl) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, username, email, password_hash, avatar_url, bio, is_active, is_verified, 
		       last_login, login_count, created_at, updated_at 
		FROM users WHERE email = $1 AND is_active = true
	`

	row := us.db.QueryRow(query, email)
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.AvatarURL, &user.Bio,
		&user.IsActive, &user.IsVerified, &user.LastLogin, &user.LoginCount, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		utils.Logger.Errorf("Error scanning user by email: %v", err)
		return nil, fmt.Errorf("database error")
	}
	return user, nil
}

func (us *UserStoreImpl) GetUserByID(id string) (*User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format")
	}

	user := &User{}
	query := `
		SELECT id, username, email, password_hash, avatar_url, bio, is_active, is_verified, 
		       last_login, login_count, created_at, updated_at 
		FROM users WHERE id = $1 AND is_active = true
	`

	row := us.db.QueryRow(query, userID)
	err = row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.AvatarURL, &user.Bio,
		&user.IsActive, &user.IsVerified, &user.LastLogin, &user.LoginCount, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		utils.Logger.Errorf("Error scanning user by ID: %v", err)
		return nil, fmt.Errorf("database error")
	}
	return user, nil
}

func (us *UserStoreImpl) GetUserByName(name string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, username, email, password_hash, avatar_url, bio, is_active, is_verified, 
		       last_login, login_count, created_at, updated_at 
		FROM users WHERE username = $1 AND is_active = true
	`

	row := us.db.QueryRow(query, name)
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.AvatarURL, &user.Bio,
		&user.IsActive, &user.IsVerified, &user.LastLogin, &user.LoginCount, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		utils.Logger.Errorf("Error scanning user by name: %v", err)
		return nil, fmt.Errorf("database error")
	}
	return user, nil
}

func (us *UserStoreImpl) UpdateUserInfoByID(id string, user User) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid user ID format")
	}

	query := `
		UPDATE users 
		SET username = $1, bio = $2, avatar_url = $3, updated_at = $4 
		WHERE id = $5 AND is_active = true
	`

	result, err := us.db.Exec(query, user.Username, user.Bio, user.AvatarURL, time.Now(), userID)
	if err != nil {
		utils.Logger.Errorf("Error updating user: %v", err)
		return fmt.Errorf("failed to update user")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or inactive")
	}

	return nil
}

func (us *UserStoreImpl) UpdateUserInfoByUsernameOrEmail(identifier string, user User) error {
	query := `
		UPDATE users 
		SET username = $1, bio = $2, avatar_url = $3, updated_at = $4 
		WHERE (username = $5 OR email = $5) AND is_active = true
	`

	result, err := us.db.Exec(query, user.Username, user.Bio, user.AvatarURL, time.Now(), identifier)
	if err != nil {
		utils.Logger.Errorf("Error updating user by identifier: %v", err)
		return fmt.Errorf("failed to update user")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or inactive")
	}

	return nil
}

func (us *UserStoreImpl) UpdateLastLogin(userID uuid.UUID) error {
	query := `
		UPDATE users 
		SET last_login = $1, login_count = login_count + 1, updated_at = $1 
		WHERE id = $2 AND is_active = true
	`

	_, err := us.db.Exec(query, time.Now(), userID)
	if err != nil {
		utils.Logger.Errorf("Error updating last login: %v", err)
		return fmt.Errorf("failed to update last login")
	}
	return nil
}

func (us *UserStoreImpl) DeleteUser(ID string) error {
	userID, err := uuid.Parse(ID)
	if err != nil {
		return fmt.Errorf("invalid user ID format")
	}

	// Soft delete - mark as inactive
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE id = $2`

	result, err := us.db.Exec(query, time.Now(), userID)
	if err != nil {
		utils.Logger.Errorf("Error deleting user: %v", err)
		return fmt.Errorf("failed to delete user")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (us *UserStoreImpl) GetUserCount() (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM users WHERE is_active = true`

	err := us.db.QueryRow(query).Scan(&count)
	if err != nil {
		utils.Logger.Errorf("Error getting user count: %v", err)
		return 0, fmt.Errorf("database error")
	}
	return count, nil
}

func (us *UserStoreImpl) SearchUsers(query string, limit, offset int) ([]*User, error) {
	sqlQuery := `
		SELECT id, username, email, avatar_url, bio, is_active, is_verified, 
		       last_login, login_count, created_at, updated_at 
		FROM users 
		WHERE is_active = true AND (username ILIKE $1 OR email ILIKE $1)
		ORDER BY username
		LIMIT $2 OFFSET $3
	`

	rows, err := us.db.Query(sqlQuery, "%"+query+"%", limit, offset)
	if err != nil {
		utils.Logger.Errorf("Error searching users: %v", err)
		return nil, fmt.Errorf("database error")
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.AvatarURL, &user.Bio,
			&user.IsActive, &user.IsVerified, &user.LastLogin, &user.LoginCount, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			utils.Logger.Errorf("Error scanning user row: %v", err)
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

// Session management methods (implement based on your session storage)
func (us *UserStoreImpl) GetUserSessions(userID uuid.UUID) ([]UserSession, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, is_active, created_at, user_agent, ip_address, last_used
		FROM user_sessions 
		WHERE user_id = $1 AND is_active = true AND expires_at > $2
		ORDER BY created_at DESC
	`

	rows, err := us.db.Query(query, userID, time.Now())
	if err != nil {
		utils.Logger.Errorf("Error querying user sessions: %v", err)
		return nil, fmt.Errorf("database error")
	}
	defer rows.Close()

	var sessions []UserSession
	for rows.Next() {
		var session UserSession
		err := rows.Scan(
			&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt,
			&session.IsActive, &session.CreatedAt, &session.UserAgent, &session.IPAddress, &session.LastUsed,
		)
		if err != nil {
			utils.Logger.Errorf("Error scanning user session: %v", err)
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (us *UserStoreImpl) CreateUserSession(session *UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at, is_active, created_at, user_agent, ip_address, last_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := us.db.Exec(query,
		session.ID, session.UserID, session.TokenHash, session.ExpiresAt,
		session.IsActive, session.CreatedAt, session.UserAgent, session.IPAddress, session.LastUsed,
	)
	if err != nil {
		utils.Logger.Errorf("Error creating user session: %v", err)
		return fmt.Errorf("failed to create user session")
	}

	return nil
}

func (us *UserStoreImpl) DeleteUserSession(sessionID uuid.UUID) error {
	query := `UPDATE user_sessions SET is_active = false WHERE id = $1`

	result, err := us.db.Exec(query, sessionID)
	if err != nil {
		utils.Logger.Errorf("Error deleting user session: %v", err)
		return fmt.Errorf("failed to delete user session")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (us *UserStoreImpl) InvalidateAllUserSessions(userID uuid.UUID) error {
	query := `UPDATE user_sessions SET is_active = false WHERE user_id = $1`

	_, err := us.db.Exec(query, userID)
	if err != nil {
		utils.Logger.Errorf("Error invalidating user sessions: %v", err)
		return fmt.Errorf("failed to invalidate user sessions")
	}

	return nil
}
