package structs

import "github.com/gorilla/websocket"

type ChatUser struct {
	UserName string
	UserConn *websocket.Conn
}

type UserCreds struct {
	UserName string
	Password string
}
