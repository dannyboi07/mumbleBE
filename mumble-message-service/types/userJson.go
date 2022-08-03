package types

type User struct {
	UserId int64  `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	UserProfileLink
}

type UserWithPw struct {
	User
	Password string `json:"-"`
}

type UserProfileLink struct {
	ProfileImgLink string `json:"profile_pic"`
}

type RegisterUser struct {
	Name        *string `json:"name"`
	Email       *string `json:"email"`
	Password    *string `json:"-"`
	Profile_Pic *string `json:"profile_pic"`
	// *UserProfileLink
}

type UserChangePwInput struct {
	OldPassword *string `json:"password"`
	NewPassword *string `json:"new_pw"`
}
