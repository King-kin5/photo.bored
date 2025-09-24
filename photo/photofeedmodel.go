package photo
import(
	"time"
	"github.com/google/uuid"
)
// Photo struct with all necessary fields
type Photo struct {
	PhotoID  uuid.UUID `json:"photo_id" db:"photo_id"`
	Filename string    `json:"filename" db:"filename"`
	Data     []byte    `json:"-" db:"data"`
	Date     time.Time `json:"date" db:"created_at"`
	MIMEType string    `json:"mime_type" db:"mime_type"`
	Location string    `json:"location" db:"location"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	Caption  string    `json:"caption" db:"caption"`
}

// PhotoWithUser includes user information for feed display
type PhotoWithUser struct {
	PhotoID       uuid.UUID `json:"photo_id" db:"photo_id"`
	Filename      string    `json:"filename" db:"filename"`
	Date          time.Time `json:"date" db:"created_at"`
	Location      string    `json:"location" db:"location"`
	Caption       string    `json:"caption" db:"caption"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Username      string    `json:"username" db:"username"`
	LikesCount    int       `json:"likes_count" db:"likes_count"`
	CommentsCount int       `json:"comments_count" db:"comments_count"`
}

// Comment struct for photo comments
type Comment struct {
	CommentID uuid.UUID `json:"comment_id" db:"comment_id"`
	PhotoID   uuid.UUID `json:"photo_id" db:"photo_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Username  string    `json:"username" db:"username"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Album struct - Fixed field names to be consistent
type Album struct {
	AlbumID   uuid.UUID `json:"album_id" db:"album_id"`
	Name      string    `json:"name" db:"name"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Photos    []Photo   `json:"photos"`
}
type FeedType string

const (
    FeedTypePublic     FeedType = "public"     // All public posts
    FeedTypeFollowing  FeedType = "following"  // Posts from followed users
    FeedTypePersonal   FeedType = "personal"   // User's own posts
    FeedTypeCombined   FeedType = "combined"   // Own posts + followed users
)

// EnhancedFeedRequest represents the request for different feed types
type EnhancedFeedRequest struct {
    FeedType   string `query:"feed_type" json:"feed_type"`
    Cursor     string `query:"cursor" json:"cursor"`
    Limit      int    `query:"limit" json:"limit"`
    Personalized bool `query:"personalized" json:"personalized"`
}

// Store interface defines the methods for data persistence
type Store interface {
	Addimage(photo *Photo) error
	GetImageByFilename(filename string) (*Photo, error)
	DeleteImageByFilename(filename string) (*Photo, error)
	UpdateCaptionByFilename(filename, caption string) error
	Createalbum(album *Album) error
	AddPhotoToAlbum(albumID, photoID uuid.UUID) error
	GetAlbumWithPhotos(albumID uuid.UUID) (*Album, error)
	GetPaginatedPhotos(itemsPerPage, offset int) ([]PhotoWithUser, error) // Keep for backward compatibility
	GetPhotoComments(photoID uuid.UUID) ([]Comment, error)
	GetUserFeed(userID uuid.UUID, limit, offset int) ([]PhotoWithUser, error) // Keep for backward compatibility
	
	// New infinite scroll methods
	GetInfiniteScrollPhotos(limit int, cursor *time.Time) ([]PhotoWithUser, *time.Time, error)
	GetUserFeedInfinite(userID uuid.UUID, limit int, cursor *time.Time) ([]PhotoWithUser, *time.Time, error)
}
