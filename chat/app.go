package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"

	"github.com/eshyong/chatapp/chat/models"
	"github.com/eshyong/chatapp/chat/repository"
	"github.com/eshyong/chatapp/chat/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	staticDir = "/static/"
	// 1 day
	cookieMaxAge = 86400
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
	r.StrictSlash(true)
	// Serve static files
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir(a.staticFilesPath))))

	// Main pages
	r.Handle("/", a.checkAuthentication(a.indexHandler())).Methods("GET")
	r.Handle("/login", a.loginHandler()).Methods("GET", "POST")
	r.Handle("/register", a.registrationHandler()).Methods("POST")

	// User auth
	r.Handle("/user/current", a.userInfo())

	// API router
	api := r.PathPrefix("/api").Subrouter()
	api.Handle("/chatroom", a.checkAuthentication(a.chatRoomHandler())).Methods("POST")
	api.Handle("/chatroom/{name}", a.checkAuthentication(a.chatRoomHandler())).Methods("DELETE")
	api.Handle("/chatroom/list", a.checkAuthentication(a.listChatRoomsHandler())).Methods("GET")
	return r
}

func (a *Application) userInfo() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.getUserInfo(r)
		if err != nil {
			if err == http.ErrNoCookie {
				http.Error(w, "Your session expired. Please login again", http.StatusBadRequest)
				return
			}
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		body, err := json.Marshal(user)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})
}

func (a *Application) checkAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.isUserAuthenticated(r) && r.URL.Path != "/login" {
			http.Error(w, "Please login to access the app", http.StatusUnauthorized)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (a *Application) indexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("GET /")
		a.serveHtmlPage(w, r, "login")
	})
}

func (a *Application) loginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// Page request
			a.serveLoginPage(w, r)
		case "POST":
			// Login request
			a.loginUser(w, r)
		}
	})
}

func (a *Application) registrationHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("POST /register")
		registerRequest := &models.RegisterRequest{}
		if err := utils.UnmarshalJsonRequest(r, registerRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if registerRequest.UserName == "" || registerRequest.Password == "" {
			http.Error(w, `"username" and "password" fields cannot be empty`, http.StatusBadRequest)
			return
		}

		if err := a.registerUser(registerRequest); err != nil {
			log.Println("Application.registerUser: ", err)
			if pqErr, ok := err.(*pq.Error); ok {
				if pqErr.Code.Name() == "unique_violation" {
					http.Error(w, "A user with that name already exists", http.StatusBadRequest)
					return
				}
			}
			http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
			return
		}

		if err := a.createUserSession(w, r, registerRequest.UserName); err != nil {
			log.Println("Application.createUserSession: ", err)
			http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (a *Application) chatRoomHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			a.createChatRoom(w, r)
		case "DELETE":
			a.deleteChatRoom(w, r)
		}
	})
}

func (a *Application) listChatRoomsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatRoomList, err := a.repository.ListChatRooms()
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry, try again later", http.StatusInternalServerError)
			return
		}
		responseBody, err := json.Marshal(chatRoomList)
		if err != nil {
			http.Error(w, "Unable to send JSON response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	})
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
	var loginRequest models.LoginRequest
	if err := utils.UnmarshalJsonRequest(r, &loginRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if loginRequest.UserName == "" || loginRequest.Password == "" {
		http.Error(w, `"username" and "password" fields cannot be empty`, http.StatusBadRequest)
		return
	}

	user, err := a.repository.FindUserByName(loginRequest.UserName)
	if err != nil {
		log.Println("UserRepository.FindUserByName: ", err)
		if err == sql.ErrNoRows {
			http.Error(w, "No user found with that name", http.StatusBadRequest)
			return
		}
		http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
		return
	}

	hashedPassword := user.Password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginRequest.Password)); err != nil {
		log.Println("bcrypt.CompareHashAndPassword: ", err)
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	if err := a.createUserSession(w, r, loginRequest.UserName); err != nil {
		log.Println("Application.createUserSession: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *Application) createUserSession(w http.ResponseWriter, r *http.Request, userName string) error {
	log.Println("createUserSession")
	session := map[string]string{
		"userName":      userName,
		"authenticated": "true",
	}
	cookieName := "userSession"
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
	log.Println("Authenticated user " + userName)
	return nil
}

func (a *Application) getUserInfo(r *http.Request) (*models.UserInfo, error) {
	cookieName := "userSession"
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	var sessionValues map[string]string
	if err = a.secureCookie.Decode(cookieName, cookie.Value, &sessionValues); err != nil {
		log.Println(err)
		return nil, err
	}

	return &models.UserInfo{
		Authenticated: sessionValues["authenticated"] == "true",
		UserName:      sessionValues["userName"],
	}, nil
}

func (a *Application) isUserAuthenticated(r *http.Request) bool {
	userInfo, err := a.getUserInfo(r)
	if err != nil {
		return false
	}
	return userInfo.Authenticated
}

func (a *Application) registerUser(r *models.RegisterRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("bcrypt hash failed")
	}

	return a.repository.InsertUser(r.UserName, string(hashedPassword))
}

func (a *Application) createChatRoom(w http.ResponseWriter, r *http.Request) {
	var createRequest models.CreateChatRoomRequest
	if err := utils.UnmarshalJsonRequest(r, &createRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if createRequest.RoomName == "" || createRequest.CreatedBy == "" {
		http.Error(w, `"name" and "createdBy" fields cannot be empty`, http.StatusBadRequest)
		return
	}

	err := a.repository.CreateChatRoom(createRequest.RoomName, createRequest.CreatedBy)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				http.Error(w, "A chat room with that name has already been created", http.StatusBadRequest)
				return
			}
		}
		http.Error(w, "Sorry, something went wrong. Please try again later", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *Application) deleteChatRoom(w http.ResponseWriter, r *http.Request) {
	roomName := mux.Vars(r)["name"]
	if err := a.repository.DeleteChatRoom(roomName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
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
