package repository

import (
	"database/sql"
	"errors"

	"github.com/eshyong/chatapp/chat/models"
)

type Repository struct {
	dbConn *sql.DB
}

func NewUserRepository(dbConn *sql.DB) *Repository {
	return &Repository{dbConn: dbConn}
}

func (r *Repository) FindUserByName(name string) (*models.ChatUser, error) {
	u := &models.ChatUser{}
	err := r.dbConn.QueryRow(
		"SELECT id, username, hashed_password FROM data.chat_users WHERE username = $1",
		name,
	).Scan(&u.Id, &u.UserName, &u.Password)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) InsertUser(userName, hashedPassword string) error {
	result, err := r.dbConn.Exec(
		"INSERT INTO data.chat_users (username, hashed_password) VALUES ($1, $2)",
		userName, hashedPassword)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("Unable to create new user")
	}
	return nil
}

func (r *Repository) CreateChatRoom(roomName, createdBy string) error {
	user, err := r.FindUserByName(createdBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("User with that name does not exist")
		}
		return err
	}

	result, err := r.dbConn.Exec(
		"INSERT INTO data.chat_rooms (name, created_by) VALUES ($1, $2)",
		roomName, user.Id,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("Unable to create new chatroom")
	}
	return nil
}


func (r *Repository) DeleteChatRoom(roomName string) error {
	_, err := r.dbConn.Exec("DELETE FROM data.chat_rooms WHERE name=$1", roomName)
	if err != nil {
		return err
	}
	return nil
}
