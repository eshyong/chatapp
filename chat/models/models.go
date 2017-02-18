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
	RoomName  string `json:"roomName"`
	CreatedBy string `json:"createdBy"`
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

type ChatMessage struct {
	SentBy   string `json:"sentBy"`
	Contents string `json:"contents"`
	TimeSent string `json:"timeSent"`
}

// Websocket chat protocol struct
type WsServerMessage struct {
	// If there was an error in processing a websocket request, this field will be true
	Error bool `json:"error"`
	// A string describing the reason for an error
	Reason string `json:"reason"`
	// A variable length slice containing chat messages to send to the client
	Body []*ChatMessage `json:"body"`
}
