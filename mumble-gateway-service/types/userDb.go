package types

type UserDb struct {
	User
	Password string `json:"-"`
}

type UserForToken struct {
	UserId int64
	Email  string
}
