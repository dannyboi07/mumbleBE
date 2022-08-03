package db

func UserExistsByEmail(userEmail string) (bool, error) {
	var exists bool
	row := db.QueryRow(dbContext, "SELECT EXISTS(SELECT user_id FROM users WHERE email = $1)", userEmail)
	err := row.Scan(&exists)

	return exists, err
}

func UserExistsById(userId int64) (bool, error) {
	var exists bool
	row := db.QueryRow(dbContext, "SELECT EXISTS(SELECT user_id FROM users WHERE user_id = $1)", userId)
	err := row.Scan(&exists)

	return exists, err
}

func ContactExists(userId int64, contactId int64) (bool, error) {
	var exists bool
	row := db.QueryRow(dbContext, "SELECT EXISTS(SELECT contact_id FROM user_contact WHERE user_1_id = $1, OR user_2_id = $2)", userId, contactId)
	err := row.Scan(&exists)

	return exists, err
}
