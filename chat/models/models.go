package models

type UserCreds struct {
	Id       int
	UserName string
	Password string
}

type ChatRoom struct {
	Name      string
	CreatedBy int
}
