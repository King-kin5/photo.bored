package photo

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
)



type Photostore struct{
    db *sql.DB
}
func NewPhotostore(db *sql.DB) *Photostore{
	return &Photostore{db:db}
	
}
func (im *Photostore) Addimage(photo *Photo) error {
    query := `INSERT INTO photos (photo_id, Filename, Data, Date, Location, UserID) VALUES ($1, $2, $3, $4, $5, $6)`
    _, err := im.db.Exec(query, photo.PhotoID, photo.Filename, photo.Data, photo.Date, photo.Location, photo.UserID)
    if err != nil {
        log.Printf("Error creating image: %v", err)
    }
    return err
}

func (im*Photostore)GetImagebyID(imageID string)(*Photo,error){
	image:=new(Photo)
	query := `SELECT photo_id, Filename,Data, Date, Location FROM images WHERE ID=$1`
	row := im.db.QueryRow(query, imageID)
	err := row.Scan(&image.PhotoID, &image.Filename,&image.Data, &image.Date, &image.Location)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No image found, return nil for image
		}
		log.Printf("Error scanning row: %v", err)
		return nil, err // Return the error if something else went wrong
	}
	return image, nil
}
//func (is *Photostore) GetImagesByUserID(userID string) ([]*Photo, error) {
	//query := `SELECT ID, Filename, Data,Date, Location, UserID FROM images WHERE UserID=$1`
	//rows, err := is.db.Query(query, userID)
	//if err != nil {
		//log.Printf("Error querying images: %v", err)
		//return nil, err
	//}
	//defer rows.Close()

	//var images []*Photo
	//for rows.Next() {
		//image := new(Photo)
		//err := rows.Scan(&image.ID, &image.Filename,&image.Data, &image.Date, &image.Location)
		//if err != nil {
			//log.Printf("Error scanning row: %v", err)
			//continue
		//}
		//images = append(images, image)
	//}

	//return images, nil
//}

func (ps *Photostore) GetImageByFilename(filename string) (*Photo, error) {
	query := `SELECT photo_id , Filename, Data, Date, Location FROM photos WHERE Filename = $1`
	row := ps.db.QueryRow(query, filename)

	photo := &Photo{}
	err := row.Scan(&photo.PhotoID, &photo.Filename, &photo.Data, &photo.Date, &photo.Location)
	if err != nil {
		return nil, err
	}

	return photo, nil
}
func (ps *Photostore) GetPaginatedPhotos(limit, offset int) ([]PhotoWithUser, error) {
    query := `
    SELECT p.photo_id, p.filename, p.date, p.location, p.caption,
        u.user_id, u.username,
        COALESCE((SELECT COUNT(*) FROM photo_likes WHERE photo_id = p.photo_id), 0) as likes_count,
        COALESCE((SELECT COUNT(*) FROM photo_comments WHERE photo_id = p.photo_id), 0) as comments_count
        FROM photos p
        JOIN users u ON p.user_id = u.user_id
        ORDER BY p.date DESC
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

// GetUserFeed gets photos from users that the current user follows
func (ps *Photostore) GetUserFeed(userID uuid.UUID, limit, offset int) ([]PhotoWithUser, error) {
    query := `
        SELECT p.photo_id,p.filename,p.date,p.location,p.caption,u.user_id,u.username,
        COALESCE((SELECT COUNT(*) FROM photo_likes WHERE photo_id = p.photo_id), 0) as likes_count,
        COALESCE((SELECT COUNT(*) FROM photo_comments WHERE photo_id = p.photo_id), 0) as comments_count
        FROM photos p
        JOIN users u ON p.user_id = u.user_id
        WHERE p.user_id IN (
            SELECT followed_id 
            FROM user_follows 
            WHERE follower_id = $1
        ) OR p.user_id = $1
        ORDER BY p.date DESC
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
// GetPhotoComments gets comments for a specific photo
func (ps *Photostore) GetPhotoComments(photoID uuid.UUID) ([]Comment, error) {
    query := `
        SELECT 
            c.comment_id,
            c.content,
            c.created_at,
            u.user_id,
            u.username
        FROM photo_comments c
        JOIN users u ON c.user_id = u.user_id
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
	query := `SELECT photo_id, Filename, Data, Date, Location FROM photos WHERE Filename = $1`
	row := ps.db.QueryRow(query, filename)

	photo := &Photo{}
	err := row.Scan(&photo.PhotoID, &photo.Filename, &photo.Data, &photo.Date, &photo.Location)
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
func (s *Photostore) Createalbum(album *Album) error {
	query := "INSERT INTO albums (album_id,Name, Created_at) VALUES ($1,$2,$3) RETURNING id"
	_, err := s.db.Exec(query,album.Album_id, album.Name,album.CreatedAt)
	if err != nil {
		log.Printf("Error creating Album: %v", err)	
	}
	return err

}         
//add an existing photo to an album 
func (s *Photostore) AddPhotoToAlbum(albumID uuid.UUID, photoID uuid.UUID) error {
    // Correct table name from album_photos to album_photo
    query := `INSERT INTO album_photo (album_id, photo_id) VALUES ($1, $2)`
    _, err := s.db.Exec(query, albumID, photoID)
    if err != nil {
        log.Printf("Error adding photo to album: %v", err)
        return err
    }
    return nil
}
func (s *Photostore) GetAlbumWithPhotos(albumID uuid.UUID) (*Album, error) {
    // Query to get album details
    albumQuery := `SELECT album_id, name, created_at FROM albums WHERE id = $1`
    var album Album
    err := s.db.QueryRow(albumQuery, albumID).Scan(&album.Album_id, &album.Name, &album.CreatedAt)
    if err != nil {
        log.Printf("Error retrieving album: %v", err)
        return nil, err
    }
    // Query to get photos associated with the album
    photoQuery := `
        SELECT p.photo_id, p.filename, p.data, p.date, p.location 
        FROM photos p 
        INNER JOIN album_photo ap ON p.photo_id = ap.photo_id 
        WHERE ap.album_id = $1`
    
    rows, err := s.db.Query(photoQuery, albumID)
    if err != nil {
        log.Printf("Error retrieving photos for album: %v", err)
        return nil, err
    }
    defer rows.Close()
    for rows.Next() {
        var photo Photo
        if err := rows.Scan(&photo.PhotoID, &photo.Filename, &photo.Data, &photo.Date, &photo.Location); err != nil {
            log.Printf("Error scanning photo: %v", err)
            return nil, err
        }
        album.Photos = append(album.Photos, photo)
    }
    return &album, nil
}