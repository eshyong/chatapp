package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"path/filepath"
	"github.com/gorilla/websocket"
	"log"
)

type ChatServer struct {
	router *mux.Router
	chatRoom *ChatRoom
	upgrader *websocket.Upgrader
	staticFilesPath string
}

type ChatUser struct {
	userName string
	userConn *websocket.Conn
}

type ChatRoom struct {
	chatUsers []*ChatUser
}

const staticDir = "/static/"

func main() {
	chatServer := &ChatServer{
		chatRoom: &ChatRoom{},
		upgrader: &websocket.Upgrader{
			ReadBufferSize: 1024,
			WriteBufferSize: 1024,
		},
		staticFilesPath: filepath.Join(".", staticDir),
	}
	http.Handle("/", chatServer.setupRouter())

	addr := ":8080"
	log.Println("Starting server on " + addr)
	http.ListenAndServe(addr, nil)
}

func (c *ChatServer) setupRouter() *mux.Router {
	r := mux.NewRouter()
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(c.staticFilesPath))))
	r.HandleFunc("/", c.serveHomePage)
	r.HandleFunc("/chat-room", c.acceptChatConnection)
	return r
}

func (c *ChatServer) serveHomePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(c.staticFilesPath, "html", "index.html"))
}

func (c *ChatServer) acceptChatConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c.chatRoom.handleUser(conn)
}

// TODO: create a real chat protocol
func (c *ChatRoom) handleUser(conn *websocket.Conn) {
	log.Println("User connected from " + conn.RemoteAddr().String())
	messageType, message, err := conn.ReadMessage()
	if err != nil || messageType != websocket.TextMessage {
		log.Println("Unable to authenticate user. Error:")
		log.Println(err)
		return
	}
	name := string(message)
	log.Println("Authenticated user: " + name)
	chatUser := &ChatUser{
		userName: name,
		userConn: conn,
	}
	c.chatUsers = append(c.chatUsers, chatUser)

	// Handle user connections in a separate goroutine
	go func() {
		defer conn.Close()
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			if messageType != websocket.TextMessage {
				// Send an error to the user
				chatUser.userConn.WriteMessage(websocket.TextMessage, []byte("Unable to handle binary messages"))
				continue
			}

			c.broadcastMessage(chatUser.userName, message)
		}
	}()
}

func (c *ChatRoom) broadcastMessage(sender string, message []byte) {
	log.Println("Sending message from: " + sender)
	for _, user := range c.chatUsers {
		// Avoid broadcasting message to sender
		if user.userName != sender {
			log.Println("Sending message to: " + user.userName)
			if err := user.userConn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Unable to send message to " + user.userName)
				continue
			}
		}
	}
}
