package Auth

import (
	"time"

	"github.com/google/uuid"
)

// UserSession represents a user's active session
type UserSession struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UserAgent *string   `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress *string   `json:"ip_address,omitempty" db:"ip_address"`
	LastUsed  time.Time `json:"last_used" db:"last_used"`
}

type UserStore interface {
	CreateUser(user *User) error
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	GetUserByName(name string) (*User, error)
	UpdateUserInfoByID(id string, user User) error
	UpdateUserInfoByUsernameOrEmail(identifier string, user User) error
	CheckDBConnection() error

	// Production-ready additions
	DeleteUser(id string) error
	GetUserSessions(userID uuid.UUID) ([]UserSession, error)
	CreateUserSession(session *UserSession) error
	DeleteUserSession(sessionID uuid.UUID) error
	InvalidateAllUserSessions(userID uuid.UUID) error
	UpdateLastLogin(userID uuid.UUID) error
	GetUserCount() (int64, error)
	SearchUsers(query string, limit, offset int) ([]*User, error)
}

// AuditLog interface for security tracking
type AuditLogger interface {
	LogLogin(userID uuid.UUID, ipAddress, userAgent string, success bool)
	LogRegistration(userID uuid.UUID, ipAddress, userAgent string)
	LogPasswordChange(userID uuid.UUID, ipAddress string)
	LogProfileUpdate(userID uuid.UUID, ipAddress string)
	LogFailedLogin(email, ipAddress, userAgent string)
}
