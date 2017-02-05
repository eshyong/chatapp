package models

type LoginRequest struct {
	UserName string
	Password string
}

type RegisterRequest LoginRequest

type ChatUser struct {
	Id       int
	UserName string
	Password string
}

type CreateChatRoomRequest struct {
	Name      string
	CreatedBy string
}

type ChatRoom struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	CreatedBy int    `json:"createdBy"`
}

type ChatRoomList struct {
	Results []*ChatRoom `json:"results"`
}
