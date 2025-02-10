package photo
import(
	"time"
	"github.com/google/uuid"
)
type PhotoFeedItem struct {
    PhotoID   uuid.UUID `json:"photo_id"`
    Data      []byte    `json:"data"`
    Filename   string   `json:"filename"`
    Username  string    `json:"username"`
    Date      time.Time `json:"date"`
    Location  string    `json:"location"`
    
}
type PhotoWithUser struct {
    PhotoID      uuid.UUID `json:"photo_id"`
    Filename     string    `json:"filename"`
    Data         []byte    `json:"-"`
    Date         time.Time `json:"date"`
    Location     string    `json:"location"`
    Caption      string    `json:"caption"`
    UserID       uuid.UUID `json:"user_id"`
    Username     string    `json:"username"`
    LikesCount   int       `json:"likes_count"`
    CommentsCount int      `json:"comments_count"`
}

type Comment struct {
    CommentID  uuid.UUID `json:"comment_id"`
    Content    string    `json:"content"`
    CreatedAt  time.Time `json:"created_at"`
    UserID     uuid.UUID `json:"user_id"`
    Username   string    `json:"username"`
}