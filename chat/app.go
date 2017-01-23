package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const staticDir = "/static/"

type Application struct {
	dbConn          *sql.DB
	router          *mux.Router
	chatUsers       []*ChatUser
	upgrader        *websocket.Upgrader
	staticFilesPath string
}

type ChatUser struct {
	userName string
	userConn *websocket.Conn
}

type loginRequest struct {
	UserName string
	Password string
}

func NewApp() *Application {
	db, err := sql.Open("postgres", "postgres://chatapp:chatapp@localhost/chatapp?sslmode=disable")
	if err != nil {
		log.Println("Unable to connect to database")
		log.Fatal(err)
	}
	return &Application{
		dbConn:    db,
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
	r.HandleFunc("/login", a.handleLoginPage)
	r.HandleFunc("/register", a.handleRegistration)
	r.HandleFunc("/chat-room", a.acceptChatConnection)
	return r
}

func (a *Application) serveHomePage(w http.ResponseWriter, r *http.Request) {
	a.serveHtmlPage(w, r, "index")
}

func (a *Application) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		a.serveLoginPage(w, r)
	case "POST":
		a.loginUser(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *Application) serveLoginPage(w http.ResponseWriter, r *http.Request) {
	a.serveHtmlPage(w, r, "login")
}

func (a *Application) loginUser(w http.ResponseWriter, r *http.Request) {
	// TODO
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

func (a *Application) handleRegistration(w http.ResponseWriter, r *http.Request) {
	log.Println("Got register request")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// Probably EOF errors, according to golang docs
		log.Println(err)
		return
	}

	l := &loginRequest{}
	if err := json.Unmarshal(body, l); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := a.registerUser(l); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Redirect to index page on success
	http.Redirect(w, r, "/", http.StatusPermanentRedirect)
}

func (a *Application) registerUser(l *loginRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(l.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("bcrypt hash failed")
	}

	result, err := a.dbConn.Exec(
		"INSERT INTO data.chat_users (username, hashed_password) VALUES ($1, $2)",
		l.UserName, string(hashedPassword))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			log.Println(pqErr.Code.Name())
		}
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		return errors.New("Unable to create new user. Please try again")
	}
	return nil
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
