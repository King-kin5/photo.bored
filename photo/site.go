package photo

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
	"strings"
	"sync"
	"path/filepath"
	"mime/multipart"
	"image"
	"image/jpeg"
	"image/png"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
	"github.com/disintegration/imaging"
)

type Photo struct {         
	PhotoID  uuid.UUID `json:"photo_id"`
	Filename string    `json:"filename"`
	Data     []byte    `json:"-"` 
	Date     time.Time `json:"date"`
	MIMEType string    `json:"mime_type"`
	Location string    `json:"location"`
	UserID  uuid.UUID    `json:"UserID"`
}

type Album struct {
	Album_id         uuid.UUID `json:"album_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	Photos     []Photo   `json:"photos"`
}

//Constantsn for file Validation and imageProcessing

const(
	maxFileSize = 10<<20
	MaxUploadFiles=10
	thumbSize=800    // maximum dimension for thumbnails
)
// ValidImageTypes maps allowed MIME types to their file extensions
var ValidImageTypes=map[string]string{
	"image/jpeg":".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",

}
// RateLimiter implements a simple rate limiting mechanism
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
	}
}

func (r *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(2, 5) // 2 requests per second with burst of 5
		r.visitors[ip] = limiter
	}

	return limiter
}
// validateFile performs security checks on uploaded files
func validateFile(header *multipart.FileHeader) error {
	if header.Size > maxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif"}
	valid := false
	for _, validExt := range validExts {
		if ext == validExt {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid file type: %s", ext)
	}

	return nil
}

// optimizeImage resizes and compresses the image for storage
func optimizeImage(img image.Image, format string) ([]byte, error) {
	// Resize image if it's too large
	bounds := img.Bounds()
	if bounds.Dx() > thumbSize || bounds.Dy() > thumbSize {
		img = imaging.Fit(img, thumbSize, thumbSize, imaging.Lanczos)
	}

	var buf bytes.Buffer
	var err error

	switch format {
	case "image/jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	case "image/png":
		err = png.Encode(&buf, img)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}



func (m *Model) HandleUpload(c echo.Context) error {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		return c.String(http.StatusBadRequest, "Error parsing form")
	}
		// Apply rate limiting
	limiter := m.rateLimiter.GetLimiter(c.RealIP())
	if !limiter.Allow() {
		return c.String(http.StatusTooManyRequests, "Rate limit exceeded")
	}
	

	files := form.File["images"]
	if len(files) == 0 {
		return c.String(http.StatusBadRequest, "No files uploaded")
	}

	if len(files) > MaxUploadFiles {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Maximum %d files allowed per upload", MaxUploadFiles))
	}

	var uploadedImages []string

	for _, fileHeader := range files {
		if err := validateFile(fileHeader); err != nil {
			log.Printf("File validation failed for %s: %v", fileHeader.Filename, err)
			continue
		}
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Error opening file %s: %v", fileHeader.Filename, err)
			continue
		}
		defer file.Close()

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, file); err != nil {
			log.Printf("Failed to read file %s: %v", fileHeader.Filename, err)
			continue
		}

		mimeType := http.DetectContentType(buf.Bytes())

		// Decode the image for optimization
		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Printf("Failed to decode image %s: %v", fileHeader.Filename, err)
			continue
		}

		// Optimize the image
		optimizedData, err := optimizeImage(img, mimeType)
		if err != nil {
			log.Printf("Failed to optimize image %s: %v", fileHeader.Filename, err)
			continue
		}

		image := &Photo{
			PhotoID:  uuid.New(),
			Filename: fileHeader.Filename,
			Data:     optimizedData,
			Date:     time.Now(),
			Location: "Unknown Location",
			MIMEType: mimeType,
		}

		if err := m.store.Addimage(image); err != nil {
			log.Printf("Failed to save image %s: %v", fileHeader.Filename, err)
			continue
		}
		// Broadcast the new photo to WebSocket clients
		m.wsManager.Broadcast <- *image

		imageURL := fmt.Sprintf("/serveimage/%s", image.Filename)
		uploadedImages = append(uploadedImages, imageURL)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Successfully uploaded %d images", len(uploadedImages)),
		"images":  uploadedImages,
	})
}

func (m *Model) HandleServeImage(c echo.Context) error {
	filename := c.Param("filename")
	
	photo, err := m.store.GetImageByFilename(filename)
	if err != nil {
		log.Printf("Image not found: %v", filename)
		return c.String(http.StatusNotFound, "Image not found")
	}

	if photo == nil {
		return c.String(http.StatusNotFound, "Image not found")
	}

	return c.Blob(http.StatusOK, photo.MIMEType, photo.Data)
}

func (m *Model) HandleDeleteImage(c echo.Context) error {
	filename := c.Param("filename")
	
	photo, err := m.store.DeleteImageByFilename(filename)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Could not delete image")
	}

	if photo == nil {
		return c.String(http.StatusNotFound, "Image not found")
	}

	return c.String(http.StatusOK, fmt.Sprintf("Image %s deleted successfully", photo.Filename))
}

func (m *Model) HandleCaption(c echo.Context) error {
	filename := c.Param("filename")
	newCaption := c.FormValue("caption")

	if newCaption == "" {
		return c.String(http.StatusBadRequest, "Caption cannot be empty")
	}

	err := m.store.UpdateCaptionByFilename(filename, newCaption)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update caption")
	}

	return c.String(http.StatusOK, "Caption updated successfully")
}

func (m *Model) HandleCreateAlbum(c echo.Context) error {
	name := c.FormValue("Name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Album name is required")
	}

	album := &Album{
		Album_id:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now(),
	}

	err := m.store.Createalbum(album)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create album")
	}

	return c.String(http.StatusOK, fmt.Sprintf("Album created successfully with ID: %s", album.Album_id))
}

func (m *Model) HandleAddPhotoToAlbum(c echo.Context) error {
	albumID, err := uuid.Parse(c.FormValue("AlbumID"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid AlbumID")
	}

	photoID, err := uuid.Parse(c.FormValue("PhotoID"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid PhotoID")
	}

	err = m.store.AddPhotoToAlbum(albumID, photoID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to add photo to album")
	}

	return c.String(http.StatusOK, "Photo added to album successfully")
}

func (m *Model) HandleGetAlbumWithPhotos(c echo.Context) error {
	albumID, err := uuid.Parse(c.QueryParam("AlbumID"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid AlbumID")
	}

	album, err := m.store.GetAlbumWithPhotos(albumID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve album and photos")
	}

	// Remove binary data and set location for each photo
	for i := range album.Photos {
		album.Photos[i].Data = nil
		album.Photos[i].Location = fmt.Sprintf("/serveimage/%s", album.Photos[i].Filename)
	}

	return c.JSON(http.StatusOK, album)
}

// Helper function to get user ID from JWT token
func getUserIDFromToken(c echo.Context) (uuid.UUID, error) {
    // Implementation depends on your JWT token structure
    // This is a placeholder - implement based on your authentication setup
    user := c.Get("user") // Assuming you store the user in context
    if user == nil {
        return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
    }
    
    // Convert user ID to UUID - implement based on your user structure
    userID, ok := user.(uuid.UUID)
    if !ok {
        return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError, "Invalid user ID format")
    }
    
    return userID, nil
}
// GetFeed handles the feed request with pagination and user-specific content
func (m *Model) GetFeed(c echo.Context) error {
    // Get user ID from JWT token
    userID, err := getUserIDFromToken(c)
    if err != nil {
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "Invalid or missing authentication",
        })
    }

    // Get pagination parameters
    page, err := strconv.Atoi(c.QueryParam("page"))
    if err != nil || page < 1 {
        page = 1
    }
    
    const itemsPerPage = 10
    offset := (page - 1) * itemsPerPage

    // Get photos for user's feed
    photos, err := m.store.GetUserFeed(userID, itemsPerPage, offset)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to load feed",
        })
    }

    // Transform photos for response
    var response []map[string]interface{}
    for _, photo := range photos {
        photoData := map[string]interface{}{
            "photo_id":       photo.PhotoID,
            "filename":       photo.Filename,
            "date":          photo.Date,
            "location":      photo.Location,
            "caption":       photo.Caption,
            "user_id":       photo.UserID,
            "username":      photo.Username,
            "likes_count":   photo.LikesCount,
            "comments_count": photo.CommentsCount,
            "image_url":     "/serveimage/" + photo.Filename,
        }
        response = append(response, photoData)
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "photos": response,
        "page":   page,
        "hasMore": len(photos) == itemsPerPage,
    })
}
// GetPhotoComments handles the request for photo comments
func (m *Model) GetPhotoComments(c echo.Context) error {
    photoID, err := uuid.Parse(c.Param("photoId"))
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid photo ID",
        })
    }

    comments, err := m.store.GetPhotoComments(photoID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to load comments",
        })
    }

    return c.JSON(http.StatusOK, comments)
}
// Modified GetPaginatedPhotos to include user information
func (m *Model) GetPaginatedPhotos(itemsPerPage, offset int) ([]PhotoWithUser, error) {
    return m.store.GetPaginatedPhotos(itemsPerPage, offset)
}

