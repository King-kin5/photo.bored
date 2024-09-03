package user

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"github.com/google/uuid"
)
type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
}
// RegisterUser handles user registration
func (m *Model) RegisterUser(w http.ResponseWriter, r *http.Request) {
	   // Log the request body
	   body, _ := ioutil.ReadAll(r.Body)
	   log.Printf("Received request body: %s", string(body))
	   r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
   
	// Define a struct to hold the incoming JSON data
	var input struct {
        Username string `json:"Username" form:"Username"`
        Email    string `json:"Email" form:"Email"`
        Password string `json:"Password" form:"Password"`
    }
	// Get the Content-Type of the request
    contentType := r.Header.Get("Content-Type")
    // Check if the content type is JSON
    if strings.Contains(contentType, "application/json") {
		// If it's JSON, decode the request body into the input struct
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
            log.Printf("Error decoding JSON: %v", err)
            http.Error(w, "Invalid input", http.StatusBadRequest)
            return
        }
    } else {
		// If it's not JSON, assume it's form data and parse it
        if err := r.ParseForm(); err != nil {
            log.Printf("Error parsing form: %v", err)
            http.Error(w, "Invalid input", http.StatusBadRequest)
            return
        }
		// Manually assign form values to the input struct
        input.Username = r.FormValue("Username")
        input.Email = r.FormValue("Email")
        input.Password = r.FormValue("Password")
    }

    log.Printf("Received input: %+v", input)

	if input.Username == "" || input.Email == "" || input.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	// Check password strength
	//if err := CheckPasswordLevel(input.Password); err != nil {
		//http.Error(w, err.Error(), http.StatusBadRequest)
		//return
	//}
	// Check the database connection
	if err := m.userstore.CheckDBConnection(); err != nil {
		log.Printf("Database connection error: %v", err)
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	existingUser, err := m.userstore.GetUserByEmail(input.Email)
	if err != nil {
		http.Error(w, "Failed to check if email exists", http.StatusInternalServerError)
		return
	}
	if existingUser != nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}
	hashedPassword, err := PasswordHash(input.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	user := &User{
		ID:       uuid.New(),
		Username: input.Username,
		Email:    input.Email,
		Password: hashedPassword,
	}
	if err := m.userstore.CreateUser(user); err != nil {
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}
func (m *Model) LoginUser(w http.ResponseWriter, r *http.Request) {
    // Log the request body for debugging
    body, _ := ioutil.ReadAll(r.Body)
    log.Printf("Received request body: %s", string(body))
    r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

    // Define a struct to hold the incoming data
    var input struct {
        Email    string `json:"Email" form:"Email"`
        Password string `json:"Password" form:"Password"`
    }

    // Get the Content-Type of the request
    contentType := r.Header.Get("Content-Type")

    // Check if the content type is JSON
    if strings.Contains(contentType, "application/json") {
        // If it's JSON, decode the request body into the input struct
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
            log.Printf("Error decoding JSON: %v", err)
            http.Error(w, "Invalid input", http.StatusBadRequest)
            return
        }
    } else {
        // If it's not JSON, assume it's form data and parse it
        if err := r.ParseForm(); err != nil {
            log.Printf("Error parsing form data: %v", err)
            http.Error(w, "Invalid input", http.StatusBadRequest)
            return
        }
        // Manually assign form values to the input struct
        input.Email = r.FormValue("Email")
        input.Password = r.FormValue("Password")
    }

    log.Printf("Received login input: %+v", input)

    // Check if email or password is empty
    if input.Email == "" || input.Password == "" {
        log.Printf("Login failed: Email or Password is empty")
        w.WriteHeader(http.StatusBadRequest)
        w.Write([]byte("Email and Password are required"))
        return
    }

    // Retrieve the user by email
    user, err := m.userstore.GetUserByEmail(input.Email)
    if err != nil {
        log.Printf("UserLogin: Failed to get user by email: %v\n", err)
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("Login failed: Internal server error"))
        return
    }

    // Check if user exists
    if user == nil {
        log.Printf("Login failed: Invalid email or password")
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte("Invalid email or password"))
        return
    }

    // Validate the password
    if !CheckPasswordSame(user.Password, input.Password) {
        log.Printf("Login failed: Invalid email or password")
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte("Invalid email or password"))
        return
    }

    log.Printf("User login successful for email: %s", input.Email)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("User login successful"))
}
