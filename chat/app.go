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
	"golang.org/x/crypto/bcrypt"
)

const (
	// Output directory of react build
	buildDir = "/frontend/build/"
	// 1 day
	cookieMaxAge = 86400
)

type ChatSession struct {
	UserName string
	UserConn *websocket.Conn
}

type ChatRoom struct {
	roomId       int
	chatSessions []*ChatSession
}

type Application struct {
	// A directory of chat rooms
	chatRoomDirectory map[string]*ChatRoom

	// A repository object used for database access
	repository *repository.Repository

	// HTTP routing/security
	router          *mux.Router
	secureCookie    *securecookie.SecureCookie
	staticFilesPath string

	// Websocket connector
	upgrader *websocket.Upgrader
}

func NewApp(hashKey, blockKey, env string) *Application {
	// TODO: this is ok for now since postgres is local. If we ever use a remote postgres instance, provision
	// passwords
	dbConn, err := sql.Open("postgres", "postgres://chatapp:chatapp@localhost/chatapp?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	// Execute a dummy query to test the connection.
	_, err = dbConn.Exec("SELECT current_user")
	if err != nil {
		log.Fatal("Unable to connect to database: ", err)
	}

	var checkOrigin func(r *http.Request) bool
	if env == "prod" {
		checkOrigin = nil
	} else {
		checkOrigin = func(r *http.Request) bool {
			return true
		}
	}

	return &Application{
		chatRoomDirectory: make(map[string]*ChatRoom),
		secureCookie:      securecookie.New([]byte(hashKey), []byte(blockKey)),
		staticFilesPath:   filepath.Join(".", buildDir),
		upgrader: &websocket.Upgrader{
			CheckOrigin:     checkOrigin,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
		repository: repository.NewUserRepository(dbConn),
	}
}

func (a *Application) SetupRouter() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	// Serve static files at the NPM build root
	r.PathPrefix("/static/").Handler(http.FileServer(http.Dir(a.staticFilesPath)))

	// Login/register routes
	r.Handle("/login", a.loginHandler()).Methods("POST")
	r.Handle("/register", a.registrationHandler()).Methods("POST")

	// User auth
	r.Handle("/user/current", a.userInfo())

	// API router
	api := r.PathPrefix("/api").Subrouter()
	api.Handle("/chatroom", a.checkAuthentication(a.chatRoomHandler())).Methods("POST")
	// Order matters!
	api.Handle("/chatroom/list", a.checkAuthentication(a.listChatRoomsHandler())).Methods("GET")
	api.Handle("/chatroom/{name}", a.checkAuthentication(a.chatRoomHandler())).Methods("DELETE")
	api.Handle("/chatroom/{name}/join", a.checkAuthentication(a.chatRoomHandler())).Methods("GET")

	// Whitelisted routers to get frontend routing to work
	// Unfortunately, using a wildcard router such as "/{.*}" seems to result in an infinite redirect loop, so we
	// have to specify each route individually here.
	r.Handle("/", a.indexHandler()).Methods("GET")
	r.Handle("/login", a.indexHandler()).Methods("GET")
	r.Handle("/chatroom/{name}", a.indexHandler()).Methods("GET")
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
		http.ServeFile(w, r, filepath.Join(a.staticFilesPath, "index.html"))
	})
}

func (a *Application) serveHtmlPage(w http.ResponseWriter, r *http.Request, name string) {
	http.ServeFile(w, r, filepath.Join(a.staticFilesPath, name+".html"))
}

func (a *Application) loginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.loginUser(w, r)
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
		case "GET":
			name := mux.Vars(r)["name"]
			log.Println("GET /chatroom/" + name + "/join")
			a.acceptChatConnection(w, r)
		case "POST":
			log.Println("POST /chatroom")
			a.createChatRoom(w, r)
		case "DELETE":
			log.Println("DELETE /chatroom/{name}")
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
		http.Error(w, `"roomName" and "createdBy" fields cannot be empty`, http.StatusBadRequest)
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
	// Create websocket connection
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("User connected from " + conn.RemoteAddr().String())

	// Get user info if possible, and send an error message if not
	userInfo, err := a.getUserInfo(r)
	if err != nil {
		conn.WriteJSON(&models.WsServerMessage{
			Error:  true,
			Reason: "Could not find user with that name",
		})
		conn.Close()
		return
	}

	// Check if chat room exists in database
	roomName := mux.Vars(r)["name"]
	roomModel, err := a.repository.FindChatRoomByName(roomName)
	if err != nil {
		if err == sql.ErrNoRows {
			conn.WriteJSON(&models.WsServerMessage{
				Error:  true,
				Reason: "Could not find room with that name",
			})
			conn.Close()
			return
		}
		conn.WriteJSON(&models.WsServerMessage{
			Error:  true,
			Reason: "Sorry, please try again later",
		})
		conn.Close()
		return
	}
	chatHistory, err := a.repository.GetChatMessagesByRoomId(roomModel.Id)
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}
	// TODO: send error
	conn.WriteJSON(&models.WsServerMessage{
		Error: false,
		Body:  chatHistory,
	})

	// Search for an active chat room session, or create one if not present
	chatRoom, ok := a.chatRoomDirectory[roomName]
	if !ok {
		chatRoom = &ChatRoom{
			roomId:       roomModel.Id,
			chatSessions: []*ChatSession{},
		}
	}

	// Create a new user session and add it to the chat room
	newChatSession := &ChatSession{
		UserName: userInfo.UserName,
		UserConn: conn,
	}
	chatRoom.chatSessions = append(chatRoom.chatSessions, newChatSession)
	a.chatRoomDirectory[roomName] = chatRoom
	go a.handleChatSession(newChatSession, roomName)
}

func (a *Application) handleChatSession(chatSession *ChatSession, roomName string) {
	defer chatSession.UserConn.Close()
	for {
		clientMessage := &models.ChatMessage{}
		err := chatSession.UserConn.ReadJSON(clientMessage)
		if err != nil {
			// TODO: remove user from pool if disconnected, to prevent sending messages to closed sockets
			log.Println("chatUser.userConn.ReadJSON: ", err)
			break
		}

		if room, ok := a.chatRoomDirectory[roomName]; ok {
			if err := a.repository.InsertChatMessage(room.roomId, clientMessage); err != nil {
				log.Println("Unable to insert chat message: " + err.Error())
			}
		}
		body := []*models.ChatMessage{clientMessage}
		a.broadcastMessage(roomName, chatSession.UserName, body)
	}
}

func (a *Application) broadcastMessage(roomName, sender string, body []*models.ChatMessage) {
	chatRoom, ok := a.chatRoomDirectory[roomName]
	if !ok {
		log.Println("User " + sender + "sent message to an unknown chat room: " + roomName)
		return
	}
	for _, user := range chatRoom.chatSessions {
		// Avoid broadcasting message to sender
		if user.UserName != sender {
			if err := user.UserConn.WriteJSON(&models.WsServerMessage{
				Error: false,
				Body:  body,
			}); err != nil {
				// Ignore errors
				continue
			}
		}
	}
}
