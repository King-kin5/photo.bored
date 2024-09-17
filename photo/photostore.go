package photo

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
)

type Photostore struct {
	db *sql.DB
}

func NewPhotostore(db *sql.DB) *Photostore {
	return &Photostore{db: db}

}

func (im *Photostore) Addimage(photo *Photo) error {
	query := `INSERT INTO photos (photo_id, Filename, Data, Date, Location) VALUES ($1, $2, $3, $4, $5)`
	_, err := im.db.Exec(query, photo.photo_id, photo.Filename, photo.Data, photo.Date, photo.Location)
	if err != nil {
		log.Printf("Error creating image: %v", err)
	}
	return err
}

func (im *Photostore) GetImagebyID(imageID string) (*Photo, error) {
	image := new(Photo)
	query := `SELECT photo_id, Filename,Data, Date, Location FROM images WHERE ID=$1`
	row := im.db.QueryRow(query, imageID)
	err := row.Scan(&image.photo_id, &image.Filename, &image.Data, &image.Date, &image.Location)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No image found, return nil for image
		}
		log.Printf("Error scanning row: %v", err)
		return nil, err // Return the error if something else went wrong
	}
	return image, nil
}

// func (is *Photostore) GetImagesByUserID(userID string) ([]*Photo, error) {
// 	query := `SELECT ID, Filename, Data,Date, Location, UserID FROM images WHERE UserID=$1`
// 	rows, err := is.db.Query(query, userID)
// 	if err != nil {
// 		log.Printf("Error querying images: %v", err)
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var images []*Photo
// 	for rows.Next() {
// 		image := new(Photo)
// 		err := rows.Scan(&image.ID, &image.Filename, &image.Data, &image.Date, &image.Location)
// 		if err != nil {
// 			log.Printf("Error scanning row: %v", err)
// 			continue
// 		}
// 		images = append(images, image)
// 	}

// 	return images, nil
// }

func (ps *Photostore) GetImageByFilename(filename string) (*Photo, error) {
	query := `SELECT photo_id , Filename, Data, Date, Location FROM photos WHERE Filename = $1`
	row := ps.db.QueryRow(query, filename)

	photo := &Photo{}
	err := row.Scan(&photo.photo_id, &photo.Filename, &photo.Data, &photo.Date, &photo.Location)
	if err != nil {
		return nil, err
	}

	return photo, nil
}

func (ps *Photostore) DeleteImageByFilename(filename string) (*Photo, error) {
	query := `SELECT photo_id, Filename, Data, Date, Location FROM photos WHERE Filename = $1`
	row := ps.db.QueryRow(query, filename)

	photo := &Photo{}
	err := row.Scan(&photo.photo_id, &photo.Filename, &photo.Data, &photo.Date, &photo.Location)
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
	query := "INSERT INTO albums (ID,Name,CreatedAt) VALUES ($1,$2,$3) RETURNING id"
	_, err := s.db.Exec(query, album.Name)
	if err != nil {
		log.Printf("Error creating Album: %v", err)
	}
	return err

}

// add an existing photo to an album
func (s *Photostore) AddPhotoToAlbum(albumID uuid.UUID, photoID uuid.UUID) error {
	query := `INSERT INTO album_photos (album_id, photo_id) VALUES ($1, $2)`
	_, err := s.db.Exec(query, albumID, photoID)
	if err != nil {
		log.Printf("Error adding photo to album: %v", err)
		return err
	}
	return nil
}
