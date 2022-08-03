package types

type User struct {
	UserId      int64  `json:"user_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Profile_pic string `json:"profile_pic"`
}

type UserProfileLink struct {
	ProfileImgLink string `json:"profile_pic"`
}

type RegisterUser struct {
	Name        *string `json:"name"`
	Email       *string `json:"email"`
	Password    *string `json:"-"`
	Profile_Pic *string `json:"profile_pic"`
}

type LoginUser struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

type UserChangePwInput struct {
	OldPassword *string `json:"password"`
	NewPassword *string `json:"new_pw"`
}

type SearchUserInput struct {
	Email *string `json:"email"`
}

type UserLastSeen struct {
	UserLastSeenTime string `json:"last_seen"`
}
