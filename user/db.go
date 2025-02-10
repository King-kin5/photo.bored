package user
import(
	"database/sql"
	"fmt"
	"sync"
	_"github.com/lib/pq"
)
var dbInstance *sql.DB
var dbInstanceError error
var dbOnce sync.Once

const (
	// Update the connection string to match your PostgreSQL configuration
	// Replace "user", "password", "host", "port", and "dbname" with your actual PostgreSQL credentials and database name
	ConnectionStr = "user=postgres password=postgres host=localhost port=5432 dbname=board sslmode=disable"
)

func GetPostgresDB() (*sql.DB, error) {
	dbOnce.Do(func() {
		db, err := sql.Open("postgres", ConnectionStr)
		if err != nil {
			dbInstanceError = fmt.Errorf("failed to connect to PostgreSQL: %v", err)
			return
		}

		err = db.Ping()
		if err != nil {
			dbInstanceError = fmt.Errorf("failed to ping PostgreSQL: %v", err)
			return
		}

		dbInstance = db

		err = createSchema(db)
		if err != nil {
			dbInstanceError = fmt.Errorf("failed to create schema: %v", err)
			dbInstance = nil
		}
	})
	return dbInstance, dbInstanceError
}

func createSchema(db *sql.DB) error {
    query := `
CREATE TABLE IF NOT EXISTS users (
    UserID UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    Username TEXT NOT NULL,
    Email TEXT NOT NULL,
    Password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS photos (
    photo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    Filename TEXT NOT NULL,
    Data BYTEA NOT NULL,
    Date TIMESTAMP NOT NULL,
    Location TEXT NOT NULL,
    UserID UUID REFERENCES users(UserID)
);

CREATE TABLE IF NOT EXISTS albums (
    album_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    Name VARCHAR(255) NOT NULL,
    Created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS album_photo (
    album_id UUID NOT NULL,
    photo_id UUID NOT NULL,
    PRIMARY KEY (album_id, photo_id),
    FOREIGN KEY (album_id) REFERENCES albums(album_id) ON DELETE CASCADE,
    FOREIGN KEY (photo_id) REFERENCES photos(photo_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS photo_likes (
    photo_id UUID REFERENCES photos(photo_id),
    UserID UUID REFERENCES users(UserID),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (photo_id, UserID)
);

CREATE TABLE IF NOT EXISTS photo_comments (
    comment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id UUID REFERENCES photos(photo_id),
    UserID UUID REFERENCES users(UserID),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_follows (
    follower_id UUID REFERENCES users(UserID),
    followed_id UUID REFERENCES users(UserID),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, followed_id)
);`

    _, err := db.Exec(query)
    return err
}