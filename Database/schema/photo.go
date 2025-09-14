
package schema

import (
	"database/sql"
	"fmt"
	utils "app/pkg/utils"
	"strings"
)

// CreatePhotosTable creates the photos table
func CreatePhotosTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS photos (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			filename VARCHAR(255) NOT NULL,
			caption TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_photos_user_id ON photos(user_id);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create photos table: %v", err)
			return fmt.Errorf("failed to create photos table: %w", err)
		}
	}

	utils.Logger.Info("Photos table created successfully")
	return nil
}

// CreateCommentsTable creates the comments table
func CreateCommentsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			photo_id UUID NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
		CREATE INDEX IF NOT EXISTS idx_comments_photo_id ON comments(photo_id);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create comments table: %v", err)
			return fmt.Errorf("failed to create comments table: %w", err)
		}
	}

	utils.Logger.Info("Comments table created successfully")
	return nil
}

// CreateLikesTable creates the likes table
func CreateLikesTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS likes (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			photo_id UUID NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (user_id, photo_id)
		);

		CREATE INDEX IF NOT EXISTS idx_likes_user_id ON likes(user_id);
		CREATE INDEX IF NOT EXISTS idx_likes_photo_id ON likes(photo_id);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create likes table: %v", err)
			return fmt.Errorf("failed to create likes table: %w", err)
		}
	}

	utils.Logger.Info("Likes table created successfully")
	return nil
}

// CreateAllPhotoTables creates all photo-related tables
func CreateAllPhotoTables(db *sql.DB) error {
	tables := []struct {
		name string
		fn   func(*sql.DB) error
	}{
		{"photos", CreatePhotosTable},
		{"comments", CreateCommentsTable},
		{"likes", CreateLikesTable},
	}

	for _, table := range tables {
		utils.Logger.Infof("Creating %s table...", table.name)
		if err := table.fn(db); err != nil {
			return fmt.Errorf("failed to create %s table: %w", table.name, err)
		}
	}

	utils.Logger.Info("All photo tables created successfully")
	return nil
}
