package photo

import (
    "github.com/gorilla/websocket"
    "github.com/labstack/echo/v4"
    "net/http"
    
)

type WebSocketManager struct {
    // Maintain a list of connected WebSocket clients
    Clients   map[*websocket.Conn]bool
    Broadcast chan Photo
    Upgrader  websocket.Upgrader
}

func NewWebSocketManager() *WebSocketManager {
    return &WebSocketManager{
        Clients:   make(map[*websocket.Conn]bool),
        Broadcast: make(chan Photo),
        Upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                return true // In production, implement proper origin checking
            },
        },
    }
}

// HandleWebSocket handles WebSocket connections for real-time feed updates
func (wsm *WebSocketManager) HandleWebSocket(c echo.Context) error {
    ws, err := wsm.Upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        return err
    }
    defer ws.Close()

    // Register new client
    wsm.Clients[ws] = true
    defer delete(wsm.Clients, ws)

    // Keep connection alive and handle messages
    for {
        // Read message (not used in this implementation but could be used for client actions)
        _, _, err := ws.ReadMessage()
        if err != nil {
            break
        }
    }

    return nil
}

// BroadcastNewPhoto sends new photo updates to all connected clients
func (wsm *WebSocketManager) BroadcastNewPhoto(photo Photo) {
    for client := range wsm.Clients {
        err := client.WriteJSON(photo)
        if err != nil {
            client.Close()
            delete(wsm.Clients, client)
        }
    }
}

// StartBroadcaster starts the WebSocket broadcaster
func (wsm *WebSocketManager) StartBroadcaster() {
    go func() {
        for {
            newPhoto := <-wsm.Broadcast
            wsm.BroadcastNewPhoto(newPhoto)
        }
    }()
}
