package db

import (
	"context"
	"errors"

	msp "mumble-message-service/message-service-proto"
	"mumble-message-service/types"
	"mumble-message-service/utils"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	protoTStamp "google.golang.org/protobuf/types/known/timestamppb"
)

var db *pgxpool.Pool
var dbContext context.Context

func InitDB() error {
	dbContext = context.Background()
	var err error
	db, err = pgxpool.Connect(dbContext, os.Getenv("DB_CONN"))

	return err
}

func CloseDB() {
	db.Close()
}

func SelectMsgs(contactId1, contactId2, offset int64) (*msp.Messages, error) {
	var messages msp.Messages = msp.Messages{}
	rows, err := db.Query(dbContext, `SELECT * FROM message WHERE
										msg_from = $1 AND msg_to = $2
										OR msg_from = $2 AND msg_to = $1
										ORDER BY time DESC OFFSET $3 LIMIT 10`, contactId1, contactId2, offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			message  msp.Message
			tempTime time.Time
		)

		err := rows.Scan(&message.MsgId, &message.From, &message.To, &message.Text, &tempTime, &message.Status)
		if err != nil {
			return nil, err
		}

		message.Time = protoTStamp.New(tempTime)
		messages.Messages = append(messages.Messages, &message)
	}

	return &messages, nil
}

// Change schema to hold message status
func InsertMessage(insertMessage types.WsMsg) (types.WsMsg, error) {
	var message types.WsMsg
	row := db.QueryRow(dbContext, "INSERT INTO message (msg_from, msg_to, text, status) VALUES ($1, $2, $3, $4) RETURNING *", insertMessage.From, insertMessage.To, insertMessage.Text, "saved")
	err := row.Scan(&message.MsgId, &message.From, &message.To, &message.Text, &message.Time, &message.Status)
	if err != nil {
		return message, err
	}
	return message, nil
}

func UpdateMessageStatus(msgId int64, status string) error {
	utils.Log.Println(msgId, status)
	commandTag, err := db.Exec(dbContext, "UPDATE message SET status = $1 WHERE message_id = $2", status, msgId)
	if err == nil && commandTag.RowsAffected() != 1 {
		return errors.New("Error updating message status at DB")
	}
	return err
}

func ContactMsgs(userId int64, contactId int64, offset int64) ([]types.Message, error) {
	var messages []types.Message
	rows, err := db.Query(dbContext, `SELECT * FROM message WHERE
										msg_from = $1 AND msg_to = $2
										OR msg_from = $2 AND msg_to = $1
										ORDER BY time DESC OFFSET $3 LIMIT 10`, userId, contactId, offset)

	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var message types.Message
		err := rows.Scan(&message.MessageId, &message.From, &message.To, &message.Text, &message.Time)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func UpdateLastSeen(userId int64, time time.Time) error {
	commandTag, err := db.Exec(dbContext, "UPDATE users SET last_seen = $1 WHERE user_id = $2", time, userId)
	if err != nil {
		return err
	} else if commandTag.RowsAffected() != 1 {
		return errors.New("Error updating user last seen at DB")
	}
	return nil
}

// func InsertUser(newUser types.RegisterUser) error {
// 	var (
// 		commandTag pgconn.CommandTag
// 		err        error
// 	)
// 	if *newUser.Profile_Pic != "" {
// 		commandTag, err = db.Exec(dbContext, "INSERT INTO users (name, email, profile_pic, password_hash) VALUES ($1, $2, $3, $4)", *newUser.Name, *newUser.Email, *newUser.Profile_Pic, *newUser.Password)
// 	} else {
// 		commandTag, err = db.Exec(dbContext, "INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)", *newUser.Name, *newUser.Email, *newUser.Password)
// 	}

// 	if commandTag.RowsAffected() != 1 {
// 		return errors.New("Failed to create a new account, try again")
// 	}
// 	return err
// }

// func UserExists(userEmail *string) (bool, error) {
// 	var exists bool
// 	row := db.QueryRow(dbContext, "SELECT EXISTS(SELECT * FROM users WHERE email = $1)", *userEmail)
// 	err := row.Scan(&exists)

// 	if err != nil {
// 		return false, err
// 	}
// 	return exists, nil
// }

// func GetUser(userEmail string) (types.UserWithPw, error) {
// 	var user types.UserWithPw
// 	row := db.QueryRow(dbContext, "SELECT user_id, name, email, profile_pic, password_hash FROM users WHERE email = $1", userEmail)
// 	err := row.Scan(&user.UserId, &user.Name, &user.Email, &user.Profile_Pic, &user.Password)
// 	if err != nil {
// 		return user, err
// 	}
// 	return user, nil
// }

// func UserExistsById(userId int64) (bool, error) {
// 	var exists bool
// 	row := db.QueryRow(dbContext, "SELECT EXISTS(SELECT user_id from users WHERE user_id = $1)", userId)
// 	err := row.Scan(&exists)
// 	if err != nil {
// 		return false, err
// 	}
// 	return exists, nil
// }

// func GetUserById(userId int64) (types.UserWithId, error) {
// 	var user types.UserWithId
// 	row := db.QueryRow(dbContext, "SELECT user_id, name, email, profile_pic, password_hash FROM users WHERE user_id = $1", userId)
// 	err := row.Scan(&user.UserId, &user.Name, &user.Email, &user.Profile_Pic, &user.Password)
// 	if err != nil {
// 		return user, err
// 	}
// 	return user, nil
// }

// func UpdateUserPw(userId int64, userPw string) error {
// 	commandTag, err := db.Exec(dbContext, "UPDATE users SET password_hash = $1 WHERE user_id = $2", userPw, userId)
// 	if err != nil {
// 		return err
// 	} else if commandTag.RowsAffected() != 1 {
// 		return errors.New("Error updating user password at DB")
// 	}
// 	return nil
// }

// func UpdateUserDP(userId int64, profImgLink string) error {
// 	commandTag, err := db.Exec(dbContext, "UPDATE users SET profile_pic = $1 WHERE user_id = $2", profImgLink, userId)
// 	if err != nil {
// 		return err
// 	} else if commandTag.RowsAffected() != 1 {
// 		return errors.New("Error updating user profile image at DB")
// 	}
// 	return nil
// }

// func UserContacts(userId int64) ([]types.Contact, error) {
// 	var contacts []types.Contact
// 	rows, err := db.Query(dbContext, `SELECT users.user_id, users.name, users.profile_pic FROM users
// 										JOIN user_contact
// 										ON users.user_id = user_contact.user_1_id AND user_contact.user_2_id = $1
// 										OR users.user_id = user_contact.user_2_id AND user_contact.user_1_id = $1`, userId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for rows.Next() {
// 		var contact types.Contact
// 		err := rows.Scan(&contact.UserId, &contact.Name, &contact.Profile_Pic)
// 		if err != nil {
// 			return nil, err
// 		}
// 		contacts = append(contacts, contact)
// 	}
// 	return contacts, nil
// }

// func ContactExists(userId int64, contactId int64) bool {
// 	var exists bool
// 	row := db.QueryRow(dbContext, `SELECT EXISTS(SELECT * FROM user_contact
// 									WHERE user_1_id = $1 AND user_2_id = $2)`, userId, contactId)
// 	err := row.Scan(&exists)
// 	if err != nil {
// 		return false
// 	}
// 	return exists
// }

// func AddContact(userId1 int64, userId2 int64) error {
// 	commandTag, err := db.Exec(dbContext, `INSERT INTO user_contact (user_1_id, user_2_id)
// 											VALUES ($1, $2)`, userId1, userId2)
// 	if err != nil {
// 		return err
// 	} else if commandTag.RowsAffected() != 1 {
// 		return errors.New("Error inserting new contact at DB")
// 	}
// 	return nil
// }

// func GetUserLastSeen(userId int64) (types.UserLastSeen, error) {
// 	row := db.QueryRow(dbContext, "SELECT date_trunc('second', last_seen) FROM users WHERE user_id = $1", userId)
// 	var resultTime types.UserLastSeen
// 	err := row.Scan(&resultTime.UserLastSeenTime)
// 	if err != nil {
// 		return resultTime, err
// 	}
// 	return resultTime, nil
// }
