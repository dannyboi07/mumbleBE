package db

import "errors"

func UpdateUserPwd(userId int64, hashPwd string) error {
	commandTag, err := db.Exec(dbContext, "UPDATE users SET password_hash = $1 WHERE user_id = $2", hashPwd, userId)

	if err == nil && commandTag.RowsAffected() != 1 {
		return errors.New("Failed to update user's pwd")
	}

	return err
}

func UpdateUserDp(userId int64, profilePicUrl string) error {
	commandTag, err := db.Exec(dbContext, "UPDATE users SET profile_pic = $1 WHERE user_id = $2", profilePicUrl, userId)

	if err == nil && commandTag.RowsAffected() != 1 {
		return errors.New("Failed to update user's profile pic")
	}

	return err
}
