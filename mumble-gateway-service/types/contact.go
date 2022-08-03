package types

type Contact struct {
	UserId int64  `json:"user_id"`
	Name   string `json:"name"`
	// Profile_Pic string `json:"profile_pic"`
	UserProfileLink
}

type ContactSearch struct {
	Contact
	IsFriend bool `json:"is_friend"`
}

type AddContactId struct {
	UserId *int64 `json:"user_id"`
}
