package user

type Userstore interface {
	CreateUser(user User) error
	GetUserByEmail(email string) (User, error)
	GetUserByID(userID string) (User, error)
	GetUserByName(name string) (User, error)
	UpdateUserInfoByID(userID string, user User) error
	CheckDBConnection() error
}
