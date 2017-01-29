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
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	staticDir = "/static/"
	// 15 minutes
	cookieMaxAge = 900
)

type Application struct {
	chatUsers       []*ChatUser
	dbConn          *sql.DB
	router          *mux.Router
	secureCookie    *securecookie.SecureCookie
	staticFilesPath string
	upgrader        *websocket.Upgrader
}

type ChatUser struct {
	userName string
	userConn *websocket.Conn
}

type UserCreds struct {
	UserName string
	Password string
}

func NewApp(hashKey, blockKey string) *Application {
	dbConn, err := sql.Open("postgres", "postgres://chatapp:chatapp@localhost/chatapp?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	// Execute a dummy query to test the connection.
	_, err = dbConn.Exec("SELECT current_user")
	if err != nil {
		log.Fatal("Unable to connect to database: ", err)
	}

	return &Application{
		chatUsers:       []*ChatUser{},
		dbConn:          dbConn,
		secureCookie:    securecookie.New([]byte(hashKey), []byte(blockKey)),
		staticFilesPath: filepath.Join(".", staticDir),
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (a *Application) SetupRouter() *mux.Router {
	r := mux.NewRouter()
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(a.staticFilesPath))))
	r.HandleFunc("/", a.serveHomePage).Methods("GET")
	r.HandleFunc("/login", a.handleLogin).Methods("GET", "POST")
	r.HandleFunc("/register", a.handleRegistration).Methods("POST")
	r.HandleFunc("/chat-room", a.acceptChatConnection).Methods("GET")
	return r
}

func (a *Application) serveHomePage(w http.ResponseWriter, r *http.Request) {
	log.Println("GET /")
	if a.isUserAuthenticated(r) {
		a.serveHtmlPage(w, r, "index")
	} else {
		// Redirect unauthorized users
		a.serveHtmlPage(w, r, "login")
	}
}

func (a *Application) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Page request
		a.serveLoginPage(w, r)
	case "POST":
		// Login request
		a.loginUser(w, r)
	}
}

func (a *Application) serveLoginPage(w http.ResponseWriter, r *http.Request) {
	log.Println("GET /login")
	a.serveHtmlPage(w, r, "login")
}

func (a *Application) serveHtmlPage(w http.ResponseWriter, r *http.Request, name string) {
	http.ServeFile(w, r, filepath.Join(a.staticFilesPath, "html", name+".html"))
}

func (a *Application) loginUser(w http.ResponseWriter, r *http.Request) {
	log.Println("POST /login")
	requestCreds, err := readUserCreds(r)
	if err != nil {
		log.Println("readUserCreds: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	storedCreds, err := a.findUserByName(requestCreds.UserName)
	if err != nil {
		log.Println("Application.findUserByName: ", err)
		if pqErr, ok := err.(*pq.Error); ok {
			log.Println(pqErr.Code.Name())
		}
		if err == sql.ErrNoRows {
			http.Error(w, "No user found with that name", http.StatusBadRequest)
			return
		}
		http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
		return
	}

	hashedPassword := storedCreds.Password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(requestCreds.Password)); err != nil {
		log.Println("bcrypt.CompareHashAndPassword: ", err)
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	if err := a.createUserSession(w, r, requestCreds.UserName); err != nil {
		log.Println("Application.createUserSession: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *Application) findUserByName(userName string) (*UserCreds, error) {
	u := &UserCreds{}
	err := a.dbConn.QueryRow(
		"SELECT username, hashed_password FROM data.chat_users WHERE username = $1",
		userName,
	).Scan(&u.UserName, &u.Password)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (a *Application) handleRegistration(w http.ResponseWriter, r *http.Request) {
	log.Println("POST /register")
	userCreds, err := readUserCreds(r)
	if err != nil {
		log.Println("readUserCreds: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := a.registerUser(userCreds); err != nil {
		log.Println("Application.registerUser: ", err)
		if pqErr, ok := err.(*pq.Error); ok {
			log.Println(pqErr.Code.Name())
			if pqErr.Code.Name() == "unique_violation" {
				http.Error(w, "A user with that name already exists", http.StatusBadRequest)
				return
			}
		}
		http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
		return
	}

	if err := a.createUserSession(w, r, userCreds.UserName); err != nil {
		log.Println("Application.createUserSession: ", err)
		http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func readUserCreds(r *http.Request) (*UserCreds, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// Probably EOF errors, according to golang docs
		return nil, err
	}

	u := &UserCreds{}
	if err := json.Unmarshal(body, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (a *Application) createUserSession(w http.ResponseWriter, r *http.Request, userName string) error {
	log.Println("createUserSession")
	encoded, err := a.secureCookie.Encode("authenticated", true)
	if err != nil {
		log.Println("Unable to set cookie")
		return err
	}
	cookie := &http.Cookie{
		Name:     "authenticated",
		Value:    encoded,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   cookieMaxAge,
	}
	http.SetCookie(w, cookie)
	return nil
}

func (a *Application) isUserAuthenticated(r *http.Request) bool {
	log.Println("isUserAuthenticated")
	if cookie, err := r.Cookie("authenticated"); err == nil {
		var authenticated bool
		if err = a.secureCookie.Decode("authenticated", cookie.Value, &authenticated); err != nil {
			log.Println(err)
		}
		return authenticated
	}
	log.Println("Cookie not set")
	return false
}

func (a *Application) registerUser(l *UserCreds) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(l.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("bcrypt hash failed")
	}

	result, err := a.dbConn.Exec(
		"INSERT INTO data.chat_users (username, hashed_password) VALUES ($1, $2)",
		l.UserName, string(hashedPassword))
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		return errors.New("Unable to create new user. Please try again")
	}
	return nil
}

func (a *Application) acceptChatConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	newUser := authenticateUser(conn)
	if newUser != nil {
		a.chatUsers = append(a.chatUsers, newUser)
		go a.handleChatSession(newUser)
	}
}

// TODO: create a real chat protocol
func authenticateUser(conn *websocket.Conn) *ChatUser {
	log.Println("User connected from " + conn.RemoteAddr().String())
	messageType, message, err := conn.ReadMessage()
	if messageType != websocket.TextMessage {
		err = errors.New("Required text message, got binary message instead")
	}
	if err != nil {
		log.Println("authenticateUser: ", err)
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

func (a *Application) handleChatSession(chatUser *ChatUser) {
	defer chatUser.userConn.Close()
	for {
		messageType, message, err := chatUser.userConn.ReadMessage()
		if err != nil {
			log.Println("chatUser.userConn.ReadMessage: ", err)
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
