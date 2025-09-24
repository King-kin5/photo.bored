package photo

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"app/pkg/utils"
)

// WebSocketManager handles real-time feed updates like Twitter/X
type WebSocketManager struct {
	// Connected clients
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex

	// Channel for broadcasting new photos to all connected clients
	Broadcast chan PhotoWithUser
	register chan *websocket.Conn
	// Channel for unregistering clients
	unregister chan *websocket.Conn
	// Upgrader for WebSocket connections
	upgrader websocket.Upgrader
}

type WSMessageType string
const (
	WSNewPhoto     WSMessageType = "new_photo"
	WSPhotoUpdate  WSMessageType = "photo_update"
	WSPhotoDelete  WSMessageType = "photo_delete"
	WSUserOnline   WSMessageType = "user_online"
	WSUserOffline  WSMessageType = "user_offline"
	WSFeedRefresh  WSMessageType = "feed_refresh"
)

// WebSocket message structure
type WSMessage struct {
	Type      WSMessageType   `json:"type"`
	Data      interface{}     `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	MessageID string          `json:"message_id"`
}
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*websocket.Conn]bool),
		Broadcast:  make(chan PhotoWithUser, 256), // Buffered channel for high throughput
		register:   make(chan *websocket.Conn, 64),
		unregister: make(chan *websocket.Conn, 64),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin in development
				// In production, you should validate the origin
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// StartBroadcaster starts the main WebSocket broadcaster goroutine
func (manager *WebSocketManager) StartBroadcaster() {
	go manager.handleConnections()
	utils.Logger.Info("WebSocket broadcaster started for real-time feed updates")
}

// handleConnections manages WebSocket connections and broadcasts
func (manager *WebSocketManager) handleConnections() {
	for {
		select {
		case conn := <-manager.register:
			manager.clientsMux.Lock()
			manager.clients[conn] = true
			manager.clientsMux.Unlock()
			utils.Logger.Infof("New WebSocket client connected. Total clients: %d", len(manager.clients))
			// Send welcome message
			welcomeMsg := WSMessage{
				Type:      WSUserOnline,
				Data:      map[string]string{"status": "connected", "message": "Real-time feed connected"},
				Timestamp: time.Now(),
				MessageID: generateMessageID(),
			}
			manager.sendToClient(conn, welcomeMsg)

		case conn := <-manager.unregister:
			manager.clientsMux.Lock()
			if _, ok := manager.clients[conn]; ok {
				delete(manager.clients, conn)
				conn.Close()
			}
			manager.clientsMux.Unlock()
			utils.Logger.Infof("WebSocket client disconnected. Total clients: %d", len(manager.clients))

		case photoWithUser := <-manager.Broadcast:
			// Broadcast new photo to all connected clients (like Twitter's real-time timeline)
			message := WSMessage{
				Type: WSNewPhoto,
				Data: map[string]interface{}{
					"photo_id":       photoWithUser.PhotoID,
					"filename":       photoWithUser.Filename,
					"date":          photoWithUser.Date.Format(time.RFC3339),
					"created_at":    photoWithUser.Date.Format(time.RFC3339),
					"location":      photoWithUser.Location,
					"caption":       photoWithUser.Caption,
					"user_id":       photoWithUser.UserID,
					"username":      photoWithUser.Username,
					"likes_count":   photoWithUser.LikesCount,
					"comments_count": photoWithUser.CommentsCount,
					"image_url":     "/serveimage/" + photoWithUser.Filename,
					"timestamp":     photoWithUser.Date.Unix(),
				},
				Timestamp: time.Now(),
				MessageID: generateMessageID(),
			}

			manager.broadcastToAllClients(message)
		}
	}
}

// HandleWebSocket handles new WebSocket connections
func (manager *WebSocketManager) HandleWebSocket(c echo.Context) error {
	conn, err := manager.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		utils.Logger.Errorf("WebSocket upgrade failed: %v", err)
		return err
	}

	// Register the new client
	manager.register <- conn

	// Handle client messages and disconnection
	go manager.handleClient(conn)

	return nil
}

// handleClient manages individual WebSocket client connections
func (manager *WebSocketManager) handleClient(conn *websocket.Conn) {
	defer func() {
		manager.unregister <- conn
		conn.Close()
	}()

	// Set read deadline for ping/pong to detect disconnected clients
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Listen for client messages (mainly for ping/pong)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				utils.Logger.Errorf("WebSocket client error: %v", err)
			}
			break
		}
	}
}

// sendToClient sends a message to a specific WebSocket client
func (manager *WebSocketManager) sendToClient(conn *websocket.Conn, message WSMessage) {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteJSON(message); err != nil {
		utils.Logger.Errorf("Error sending WebSocket message: %v", err)
		manager.unregister <- conn
	}
}

// broadcastToAllClients sends a message to all connected WebSocket clients
func (manager *WebSocketManager) broadcastToAllClients(message WSMessage) {
	manager.clientsMux.RLock()
	clientCount := len(manager.clients)
	manager.clientsMux.RUnlock()

	if clientCount == 0 {
		return // No clients to broadcast to
	}

	utils.Logger.Infof("Broadcasting new photo to %d connected clients", clientCount)

	manager.clientsMux.RLock()
	defer manager.clientsMux.RUnlock()

	for conn := range manager.clients {
		select {
		case <-time.After(1 * time.Second): // Timeout for slow clients
			utils.Logger.Warn("WebSocket broadcast timeout, removing slow client")
			delete(manager.clients, conn)
			conn.Close()
		default:
			manager.sendToClient(conn, message)
		}
	}
}

// BroadcastPhotoUpdate broadcasts photo updates (likes, comments, etc.)
func (manager *WebSocketManager) BroadcastPhotoUpdate(photoID string, updateType string, data interface{}) {
	message := WSMessage{
		Type: WSPhotoUpdate,
		Data: map[string]interface{}{
			"photo_id":    photoID,
			"update_type": updateType,
			"data":        data,
		},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	go func() {
		manager.broadcastToAllClients(message)
	}()
}

// BroadcastPhotoDelete broadcasts photo deletion
func (manager *WebSocketManager) BroadcastPhotoDelete(photoID string) {
	message := WSMessage{
		Type: WSPhotoDelete,
		Data: map[string]interface{}{
			"photo_id": photoID,
		},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	go func() {
		manager.broadcastToAllClients(message)
	}()
}

// RequestFeedRefresh broadcasts a feed refresh request to all clients
func (manager *WebSocketManager) RequestFeedRefresh() {
	message := WSMessage{
		Type: WSFeedRefresh,
		Data: map[string]interface{}{
			"message": "Feed refresh requested",
		},
		Timestamp: time.Now(),
		MessageID: generateMessageID(),
	}

	go func() {
		manager.broadcastToAllClients(message)
	}()
}

// GetConnectedClientsCount returns the number of connected WebSocket clients
func (manager *WebSocketManager) GetConnectedClientsCount() int {
	manager.clientsMux.RLock()
	defer manager.clientsMux.RUnlock()
	return len(manager.clients)
}

// generateMessageID creates a unique message ID for WebSocket messages
func generateMessageID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}