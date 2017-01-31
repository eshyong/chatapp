package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/eshyong/chatapp/chat/models"
	"github.com/eshyong/chatapp/chat/repository"
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

type ChatSession struct {
	UserName string
	UserConn *websocket.Conn
}

type Application struct {
	connectedUsers  []*ChatSession
	repository      *repository.Repository
	router          *mux.Router
	secureCookie    *securecookie.SecureCookie
	staticFilesPath string
	upgrader        *websocket.Upgrader
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
		connectedUsers:  []*ChatSession{},
		secureCookie:    securecookie.New([]byte(hashKey), []byte(blockKey)),
		staticFilesPath: filepath.Join(".", staticDir),
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		repository: repository.NewUserRepository(dbConn),
	}
}

func (a *Application) SetupRouter() *mux.Router {
	r := mux.NewRouter()
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(a.staticFilesPath))))
	r.HandleFunc("/", a.serveHomePage).Methods("GET")
	r.HandleFunc("/login", a.loginHandler).Methods("GET", "POST")
	r.HandleFunc("/register", a.registrationHandler).Methods("POST")
	r.HandleFunc("/chatroom", a.chatRoomHandler).Methods("POST")
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

func (a *Application) loginHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Println("readUserCreds:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storedCreds, err := a.repository.FindUserByName(requestCreds.UserName)
	if err != nil {
		log.Println("UserRepository.FindUserByName: ", err)
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

func (a *Application) registrationHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("POST /register")
	userCreds, err := readUserCreds(r)
	if err != nil {
		log.Println("readUserCreds: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func readUserCreds(r *http.Request) (*models.ChatUser, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// Probably EOF errors, according to golang docs
		return nil, err
	}

	u := &models.ChatUser{}
	if err := json.Unmarshal(body, u); err != nil {
		return nil, err
	}
	if u.UserName == "" || u.Password == "" {
		return nil, errors.New("Unable to parse user request as JSON")
	}
	return u, nil
}

func (a *Application) createUserSession(w http.ResponseWriter, r *http.Request, userName string) error {
	log.Println("createUserSession")
	session := map[string]string{
		"user_name":     userName,
		"authenticated": "true",
	}
	cookieName := "user_session"
	encoded, err := a.secureCookie.Encode(cookieName, session)
	if err != nil {
		log.Println("Unable to set cookie")
		return err
	}
	cookie := &http.Cookie{
		Name:     cookieName,
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
	cookieName := "user_session"
	if cookie, err := r.Cookie(cookieName); err == nil {
		var sessionValues map[string]string
		if err = a.secureCookie.Decode(cookieName, cookie.Value, &sessionValues); err != nil {
			log.Println(err)
			return false
		}
		return sessionValues["authenticated"] == "true"
	}
	log.Println("Cookie not set")
	return false
}

func (a *Application) registerUser(u *models.ChatUser) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("bcrypt hash failed")
	}

	return a.repository.InsertUser(u.UserName, string(hashedPassword))
}

func (a *Application) chatRoomHandler(w http.ResponseWriter, r *http.Request) {
	createRequest, err := readCreateChatRoomRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = a.repository.CreateChatRoom(createRequest.Name, createRequest.CreatedBy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
}

func readCreateChatRoomRequest(r *http.Request) (*models.CreateChatRoomRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	createRequest := &models.CreateChatRoomRequest{}
	err = json.Unmarshal(body, createRequest)
	if err != nil {
		return nil, err
	}

	if createRequest.Name == "" || createRequest.CreatedBy == "" {
		return nil, errors.New("Unable to read create chat room request")
	}
	return createRequest, nil
}

func (a *Application) acceptChatConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	newUser := authenticateUser(conn)
	if newUser != nil {
		a.connectedUsers = append(a.connectedUsers, newUser)
		go a.handleChatSession(newUser)
	}
}

// TODO: create a real chat protocol
func authenticateUser(conn *websocket.Conn) *ChatSession {
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
	chatSession := &ChatSession{
		UserName: string(message),
		UserConn: conn,
	}
	log.Println("Authenticated user: " + chatSession.UserName)
	return chatSession
}

func (a *Application) handleChatSession(chatSession *ChatSession) {
	defer chatSession.UserConn.Close()
	for {
		messageType, message, err := chatSession.UserConn.ReadMessage()
		if err != nil {
			log.Println("chatUser.userConn.ReadMessage: ", err)
			break
		}

		if messageType != websocket.TextMessage {
			// Send an error to the user
			chatSession.UserConn.WriteMessage(websocket.TextMessage, []byte("Unable to handle binary messages"))
			continue
		}

		a.broadcastMessage(chatSession.UserName, message)
	}
}

func (a *Application) broadcastMessage(sender string, message []byte) {
	log.Println("Sending message from: " + sender)
	for _, user := range a.connectedUsers {
		// Avoid broadcasting message to sender
		if user.UserName != sender {
			log.Println("Sending message to: " + user.UserName)
			if err := user.UserConn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Unable to send message to " + user.UserName)
				continue
			}
		}
	}
}
