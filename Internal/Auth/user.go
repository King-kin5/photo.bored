package Auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	Username   string     `json:"username" db:"username"`
	Email      string     `json:"email" db:"email"`
	Password   string     `json:"-" db:"password_hash"` // Never return password in JSON
	AvatarURL  *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	Bio        *string    `json:"bio,omitempty" db:"bio"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	IsVerified bool       `json:"is_verified" db:"is_verified"`
	LastLogin  *time.Time `json:"last_login,omitempty" db:"last_login"`
	LoginCount int        `json:"login_count" db:"login_count"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

type CreateUser struct {
	Username string  `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email    string  `json:"email" validate:"required,email,max=254"`
	Password string  `json:"password" validate:"required,min=8,max=128"`
	Bio      *string `json:"bio,omitempty" validate:"omitempty,max=500"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserRequest struct {
	Username  *string `json:"username,omitempty" validate:"omitempty,min=3,max=30,alphanum"`
	Bio       *string `json:"bio,omitempty" validate:"omitempty,max=500"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,url,max=2048"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

type LoginResponse struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
	ExpiresIn    int64                  `json:"expires_in"`
}

// PublicUser returns user data safe for public consumption
func (u *User) PublicUser() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"username":   u.Username,
		"avatar_url": u.AvatarURL,
		"bio":        u.Bio,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

// PrivateUser returns user data for authenticated requests
func (u *User) PrivateUser() map[string]interface{} {
	return map[string]interface{}{
		"id":          u.ID,
		"username":    u.Username,
		"email":       u.Email,
		"avatar_url":  u.AvatarURL,
		"bio":         u.Bio,
		"is_active":   u.IsActive,
		"is_verified": u.IsVerified,
		"last_login":  u.LastLogin,
		"login_count": u.LoginCount,
		"created_at":  u.CreatedAt,
		"updated_at":  u.UpdatedAt,
	}
}
