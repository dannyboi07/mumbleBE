package db

import (
	"mumble-user-service/types"
	usp "mumble-user-service/user_service_proto"
)

func SelectUserById(userId int64) (types.User, error) {
	var user types.User
	row := db.QueryRow(dbContext, "SELECT user_id, name, email, profile_pic, password_hash FROM users WHERE user_id = $1", userId)
	err := row.Scan(&user.UserId, &user.Name, &user.Email, &user.Profile_pic, &user.PasswordHash)

	return user, err
}

func SelectUserByEmail(userEmail string) (types.User, error) {
	var user types.User
	row := db.QueryRow(dbContext, "SELECT user_id, name, email, profile_pic, password_hash FROM users WHERE email = $1", userEmail)
	err := row.Scan(&user.UserId, &user.Name, &user.Email, &user.Profile_pic, &user.PasswordHash)

	return user, err
}

func SelectUserByEmailSearch(userEmail string) (types.UserSearch, error) {
	var user types.UserSearch
	row := db.QueryRow(dbContext, "SELECT user_id, name, profile_pic, FROM users WHERE email = $1", userEmail)
	err := row.Scan(&user.UserId, &user.Name, &user.Profile_pic)

	return user, err
}

func SelectContacts(userId int64) (*usp.GetContactsResp, error) {
	// var contacts *usp.GetContactsResp
	contacts := &usp.GetContactsResp{}
	rows, err := db.Query(dbContext, `SELECT users.user_id, users.name, users.profile_pic FROM users JOIN user_contact
										ON users.user_id = user_contact.user_1_id AND user_contact.user_2_id = $1
										OR users.user_id = user_contact.user_2_id AND user_contact.user_1_id = $1`, userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		contact := &usp.GetContactsResp_Contact{}

		err := rows.Scan(&contact.UserId, &contact.Name, &contact.ProfilePic)
		if err != nil {
			return nil, err
		}

		contacts.Contacts = append(contacts.GetContacts(), contact)
	}

	return contacts, nil
}
