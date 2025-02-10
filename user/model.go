package user
import"app/security"
// Model struct expects a pointer to UserStore
type ModelHandler struct {
	userstore    UserStore
	tokenManager *security.TokenManager
}

func NewModelHandler(userstore UserStore, config *security.Config) *ModelHandler {
	return &ModelHandler{
		userstore:    userstore,
		tokenManager: security.NewTokenManager(config),
	}
}