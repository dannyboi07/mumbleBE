package types

type User struct {
	UserId int64
	UserWithoutId
}

type UserWithoutId struct {
	Name         string
	Email        string
	PasswordHash string `json:"-"`
	Profile_pic  string
}

type UserForToken struct {
	UserId int64
	Email  string
}

type UserSearch struct {
	UserId      int64
	Name        string
	Profile_pic string
}
