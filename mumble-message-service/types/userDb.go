package types

type UserDb struct {
	User
	Password string `json:"-"`
}
