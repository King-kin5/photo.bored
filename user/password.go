package user

import (
	"errors"
	"fmt"
	"strings"
	"golang.org/x/crypto/bcrypt"
	"regexp"
)

func CheckPasswordLevel(pass string) error {
    pass = strings.ToLower(pass)
    if len(pass) < 8 {
        return fmt.Errorf("password is less than 8 characters")
    }
    num := `[0-9]{1}`
    aToZ := `[a-z]{1}`

    if b, _ := regexp.MatchString(num, pass); !b {
        return fmt.Errorf("password needs numbers")
    }
    if b, _ := regexp.MatchString(aToZ, pass); !b {
        return fmt.Errorf("password needs characters")
    }
    return nil
}

// PasswordHash hashes the password
func PasswordHash(password string) (string, error) {
    if len(password) == 0 {
        return "", errors.New("password cannot be empty")
    }
    h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(h), err
}

// CheckPasswordSame compares a hashed password with a plain text password
func CheckPasswordSame(hashedPassword, plainPassword string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
    return err == nil
}