package user
// Model struct expects a pointer to UserStore
type Model struct {
    userstore *UserStore
}

// NewModel function accepts a pointer to UserStore
func NewModel(userStore *UserStore) *Model {
    return &Model{userstore: userStore}
}