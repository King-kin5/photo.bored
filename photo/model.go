package photo

type Model struct {
	store *Photostore
}

func NewModel(Store *Photostore) *Model {
	return &Model{store: Store}
}
