package repository

import (
	"database/sql"
	"errors"

	"github.com/eshyong/chatapp/chat/models"
)

type UserRepository struct {
	dbConn *sql.DB
}

func NewUserRepository(dbConn *sql.DB) *UserRepository {
	return &UserRepository{dbConn: dbConn}
}

func (ur *UserRepository) FindUserByName(name string) (*models.UserCreds, error) {
	u := &models.UserCreds{}
	err := ur.dbConn.QueryRow(
		"SELECT username, hashed_password FROM data.chat_users WHERE username = $1",
		name,
	).Scan(&u.UserName, &u.Password)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (ur *UserRepository) InsertUser(userName, hashedPassword string) error {
	result, err := ur.dbConn.Exec(
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
