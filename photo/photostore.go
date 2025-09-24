package photo

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type Photostore struct {
	db *sql.DB
}

func NewPhotostore(db *sql.DB) *Photostore {
	return &Photostore{db: db}
}

func (ps *Photostore) Addimage(photo *Photo) error {
	query := `INSERT INTO photos (photo_id, filename, data, created_at, mime_type, location, user_id, caption) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := ps.db.Exec(query, photo.PhotoID, photo.Filename, photo.Data, photo.Date, photo.MIMEType, photo.Location, photo.UserID, photo.Caption)
	if err != nil {
		log.Printf("Error creating image: %v", err)
	}
	return err
}

func (ps *Photostore) GetImageByID(imageID string) (*Photo, error) {
	image := new(Photo)
	query := `SELECT photo_id, filename, data, created_at, mime_type, location, user_id, caption FROM photos WHERE photo_id=$1`
	row := ps.db.QueryRow(query, imageID)
	err := row.Scan(&image.PhotoID, &image.Filename, &image.Data, &image.Date, &image.MIMEType, &image.Location, &image.UserID, &image.Caption)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("Error scanning row: %v", err)
		return nil, err
	}
	return image, nil
}

func (ps *Photostore) GetImageByFilename(filename string) (*Photo, error) {
	query := `SELECT photo_id, filename, data, created_at, mime_type, location, user_id, caption FROM photos WHERE filename = $1`
	row := ps.db.QueryRow(query, filename)

	photo := &Photo{}
	err := row.Scan(&photo.PhotoID, &photo.Filename, &photo.Data, &photo.Date, &photo.MIMEType, &photo.Location, &photo.UserID, &photo.Caption)
	if err != nil {
		return nil, err
	}

	return photo, nil
}

func (ps *Photostore) GetPaginatedPhotos(limit, offset int) ([]PhotoWithUser, error) {
	query := `
	SELECT p.photo_id, p.filename, p.created_at, p.location, p.caption,
		u.id, u.username,
		COALESCE((SELECT COUNT(*) FROM likes WHERE photo_id = p.photo_id), 0) as likes_count,
		COALESCE((SELECT COUNT(*) FROM comments WHERE photo_id = p.photo_id), 0) as comments_count
		FROM photos p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := ps.db.Query(query, limit, offset)
	if err != nil {
		log.Printf("Error querying paginated photos: %v", err)
		return nil, err
	}
	defer rows.Close()

	var photos []PhotoWithUser
	for rows.Next() {
		var photo PhotoWithUser
		err := rows.Scan(
			&photo.PhotoID,
			&photo.Filename,
			&photo.Date,
			&photo.Location,
			&photo.Caption,
			&photo.UserID,
			&photo.Username,
			&photo.LikesCount,
			&photo.CommentsCount,
		)
		if err != nil {
			log.Printf("Error scanning photo row: %v", err)
			continue
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

func (ps *Photostore) GetUserFeed(userID uuid.UUID, limit, offset int) ([]PhotoWithUser, error) {
	query := `
		SELECT p.photo_id, p.filename, p.created_at, p.location, p.caption, u.id, u.username,
		COALESCE((SELECT COUNT(*) FROM likes WHERE photo_id = p.photo_id), 0) as likes_count,
		COALESCE((SELECT COUNT(*) FROM comments WHERE photo_id = p.photo_id), 0) as comments_count
		FROM photos p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id IN (
			SELECT following_id 
			FROM follows 
			WHERE follower_id = $1
		) OR p.user_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := ps.db.Query(query, userID, limit, offset)
	if err != nil {
		log.Printf("Error querying user feed: %v", err)
		return nil, err
	}
	defer rows.Close()

	var photos []PhotoWithUser
	for rows.Next() {
		var photo PhotoWithUser
		err := rows.Scan(
			&photo.PhotoID,
			&photo.Filename,
			&photo.Date,
			&photo.Location,
			&photo.Caption,
			&photo.UserID,
			&photo.Username,
			&photo.LikesCount,
			&photo.CommentsCount,
		)
		if err != nil {
			log.Printf("Error scanning photo row: %v", err)
			continue
		}
		photos = append(photos, photo)
	}
	return photos, nil
}

func (ps *Photostore) GetPhotoComments(photoID uuid.UUID) ([]Comment, error) {
	query := `
		SELECT 
			c.comment_id,
			c.content,
			c.created_at,
			u.id,
			u.username
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.photo_id = $1
		ORDER BY c.created_at DESC
	`
	rows, err := ps.db.Query(query, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var comments []Comment
	for rows.Next() {
		var comment Comment
		comment.PhotoID = photoID // Set the PhotoID since it's not selected in the query
		err := rows.Scan(
			&comment.CommentID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UserID,
			&comment.Username,
		)
		if err != nil {
			log.Printf("Error scanning comment: %v", err)
			continue
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (ps *Photostore) DeleteImageByFilename(filename string) (*Photo, error) {
	photo, err := ps.GetImageByFilename(filename)
	if err != nil {
		return nil, err
	}
	if photo == nil {
		return nil, sql.ErrNoRows
	}
	query := `DELETE FROM photos WHERE filename = $1`
	_, err = ps.db.Exec(query, filename)
	if err != nil {
		return nil, err
	}
	return photo, nil
}

func (ps *Photostore) UpdateCaptionByFilename(filename, caption string) error {
	query := "UPDATE photos SET caption = $1 WHERE filename = $2"
	_, err := ps.db.Exec(query, caption, filename)
	return err
}

func (ps *Photostore) Createalbum(album *Album) error {
	query := "INSERT INTO albums (album_id, name, user_id, created_at) VALUES ($1, $2, $3, $4)"
	_, err := ps.db.Exec(query, album.AlbumID, album.Name, album.UserID, album.CreatedAt)
	if err != nil {
		log.Printf("Error creating Album: %v", err)
	}
	return err
}

func (ps *Photostore) AddPhotoToAlbum(albumID uuid.UUID, photoID uuid.UUID) error {
	query := `INSERT INTO album_photos (album_id, photo_id) VALUES ($1, $2)`
	_, err := ps.db.Exec(query, albumID, photoID)
	if err != nil {
		log.Printf("Error adding photo to album: %v", err)
		return err
	}
	return nil
}

func (ps *Photostore) GetAlbumWithPhotos(albumID uuid.UUID) (*Album, error) {
	albumQuery := `SELECT album_id, name, user_id, created_at FROM albums WHERE album_id = $1`
	var album Album
	err := ps.db.QueryRow(albumQuery, albumID).Scan(&album.AlbumID, &album.Name, &album.UserID, &album.CreatedAt)
	if err != nil {
		log.Printf("Error retrieving album: %v", err)
		return nil, err
	}
	photoQuery := `
		SELECT p.photo_id, p.filename, p.data, p.created_at, p.mime_type, p.location, p.user_id, p.caption
		FROM photos p 
		INNER JOIN album_photos ap ON p.photo_id = ap.photo_id 
		WHERE ap.album_id = $1`

	rows, err := ps.db.Query(photoQuery, albumID)
	if err != nil {
		log.Printf("Error retrieving photos for album: %v", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var photo Photo
		if err := rows.Scan(&photo.PhotoID, &photo.Filename, &photo.Data, &photo.Date, &photo.MIMEType, &photo.Location, &photo.UserID, &photo.Caption); err != nil {
			log.Printf("Error scanning photo: %v", err)
			return nil, err
		}
		album.Photos = append(album.Photos, photo)
	}

	return &album, nil
}

// FIXED: All infinite scroll queries now use consistent user table references
func (s *Photostore) GetInfiniteScrollPhotos(limit int, cursor *time.Time) ([]PhotoWithUser, *time.Time, error) {
    query := `
        SELECT 
            p.photo_id,
            p.filename,
            p.created_at as date,
            p.location,
            p.caption,
            p.user_id,
            u.username,
            COALESCE(l.like_count, 0) as likes_count,
            COALESCE(c.comment_count, 0) as comments_count
        FROM photos p
        INNER JOIN users u ON p.user_id = u.id  -- FIXED: Use u.id consistently
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as like_count 
            FROM likes 
            GROUP BY photo_id
        ) l ON p.photo_id = l.photo_id
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as comment_count 
            FROM comments 
            GROUP BY photo_id
        ) c ON p.photo_id = c.photo_id
        WHERE ($1::timestamp IS NULL OR p.created_at < $1)
        ORDER BY p.created_at DESC
        LIMIT $2;
    `
    var args []interface{}
    if cursor != nil {
        args = []interface{}{*cursor, limit}
    } else {
        args = []interface{}{nil, limit}
    }
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, nil, fmt.Errorf("error querying infinite scroll photos: %w", err)
    }
    defer rows.Close()
    
    var photos []PhotoWithUser
    var lastDate *time.Time
    
    for rows.Next() {
        var photo PhotoWithUser
        err := rows.Scan(
            &photo.PhotoID,
            &photo.Filename,
            &photo.Date,
            &photo.Location,
            &photo.Caption,
            &photo.UserID,
            &photo.Username,
            &photo.LikesCount,
            &photo.CommentsCount,
        )
        if err != nil {
            log.Printf("Error scanning photo row: %v", err)
            continue
        }
        
        // ADDED: Skip photos with empty/null filenames
        if photo.Filename == "" {
            log.Printf("Skipping photo %s with empty filename", photo.PhotoID.String())
            continue
        }
        
        photos = append(photos, photo)
        lastDate = &photo.Date
    }
   
    return photos, lastDate, nil
}

func (s *Photostore) GetUserFeedInfinite(userID uuid.UUID, limit int, cursor *time.Time) ([]PhotoWithUser, *time.Time, error) {
    query := `
        SELECT 
            p.photo_id,
            p.filename,
            p.created_at as date,
            p.location,
            p.caption,
            p.user_id,
            u.username,
            COALESCE(l.like_count, 0) as likes_count,
            COALESCE(c.comment_count, 0) as comments_count
        FROM photos p
        INNER JOIN users u ON p.user_id = u.id  -- FIXED: Use u.id consistently
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as like_count 
            FROM likes 
            GROUP BY photo_id
        ) l ON p.photo_id = l.photo_id
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as comment_count 
            FROM comments 
            GROUP BY photo_id
        ) c ON p.photo_id = c.photo_id
        WHERE 
            ($1::timestamp IS NULL OR p.created_at < $1)
            AND (
                p.user_id = $3  -- User's own photos
                OR EXISTS (     -- Photos from followed users
                    SELECT 1 FROM follows f 
                    WHERE f.follower_id = $3 AND f.following_id = p.user_id
                )
            )
        ORDER BY p.created_at DESC
        LIMIT $2;
    `
    var args []interface{}
    if cursor != nil {
        args = []interface{}{*cursor, limit, userID}
    } else {
        args = []interface{}{nil, limit, userID}
    }
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, nil, fmt.Errorf("error querying user feed infinite: %w", err)
    }
    defer rows.Close()
    
    var photos []PhotoWithUser
    var lastDate *time.Time
    
    for rows.Next() {
        var photo PhotoWithUser
        err := rows.Scan(
            &photo.PhotoID,
            &photo.Filename,
            &photo.Date,
            &photo.Location,
            &photo.Caption,
            &photo.UserID,
            &photo.Username,
            &photo.LikesCount,
            &photo.CommentsCount,
        )
        if err != nil {
            log.Printf("Error scanning photo row: %v", err)
            continue
        }
        
        // ADDED: Skip photos with empty/null filenames
        if photo.Filename == "" {
            log.Printf("Skipping photo %s with empty filename", photo.PhotoID.String())
            continue
        }
        
        photos = append(photos, photo)
        lastDate = &photo.Date
    }   
    return photos, lastDate, nil
}

func (s *Photostore) GetUserPersonalFeedInfinite(userID uuid.UUID, limit int, cursor *time.Time) ([]PhotoWithUser, *time.Time, error) {
    query := `
        SELECT 
            p.photo_id,
            p.filename,
            p.created_at as date,
            p.location,
            p.caption,
            p.user_id,
            u.username,
            COALESCE(l.like_count, 0) as likes_count,
            COALESCE(c.comment_count, 0) as comments_count
        FROM photos p
        INNER JOIN users u ON p.user_id = u.id  -- FIXED: Use u.id consistently
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as like_count 
            FROM likes 
            GROUP BY photo_id
        ) l ON p.photo_id = l.photo_id
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as comment_count 
            FROM comments 
            GROUP BY photo_id
        ) c ON p.photo_id = c.photo_id
        WHERE 
            p.user_id = $3  -- Only user's own photos
            AND ($1::timestamp IS NULL OR p.created_at < $1)
        ORDER BY p.created_at DESC
        LIMIT $2;
    `
    var args []interface{}
    if cursor != nil {
        args = []interface{}{*cursor, limit, userID}
    } else {
        args = []interface{}{nil, limit, userID}
    }
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, nil, fmt.Errorf("error querying user personal feed: %w", err)
    }
    defer rows.Close()
    
    var photos []PhotoWithUser
    var lastDate *time.Time
    
    for rows.Next() {
        var photo PhotoWithUser
        err := rows.Scan(
            &photo.PhotoID,
            &photo.Filename,
            &photo.Date,
            &photo.Location,
            &photo.Caption,
            &photo.UserID,
            &photo.Username,
            &photo.LikesCount,
            &photo.CommentsCount,
        )
        if err != nil {
            log.Printf("Error scanning photo row: %v", err)
            continue
        }
        
        // ADDED: Skip photos with empty/null filenames
        if photo.Filename == "" {
            log.Printf("Skipping photo %s with empty filename", photo.PhotoID.String())
            continue
        }
        
        photos = append(photos, photo)
        lastDate = &photo.Date
    }
    return photos, lastDate, nil
}

func (s *Photostore) GetFollowingFeedInfinite(userID uuid.UUID, limit int, cursor *time.Time) ([]PhotoWithUser, *time.Time, error) {
    query := `
        SELECT 
            p.photo_id, 
            p.filename, 
            p.created_at as date, 
            p.location, 
            p.caption, 
            p.user_id,
            u.username, 
            COALESCE(l.like_count, 0) as likes_count, 
            COALESCE(c.comment_count, 0) as comments_count
        FROM photos p
        INNER JOIN users u ON p.user_id = u.id  -- FIXED: Use u.id consistently
        INNER JOIN follows f ON f.following_id = p.user_id
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as like_count 
            FROM likes 
            GROUP BY photo_id
        ) l ON p.photo_id = l.photo_id
        LEFT JOIN (
            SELECT photo_id, COUNT(*) as comment_count 
            FROM comments 
            GROUP BY photo_id
        ) c ON p.photo_id = c.photo_id
        WHERE 
            f.follower_id = $3 
            AND ($1::timestamp IS NULL OR p.created_at < $1)
        ORDER BY p.created_at DESC
        LIMIT $2;
    `
    var args []interface{}
    if cursor != nil {
        args = []interface{}{*cursor, limit, userID}
    } else {
        args = []interface{}{nil, limit, userID}
    }    
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, nil, fmt.Errorf("error querying following feed infinite: %w", err)
    }
    defer rows.Close()
    
    var photos []PhotoWithUser
    var lastDate *time.Time
    
    for rows.Next() {
        var photo PhotoWithUser
        err := rows.Scan(
            &photo.PhotoID,
            &photo.Filename,
            &photo.Date,
            &photo.Location,
            &photo.Caption,
            &photo.UserID,
            &photo.Username,
            &photo.LikesCount,
            &photo.CommentsCount,
        )
        if err != nil {
            log.Printf("Error scanning following feed photo row: %v", err)
            continue
        }
        
        // ADDED: Skip photos with empty/null filenames
        if photo.Filename == "" {
            log.Printf("Skipping photo %s with empty filename", photo.PhotoID.String())
            continue
        }
        
        photos = append(photos, photo)
        lastDate = &photo.Date
    }
    return photos, lastDate, nil
}