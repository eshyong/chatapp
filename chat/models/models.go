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
	RoomName  string
	CreatedBy string
}

type ChatRoom struct {
	Id        int    `json:"id"`
	RoomName  string `json:"roomName"`
	CreatedBy string `json:"createdBy"`
}

type ChatRoomList struct {
	Results []*ChatRoom `json:"results"`
}

type UserInfo struct {
	Authenticated bool   `json:"authenticated"`
	UserName      string `json:"userName"`
}
