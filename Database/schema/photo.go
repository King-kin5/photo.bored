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
			photo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			filename VARCHAR(255) NOT NULL UNIQUE,
			data BYTEA NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			mime_type VARCHAR(100) NOT NULL,
			location VARCHAR(255) DEFAULT 'Unknown Location',
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			caption TEXT DEFAULT ''
		);

		CREATE INDEX IF NOT EXISTS idx_photos_user_id ON photos(user_id);
		CREATE INDEX IF NOT EXISTS idx_photos_created_at ON photos(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_photos_filename ON photos(filename);
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
			comment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			photo_id UUID NOT NULL REFERENCES photos(photo_id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
		CREATE INDEX IF NOT EXISTS idx_comments_photo_id ON comments(photo_id);
		CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);
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

// Fixed: Corrected table references
func CreateLikesTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS likes (
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			photo_id UUID NOT NULL REFERENCES photos(photo_id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, photo_id)
		);

		CREATE INDEX IF NOT EXISTS idx_likes_user_id ON likes(user_id);
		CREATE INDEX IF NOT EXISTS idx_likes_photo_id ON likes(photo_id);
		CREATE INDEX IF NOT EXISTS idx_likes_created_at ON likes(created_at DESC);
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

// Fixed: Corrected table references
func CreateAlbumsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS albums (
			album_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			description TEXT DEFAULT ''
		);

		CREATE INDEX IF NOT EXISTS idx_albums_user_id ON albums(user_id);
		CREATE INDEX IF NOT EXISTS idx_albums_created_at ON albums(created_at DESC);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create albums table: %v", err)
			return fmt.Errorf("failed to create albums table: %w", err)
		}
	}

	utils.Logger.Info("Albums table created successfully")
	return nil
}

// Fixed: Corrected table references
func CreateFollowsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS follows (
			follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (follower_id, following_id),
			CHECK (follower_id != following_id)
		);

		CREATE INDEX IF NOT EXISTS idx_follows_follower_id ON follows(follower_id);
		CREATE INDEX IF NOT EXISTS idx_follows_following_id ON follows(following_id);
		CREATE INDEX IF NOT EXISTS idx_follows_created_at ON follows(created_at DESC);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create follows table: %v", err)
			return fmt.Errorf("failed to create follows table: %w", err)
		}
	}

	utils.Logger.Info("Follows table created successfully")
	return nil
}

// Fixed: Corrected table references


// CreateAlbumPhotosTable creates the album_photos junction table
func CreateAlbumPhotosTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS album_photos (
			album_id UUID NOT NULL REFERENCES albums(album_id) ON DELETE CASCADE,
			photo_id UUID NOT NULL REFERENCES photos(photo_id) ON DELETE CASCADE,
			added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (album_id, photo_id)
		);

		CREATE INDEX IF NOT EXISTS idx_album_photos_album_id ON album_photos(album_id);
		CREATE INDEX IF NOT EXISTS idx_album_photos_photo_id ON album_photos(photo_id);
		CREATE INDEX IF NOT EXISTS idx_album_photos_added_at ON album_photos(added_at DESC);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create album_photos table: %v", err)
			return fmt.Errorf("failed to create album_photos table: %w", err)
		}
	}

	utils.Logger.Info("Album_photos table created successfully")
	return nil
}

// CreateFollowsTable creates the follows table for user following functionality


// CreatePhotoViewsTable creates a table to track photo views for analytics
func CreatePhotoViewsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS photo_views (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			photo_id UUID NOT NULL REFERENCES photos(photo_id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Nullable for anonymous views
			ip_address INET,
			user_agent TEXT,
			viewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_photo_views_photo_id ON photo_views(photo_id);
		CREATE INDEX IF NOT EXISTS idx_photo_views_user_id ON photo_views(user_id);
		CREATE INDEX IF NOT EXISTS idx_photo_views_viewed_at ON photo_views(viewed_at DESC);
		CREATE INDEX IF NOT EXISTS idx_photo_views_ip_address ON photo_views(ip_address);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create photo_views table: %v", err)
			return fmt.Errorf("failed to create photo_views table: %w", err)
		}
	}

	utils.Logger.Info("Photo_views table created successfully")
	return nil
}

// CreatePhotoTagsTable creates a table for photo tags
func CreatePhotoTagsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS photo_tags (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			photo_id UUID NOT NULL REFERENCES photos(photo_id) ON DELETE CASCADE,
			tag_name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(photo_id, tag_name)
		);

		CREATE INDEX IF NOT EXISTS idx_photo_tags_photo_id ON photo_tags(photo_id);
		CREATE INDEX IF NOT EXISTS idx_photo_tags_tag_name ON photo_tags(tag_name);
		CREATE INDEX IF NOT EXISTS idx_photo_tags_created_at ON photo_tags(created_at DESC);
	`

	_, err := db.Exec(query)
	if err != nil {
		dbErrStr := err.Error()
		if !(strings.Contains(dbErrStr, "already exists") || strings.Contains(dbErrStr, "duplicate key value")) {
			utils.Logger.Errorf("Failed to create photo_tags table: %v", err)
			return fmt.Errorf("failed to create photo_tags table: %w", err)
		}
	}

	utils.Logger.Info("Photo_tags table created successfully")
	return nil
}

// CreateAllPhotoTables creates all photo-related tables in the correct order
func CreateAllPhotoTables(db *sql.DB) error {
	tables := []struct {
		name string
		fn   func(*sql.DB) error
	}{
		{"photos", CreatePhotosTable},
		{"comments", CreateCommentsTable},
		{"likes", CreateLikesTable},
		{"albums", CreateAlbumsTable},
		{"album_photos", CreateAlbumPhotosTable},
		{"follows", CreateFollowsTable},
		{"photo_views", CreatePhotoViewsTable},
		{"photo_tags", CreatePhotoTagsTable},
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