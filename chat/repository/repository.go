package repository

import (
	"database/sql"
	"errors"

	"github.com/eshyong/chatapp/chat/models"
)

type Repository struct {
	dbConn *sql.DB
}

func New(dbConn *sql.DB) *Repository {
	return &Repository{dbConn: dbConn}
}

func (r *Repository) FindUserByName(name string) (*models.ChatUser, error) {
	u := &models.ChatUser{}
	err := r.dbConn.QueryRow(
		"SELECT id, user_name, hashed_password FROM chat_user WHERE user_name = $1",
		name,
	).Scan(&u.Id, &u.UserName, &u.Password)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) InsertUser(userName, hashedPassword string) error {
	result, err := r.dbConn.Exec(
		"INSERT INTO chat_user (user_name, hashed_password) VALUES ($1, $2)",
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
	result, err := r.dbConn.Exec(
		"INSERT INTO chat_room (room_name, created_by) VALUES ($1, $2)",
		roomName, createdBy,
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
	_, err := r.dbConn.Exec("DELETE FROM chat_room WHERE room_name=$1", roomName)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) ListChatRooms() (*models.ChatRoomList, error) {
	rows, err := r.dbConn.Query("SELECT id, room_name, created_by FROM chat_room")
	if err != nil {
		return nil, err
	}
	chatRoomList := &models.ChatRoomList{
		Results: []*models.ChatRoom{},
	}
	defer rows.Close()
	for rows.Next() {
		chatRoom := &models.ChatRoom{}
		if err := rows.Scan(&chatRoom.Id, &chatRoom.RoomName, &chatRoom.CreatedBy); err != nil {
			return nil, err
		}
		chatRoomList.Results = append(chatRoomList.Results, chatRoom)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return chatRoomList, nil
}

func (r *Repository) FindChatRoomByName(roomName string) (*models.ChatRoom, error) {
	row := r.dbConn.QueryRow("SELECT id, room_name, created_by FROM chat_room WHERE room_name=$1", roomName)
	chatRoom := &models.ChatRoom{}
	if err := row.Scan(&chatRoom.Id, &chatRoom.RoomName, &chatRoom.CreatedBy); err != nil {
		return nil, err
	}
	return chatRoom, nil
}

func (r *Repository) InsertChatMessage(roomId int, message *models.ChatMessage) error {
	result, err := r.dbConn.Exec(
		"INSERT INTO chat_message (time_sent, sent_by, chat_room_id, contents) "+
			"VALUES ($1::timestamp, $2, $3, $4)",
		message.TimeSent, message.SentBy, roomId, message.Contents,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("Unable to insert chat message")
	}
	return nil
}

func (r *Repository) GetChatMessagesByRoomId(roomId int) ([]*models.ChatMessage, error) {
	rows, err := r.dbConn.Query(
		"SELECT time_sent, sent_by, contents FROM chat_message WHERE chat_message.chat_room_id = $1",
		roomId)
	if err != nil {
		return nil, err
	}
	chatMessages := []*models.ChatMessage{}

	defer rows.Close()
	for rows.Next() {
		chatMessage := &models.ChatMessage{}
		if err := rows.Scan(&chatMessage.TimeSent, &chatMessage.SentBy, &chatMessage.Contents); err != nil {
			return nil, err
		}
		chatMessages = append(chatMessages, chatMessage)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return chatMessages, nil
}
