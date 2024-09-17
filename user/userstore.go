package user

import (
	"database/sql"
	"fmt"
	"log"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (us *UserStore) CreateUser(user *User) error {
	query := "INSERT INTO users (Username, Email, Password) VALUES ($1, $2, $3)"
	_, err := us.db.Exec(query, user.Username, user.Email, user.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
	}
	return err
}

func (us *UserStore) CheckDBConnection() error {
	err := us.db.Ping()
	if err != nil {
		return fmt.Errorf("failed to connect to the database: %v", err)
	}
	return nil
}

func (us *UserStore) GetUserByEmail(email string) (*User, error) {
	user := new(User)
	query := `SELECT ID, Username, Email, Password FROM users WHERE email=$1`
	row := us.db.QueryRow(query, email)
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No user found, return nil for user
		}
		log.Printf("Error scanning row: %v", err)
		return nil, err // Return the error if something else went wrong
	}
	return user, nil
}
