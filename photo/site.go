package photo

import (
	"app/pkg/utils"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// Constants for file Validation and imageProcessing
const (
	maxFileSize    = 10 << 20 // 10MB
	MaxUploadFiles = 10
	thumbSize      = 800 // maximum dimension for thumbnails
)

// ValidImageTypes maps allowed MIME types to their file extensions
var ValidImageTypes = map[string]string{
	"image/jpeg": ".jpg",
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

// HandleUpload handles file upload with proper error handling and WebSocket broadcasting
func (m *Model) HandleUpload(c echo.Context) error {
	utils.Logger.Info("Upload handler called")

	// Get authenticated user ID from context (set by authentication middleware)
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		utils.Logger.Error("user_id not found in context or invalid type")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not authenticated",
			"code":  "UNAUTHORIZED",
		})
	}

	utils.Logger.Infof("Upload request from user: %s", userID)

	// Apply rate limiting
	limiter := m.rateLimiter.GetLimiter(c.RealIP())
	if !limiter.Allow() {
		utils.Logger.Warnf("Rate limit exceeded for IP: %s", c.RealIP())
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": "Rate limit exceeded",
			"code":  "RATE_LIMIT_EXCEEDED",
		})
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		utils.Logger.Errorf("Error parsing multipart form: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Error parsing form data",
			"code":  "FORM_PARSE_ERROR",
		})
	}

	// Try multiple possible field names for file uploads
	var files []*multipart.FileHeader
	if form.File["images[]"] != nil {
		files = form.File["images[]"]
		utils.Logger.Info("Found files in 'images[]' field")
	} else if form.File["images"] != nil {
		files = form.File["images"]
		utils.Logger.Info("Found files in 'images' field")
	} else if form.File["file"] != nil {
		files = form.File["file"]
		utils.Logger.Info("Found files in 'file' field")
	}

	if len(files) == 0 {
		utils.Logger.Error("No files found in upload request")
		utils.Logger.Infof("Available form fields: %+v", form.File)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No files uploaded",
			"code":  "NO_FILES",
		})
	}

	if len(files) > MaxUploadFiles {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Maximum %d files allowed per upload", MaxUploadFiles),
			"code":  "TOO_MANY_FILES",
		})
	}

	// Get optional caption from form
	caption := ""
	if form.Value["caption"] != nil && len(form.Value["caption"]) > 0 {
		caption = strings.TrimSpace(form.Value["caption"][0])
		utils.Logger.Infof("Caption found: %s", caption)
	}

	var uploadedImages []string
	var successCount int
	var errors []string

	for i, fileHeader := range files {
		utils.Logger.Infof("Processing file %d/%d: %s (size: %d bytes)", i+1, len(files), fileHeader.Filename, fileHeader.Size)

		if err := validateFile(fileHeader); err != nil {
			utils.Logger.Errorf("File validation failed for %s: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("File %s: %v", fileHeader.Filename, err))
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			utils.Logger.Errorf("Error opening file %s: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("Could not open file %s", fileHeader.Filename))
			continue
		}
		defer file.Close()

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, file); err != nil {
			utils.Logger.Errorf("Failed to read file %s: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("Could not read file %s", fileHeader.Filename))
			continue
		}

		mimeType := http.DetectContentType(buf.Bytes())
		utils.Logger.Infof("Detected MIME type: %s for file: %s", mimeType, fileHeader.Filename)

		// Validate MIME type
		if !strings.HasPrefix(mimeType, "image/") {
			utils.Logger.Errorf("Invalid MIME type %s for file %s", mimeType, fileHeader.Filename)
			errors = append(errors, fmt.Sprintf("File %s is not a valid image", fileHeader.Filename))
			continue
		}

		// Decode the image for optimization
		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			utils.Logger.Errorf("Failed to decode image %s: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("Could not decode image %s", fileHeader.Filename))
			continue
		}

		// Optimize the image
		optimizedData, err := optimizeImage(img, mimeType)
		if err != nil {
			utils.Logger.Errorf("Failed to optimize image %s: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("Could not optimize image %s", fileHeader.Filename))
			continue
		}

		// Generate unique filename to avoid conflicts
		uniqueFilename := fmt.Sprintf("%s_%s", uuid.New().String(), fileHeader.Filename)

		photo := &Photo{
			PhotoID:  uuid.New(),
			Filename: uniqueFilename,
			Data:     optimizedData,
			Date:     time.Now(),
			Location: "Unknown Location",
			MIMEType: mimeType,
			UserID:   userID,
			Caption:  caption,
		}

		utils.Logger.Infof("Attempting to save image to database: %s", photo.Filename)

		if err := m.store.Addimage(photo); err != nil {
			utils.Logger.Errorf("Failed to save image %s to database: %v", fileHeader.Filename, err)
			errors = append(errors, fmt.Sprintf("Could not save image %s", fileHeader.Filename))
			continue
		}

		utils.Logger.Infof("Successfully saved image: %s", photo.Filename)

		// Get the photo with user info for broadcasting
		photoWithUser, err := m.getPhotoWithUserInfo(photo)
		if err != nil {
			utils.Logger.Warnf("Could not get user info for WebSocket broadcast: %v", err)
		} else {
			// Broadcast the new photo to all connected clients for real-time updates
			select {
			case m.wsManager.Broadcast <- *photoWithUser:
				utils.Logger.Info("New photo broadcasted to WebSocket clients for real-time feed updates")
			default:
				utils.Logger.Warn("WebSocket broadcast channel full, real-time update skipped")
			}
		}

		imageURL := fmt.Sprintf("/serveimage/%s", photo.Filename)
		uploadedImages = append(uploadedImages, imageURL)
		successCount++
	}

	if successCount == 0 {
		utils.Logger.Error("No files were successfully uploaded")
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "No files were successfully uploaded",
			"code":    "UPLOAD_FAILED",
			"details": errors,
		})
	}

	utils.Logger.Infof("Upload completed successfully: %d/%d files", successCount, len(files))

	response := map[string]interface{}{
		"message": fmt.Sprintf("Successfully uploaded %d images", successCount),
		"images":  uploadedImages,
		"success": true,
	}

	if len(errors) > 0 {
		response["warnings"] = errors
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to get photo with user info for WebSocket broadcasting
func (m *Model) getPhotoWithUserInfo(photo *Photo) (*PhotoWithUser, error) {
	// This would typically require a database query to get username
	// For now, create a basic PhotoWithUser structure
	photoWithUser := &PhotoWithUser{
		PhotoID:       photo.PhotoID,
		Filename:      photo.Filename,
		Date:          photo.Date,
		Location:      photo.Location,
		Caption:       photo.Caption,
		UserID:        photo.UserID,
		Username:      "User", // TODO: Get actual username from database
		LikesCount:    0,
		CommentsCount: 0,
	}
	
	return photoWithUser, nil
}

func (m *Model) HandleServeImage(c echo.Context) error {
	filename := c.Param("filename")

	// ADDED: Better validation for filename parameter
	if filename == "" || filename == "undefined" || filename == "null" {
		utils.Logger.Warnf("Invalid filename requested: '%s' from IP: %s", filename, c.RealIP())
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid filename",
			"code":  "INVALID_FILENAME",
		})
	}

	photo, err := m.store.GetImageByFilename(filename)
	if err != nil {
		utils.Logger.Errorf("Image not found: %s, error: %v", filename, err)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Image not found",
			"code":  "NOT_FOUND",
		})
	}

	if photo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Image not found",
			"code":  "NOT_FOUND",
		})
	}

	// Set proper cache headers
	c.Response().Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	c.Response().Header().Set("ETag", fmt.Sprintf("\"%s\"", photo.PhotoID.String()))

	return c.Blob(http.StatusOK, photo.MIMEType, photo.Data)
}

// HandleDeleteImage with user authorization
func (m *Model) HandleDeleteImage(c echo.Context) error {
	filename := c.Param("filename")

	// Get authenticated user ID
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not authenticated",
		})
	}

	// Check if photo belongs to user before deleting
	photo, err := m.store.GetImageByFilename(filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Could not retrieve image",
		})
	}

	if photo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Image not found",
		})
	}

	if photo.UserID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You can only delete your own images",
		})
	}

	deletedPhoto, err := m.store.DeleteImageByFilename(filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Could not delete image",
		})
	}

	if deletedPhoto == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Image not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Image %s deleted successfully", deletedPhoto.Filename),
	})
}

// HandleCaption with user authorization
func (m *Model) HandleCaption(c echo.Context) error {
	filename := c.Param("filename")
	newCaption := c.FormValue("caption")

	if newCaption == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Caption cannot be empty",
		})
	}

	// Get authenticated user ID
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not authenticated",
		})
	}

	// Check if photo belongs to user before updating caption
	photo, err := m.store.GetImageByFilename(filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Could not retrieve image",
		})
	}

	if photo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Image not found",
		})
	}

	if photo.UserID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You can only update captions for your own images",
		})
	}

	err = m.store.UpdateCaptionByFilename(filename, newCaption)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update caption",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Caption updated successfully",
	})
}

// HandleCreateAlbum with user association
func (m *Model) HandleCreateAlbum(c echo.Context) error {
	name := c.FormValue("Name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Album name is required",
		})
	}

	// Get authenticated user ID
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not authenticated",
		})
	}

	album := &Album{
		AlbumID:   uuid.New(),
		Name:      name,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	err := m.store.Createalbum(album)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create album",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "Album created successfully",
		"album_id": album.AlbumID,
		"name":     album.Name,
	})
}

func (m *Model) HandleAddPhotoToAlbum(c echo.Context) error {
	albumID, err := uuid.Parse(c.FormValue("AlbumID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid AlbumID",
		})
	}

	photoID, err := uuid.Parse(c.FormValue("PhotoID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid PhotoID",
		})
	}

	err = m.store.AddPhotoToAlbum(albumID, photoID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add photo to album",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Photo added to album successfully",
	})
}

func (m *Model) HandleGetAlbumWithPhotos(c echo.Context) error {
	albumID, err := uuid.Parse(c.QueryParam("AlbumID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid AlbumID",
		})
	}

	album, err := m.store.GetAlbumWithPhotos(albumID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve album and photos",
		})
	}

	// Remove binary data and set location for each photo
	for i := range album.Photos {
		album.Photos[i].Data = nil
		album.Photos[i].Location = fmt.Sprintf("/serveimage/%s", album.Photos[i].Filename)
	}

	return c.JSON(http.StatusOK, album)
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

// FIXED: Infinite scroll feed handler optimized for Twitter/X-like experience
func (m *Model) GetInfiniteFeed(c echo.Context) error {
	var req EnhancedFeedRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request parameters",
		})
	}

	// Set defaults for infinite scroll
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	if req.FeedType == "" {
		req.FeedType = "public"
	}

	// ENHANCED: Better authentication check and handling
	var userID *uuid.UUID
	if uid, ok := c.Get("user_id").(uuid.UUID); ok {
		userID = &uid
		utils.Logger.Infof("Authenticated user %s requesting %s feed", uid.String(), req.FeedType)
	} else {
		utils.Logger.Infof("Unauthenticated user requesting %s feed", req.FeedType)
		
		// Force unauthenticated users to public feed only
		if req.FeedType != "public" {
			utils.Logger.Warnf("Unauthenticated user tried to access %s feed, redirecting to public", req.FeedType)
			req.FeedType = "public"
		}
	}

	// Parse cursor for infinite scroll pagination
	var cursor *time.Time
	if req.Cursor != "" {
		if cursorTime, parseErr := time.Parse(time.RFC3339, req.Cursor); parseErr == nil {
			cursor = &cursorTime
		}
	}

	var photos []PhotoWithUser
	var nextCursor *time.Time
	var err error

	// Route to appropriate feed based on type - FIXED routing logic
	switch FeedType(req.FeedType) {
	case FeedTypePublic:
		// Public/Discover feed - all photos
		photos, nextCursor, err = m.store.GetInfiniteScrollPhotos(req.Limit, cursor)
		utils.Logger.Infof("Loading public feed with %d photos", len(photos))
		
	case FeedTypePersonal:
		// Personal feed - user's own photos only
		if userID == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authentication required for personal feed",
				"code":  "AUTH_REQUIRED",
			})
		}
		photos, nextCursor, err = m.store.GetUserPersonalFeedInfinite(*userID, req.Limit, cursor)
		utils.Logger.Infof("Loading personal feed for user %s with %d photos", userID.String(), len(photos))
		
	case FeedTypeCombined:
		// Combined feed - user's photos + following photos
		if userID == nil {
			utils.Logger.Info("Unauthenticated user accessing combined feed, falling back to public")
			photos, nextCursor, err = m.store.GetInfiniteScrollPhotos(req.Limit, cursor)
		} else {
			photos, nextCursor, err = m.store.GetUserFeedInfinite(*userID, req.Limit, cursor)
		}
		utils.Logger.Infof("Loading combined feed with %d photos", len(photos))
		
	case FeedTypeFollowing:
		// Following feed - photos from followed users only
		if userID == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authentication required for following feed",
				"code":  "AUTH_REQUIRED",
			})
		}
		photos, nextCursor, err = m.store.GetFollowingFeedInfinite(*userID, req.Limit, cursor)
		utils.Logger.Infof("Loading following feed for user %s with %d photos", userID.String(), len(photos))
		
	default:
		// Fallback to public feed
		photos, nextCursor, err = m.store.GetInfiniteScrollPhotos(req.Limit, cursor)
		utils.Logger.Warnf("Unknown feed type %s, falling back to public feed", req.FeedType)
	}

	if err != nil {
		utils.Logger.Errorf("Error fetching infinite scroll feed (%s): %v", req.FeedType, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load feed",
			"code":  "FEED_ERROR",
		})
	}

	// ADDED: Additional filtering to ensure no photos with empty filenames
	var validPhotos []PhotoWithUser
	for _, photo := range photos {
		if photo.Filename != "" && photo.Filename != "undefined" && photo.Filename != "null" {
			validPhotos = append(validPhotos, photo)
		} else {
			utils.Logger.Warnf("Filtered out photo %s with invalid filename: '%s'", photo.PhotoID.String(), photo.Filename)
		}
	}
	photos = validPhotos

	// Determine if there are more photos
	hasMore := nextCursor != nil && len(photos) > 0

	// Convert nextCursor to string for frontend
	var nextCursorStr *string
	if nextCursor != nil {
		cursorStr := nextCursor.Format(time.RFC3339)
		nextCursorStr = &cursorStr
	}

	// Response format expected by infinite scroll frontend
	response := map[string]interface{}{
		"photos":        transformPhotosForInfiniteScroll(photos),
		"has_more":      hasMore,
		"next_cursor":   nextCursorStr,
		"feed_type":     req.FeedType,
		"count":         len(photos),
		"timestamp":     time.Now().Unix(),
		"authenticated": userID != nil,
	}

	utils.Logger.Infof("Returning %d photos for %s feed (hasMore: %v)", len(photos), req.FeedType, hasMore)
	return c.JSON(http.StatusOK, response)
}

// Transform photos for infinite scroll response format (matches feed.js expectations)

func transformPhotosForInfiniteScroll(photos []PhotoWithUser) []map[string]interface{} {

	var response []map[string]interface{}

	for _, photo := range photos {

		filename := photo.Filename

		if filename == "" || filename == "undefined" || filename == "null" {

			utils.Logger.Warnf("Skipping photo %s with invalid filename: '%s'", photo.PhotoID.String(), filename)

			continue

		}

		

		photoData := map[string]interface{}{

			"photo_id":       photo.PhotoID,

			"filename":       filename,

			"date":          photo.Date.Format(time.RFC3339),

			"created_at":    photo.Date.Format(time.RFC3339),

			"location":      photo.Location,

			"caption":       photo.Caption,

			"user_id":       photo.UserID,

			"username":      photo.Username,

			"likes_count":   photo.LikesCount,

			"comments_count": photo.CommentsCount,

			"image_url":     fmt.Sprintf("/serveimage/%s", filename),

			"timestamp":     photo.Date.Unix(),

		}

		response = append(response, photoData)

	}

	return response

}



// HandleLikePhoto handles liking a photo

func (m *Model) HandleLikePhoto(c echo.Context) error {

	photoID, err := uuid.Parse(c.Param("photo_id"))

	if err != nil {

		return c.JSON(http.StatusBadRequest, map[string]string{

			"error": "Invalid photo ID",

		})

	}



	userID, ok := c.Get("user_id").(uuid.UUID)

	if !ok {

		return c.JSON(http.StatusUnauthorized, map[string]string{

			"error": "User not authenticated",

		})

	}



	err = m.store.AddLike(photoID, userID)

	if err != nil {

		return c.JSON(http.StatusInternalServerError, map[string]string{

			"error": "Failed to like photo",

		})

	}



	return c.JSON(http.StatusOK, map[string]string{

		"message": "Photo liked successfully",

	})

}



// HandleUnlikePhoto handles unliking a photo

func (m *Model) HandleUnlikePhoto(c echo.Context) error {

	photoID, err := uuid.Parse(c.Param("photo_id"))

	if err != nil {

		return c.JSON(http.StatusBadRequest, map[string]string{

			"error": "Invalid photo ID",

		})

	}



	userID, ok := c.Get("user_id").(uuid.UUID)

	if !ok {

		return c.JSON(http.StatusUnauthorized, map[string]string{

			"error": "User not authenticated",

		})

	}



	err = m.store.RemoveLike(photoID, userID)

	if err != nil {

		return c.JSON(http.StatusInternalServerError, map[string]string{

			"error": "Failed to unlike photo",

		})

	}



	return c.JSON(http.StatusOK, map[string]string{

		"message": "Photo unliked successfully",

	})

}



// HandleGetLikesCount handles getting the number of likes for a photo

func (m *Model) HandleGetLikesCount(c echo.Context) error {

	photoID, err := uuid.Parse(c.Param("photo_id"))

	if err != nil {

		return c.JSON(http.StatusBadRequest, map[string]string{

			"error": "Invalid photo ID",

		})

	}



	count, err := m.store.GetLikesCount(photoID)

	if err != nil {

		return c.JSON(http.StatusInternalServerError, map[string]string{

			"error": "Failed to get likes count",

		})

	}



	return c.JSON(http.StatusOK, map[string]interface{}{

		"likes_count": count,

	})

}



// HandleGetLikedPhotos handles getting the photos that the current user has liked

func (m *Model) HandleGetLikedPhotos(c echo.Context) error {

	userID, ok := c.Get("user_id").(uuid.UUID)

	if !ok {

		return c.JSON(http.StatusUnauthorized, map[string]string{

			"error": "User not authenticated",

		})

	}



	photos, err := m.store.GetLikedPhotos(userID)

	if err != nil {

		return c.JSON(http.StatusInternalServerError, map[string]string{

			"error": "Failed to get liked photos",

		})

	}



	return c.JSON(http.StatusOK, photos)

}


