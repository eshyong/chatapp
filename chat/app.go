package chat

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const staticDir = "/static/"

type Application struct {
	router          *mux.Router
	chatUsers       []*ChatUser
	upgrader        *websocket.Upgrader
	staticFilesPath string
}

type ChatUser struct {
	userName string
	userConn *websocket.Conn
}

func NewApp() *Application {
	return &Application{
		chatUsers: []*ChatUser{},
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		staticFilesPath: filepath.Join(".", staticDir),
	}
}

func (a *Application) SetupRouter() *mux.Router {
	r := mux.NewRouter()
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(a.staticFilesPath))))
	r.HandleFunc("/", a.serveHomePage)
	r.HandleFunc("/login", a.serveLoginPage)
	r.HandleFunc("/chat-room", a.acceptChatConnection)
	return r
}

func (a *Application) serveHomePage(w http.ResponseWriter, r *http.Request) {
	a.serveHtmlPage(w, r, "index")
}

func (a *Application) serveLoginPage(w http.ResponseWriter, r *http.Request) {
	a.serveHtmlPage(w, r, "login")
}

func (a *Application) serveHtmlPage(w http.ResponseWriter, r *http.Request, name string) {
	http.ServeFile(w, r, filepath.Join(a.staticFilesPath, "html", name+".html"))
}

func (a *Application) acceptChatConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	newUser := a.authenticateUser(conn)
	if newUser != nil {
		a.chatUsers = append(a.chatUsers, newUser)
		go a.createUserSession(newUser)
	}
}

// TODO: create a real chat protocol
func (a *Application) authenticateUser(conn *websocket.Conn) *ChatUser {
	log.Println("User connected from " + conn.RemoteAddr().String())
	messageType, message, err := conn.ReadMessage()
	if messageType != websocket.TextMessage {
		err = errors.New("Required text message, got binary message instead")
	}
	if err != nil {
		log.Println("Unable to authenticate user. Error:")
		log.Println(err)
		return nil
	}

	// Create a new user
	chatUser := &ChatUser{
		userName: string(message),
		userConn: conn,
	}
	log.Println("Authenticated user: " + chatUser.userName)
	return chatUser
}

func (a *Application) createUserSession(chatUser *ChatUser) {
	defer chatUser.userConn.Close()
	for {
		messageType, message, err := chatUser.userConn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		if messageType != websocket.TextMessage {
			// Send an error to the user
			chatUser.userConn.WriteMessage(websocket.TextMessage, []byte("Unable to handle binary messages"))
			continue
		}

		a.broadcastMessage(chatUser.userName, message)
	}
}

func (a *Application) broadcastMessage(sender string, message []byte) {
	log.Println("Sending message from: " + sender)
	for _, user := range a.chatUsers {
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
