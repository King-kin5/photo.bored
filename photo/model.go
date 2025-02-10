package photo


type Model struct {
	store *Photostore
	rateLimiter *RateLimiter
	wsManager   *WebSocketManager
}

func NewModel(store *Photostore, wsManager *WebSocketManager) *Model {
    return &Model{
        store:       store,
        wsManager:   wsManager,
        rateLimiter: NewRateLimiter(),
    }
}