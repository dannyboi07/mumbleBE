package db

import (
	"errors"
	usp "mumble-user-service/user_service_proto"

	"github.com/jackc/pgconn"
)

func InsertUser(userDetails *usp.RegisterReq, hashedPwd string) error {
	var (
		commandTag pgconn.CommandTag
		err        error
	)
	if userDetails.GetProfilePic() == "" {
		commandTag, err = db.Exec(dbContext, "INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)", userDetails.GetName(), userDetails.GetEmail(), hashedPwd)
	} else if userDetails.GetProfilePic() != "" {
		commandTag, err = db.Exec(dbContext, "INSERT INTO users (name, email, password_hash, profile_pic) VALUES ($1, $2, $3, $4)", userDetails.GetName(), userDetails.GetEmail(), hashedPwd, userDetails.GetProfilePic())
	}

	if err == nil && commandTag.RowsAffected() != 1 {
		return errors.New("Failed to create a new user, try again!")
	}
	return err
}

func InsertContact(userId1, userId2 int64) error {
	commandTag, err := db.Exec(dbContext, `INSERT INTO user_contact (user_1_id, user_2_id) VALUES ($1, $2)
											WHERE NOT EXISTS (SELECT contact_id FROM user_contact WHERE user_1_id = $1 AND user_2_id = $2
												OR user_1_id = $2 AND user_2_id = $1)`, userId1, userId2)

	if err == nil && commandTag.RowsAffected() != 1 {
		return errors.New("Contact already exists")
	}

	return err
}
