package chat

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"github.com/eshyong/chatapp/chat/models"
	"github.com/eshyong/chatapp/chat/repository"
	"github.com/eshyong/chatapp/chat/service/auth"
	"github.com/eshyong/chatapp/chat/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
)

const (
	// Output directory of react build
	buildDir            = "/frontend/build/"
	defaultErrorMessage = "Sorry, something went wrong. Please try again later"
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

	// Service handling all authentication
	authService *auth.AuthService

	// HTTP routing/security
	router          *mux.Router
	staticFilesPath string

	// Websocket connector
	upgrader *websocket.Upgrader
}

func NewApp(hashKey, blockKey, env string) *Application {
	// TODO: this is ok for now since postgres is local. If I ever use a remote postgres instance, should provision
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

	repo := repository.New(dbConn)
	secureCookie := securecookie.New([]byte(hashKey), []byte(blockKey))

	return &Application{
		authService:       auth.NewAuthenticationService(secureCookie, repo),
		chatRoomDirectory: make(map[string]*ChatRoom),
		staticFilesPath:   filepath.Join(".", buildDir),
		repository:        repo,
		upgrader: &websocket.Upgrader{
			CheckOrigin:     checkOrigin,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
	}
}

func (app *Application) SetupRouter() *mux.Router {
	router := mux.NewRouter()
	router.StrictSlash(true)

	// Serve static files at the NPM build root
	router.PathPrefix("/static/").Handler(http.FileServer(http.Dir(app.staticFilesPath)))

	// Login/register routes
	router.Handle("/login", app.loginHandler()).Methods("POST")
	router.Handle("/register", app.registrationHandler()).Methods("POST")

	// User auth
	router.Handle("/user/current", app.userInfo())

	// API router
	api := router.PathPrefix("/api").Subrouter()
	api.Handle("/chatroom", app.checkAuthentication(app.chatRoomHandler())).Methods("POST")
	// Order matters!
	api.Handle("/chatroom/list", app.checkAuthentication(app.listChatRoomsHandler())).Methods("GET")
	api.Handle("/chatroom/{name}", app.checkAuthentication(app.chatRoomHandler())).Methods("DELETE")
	api.Handle("/chatroom/{name}/join", app.checkAuthentication(app.chatRoomHandler())).Methods("GET")

	// Whitelisted routers to get frontend routing to work
	// Unfortunately, using a wildcard router such as "/{.*}" seems to result in an infinite redirect loop, so we
	// have to specify each route individually here.
	router.Handle("/", app.indexHandler()).Methods("GET")
	router.Handle("/login", app.indexHandler()).Methods("GET")
	router.Handle("/chatroom/{name}", app.indexHandler()).Methods("GET")
	return router
}

func (app *Application) userInfo() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := app.authService.GetUserInfo(r)
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

func (app *Application) checkAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isUserAuthenticated(r) && r.URL.Path != "/login" {
			http.Error(w, "Please login to access the app", http.StatusUnauthorized)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (app *Application) indexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("GET /")
		http.ServeFile(w, r, filepath.Join(app.staticFilesPath, "index.html"))
	})
}

func (app *Application) serveHtmlPage(w http.ResponseWriter, r *http.Request, name string) {
	http.ServeFile(w, r, filepath.Join(app.staticFilesPath, name+".html"))
}

func (app *Application) loginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.loginUser(w, r)
	})
}

func (app *Application) registrationHandler() http.Handler {
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

		if err := app.authService.RegisterUser(registerRequest); err != nil {
			http.Error(w, err.Message, err.Code)
			return
		}

		cookie, err := app.authService.CreateUserSession(registerRequest.UserName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusOK)
	})
}

func (app *Application) chatRoomHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			name := mux.Vars(r)["name"]
			log.Println("GET /chatroom/" + name + "/join")
			app.acceptChatConnection(w, r)
		case "POST":
			log.Println("POST /chatroom")
			app.createChatRoom(w, r)
		case "DELETE":
			log.Println("DELETE /chatroom/{name}")
			app.deleteChatRoom(w, r)
		}
	})
}

func (app *Application) listChatRoomsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatRoomList, err := app.repository.ListChatRooms()
		if err != nil {
			log.Println(err)
			http.Error(w, defaultErrorMessage, http.StatusInternalServerError)
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

func (app *Application) loginUser(w http.ResponseWriter, r *http.Request) {
	log.Println("POST /login")
	loginRequest := &models.LoginRequest{}
	if err := utils.UnmarshalJsonRequest(r, loginRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if loginRequest.UserName == "" || loginRequest.Password == "" {
		http.Error(w, `"username" and "password" fields cannot be empty`, http.StatusBadRequest)
		return
	}

	err := app.authService.LoginUser(loginRequest)
	if err != nil {
		http.Error(w, err.Message, err.Code)
		return
	}

	cookie, err := app.authService.CreateUserSession(loginRequest.UserName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusOK)
}

func (app *Application) isUserAuthenticated(r *http.Request) bool {
	userInfo, err := app.authService.GetUserInfo(r)
	if err != nil {
		return false
	}
	return userInfo.Authenticated
}

func (app *Application) createChatRoom(w http.ResponseWriter, r *http.Request) {
	var createRequest models.CreateChatRoomRequest
	if err := utils.UnmarshalJsonRequest(r, &createRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if createRequest.RoomName == "" || createRequest.CreatedBy == "" {
		http.Error(w, `"roomName" and "createdBy" fields cannot be empty`, http.StatusBadRequest)
		return
	}

	err := app.repository.CreateChatRoom(createRequest.RoomName, createRequest.CreatedBy)
	if err != nil {
		message := defaultErrorMessage
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			message = "A chat room with that name has already been created"
		}
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *Application) deleteChatRoom(w http.ResponseWriter, r *http.Request) {
	roomName := mux.Vars(r)["name"]
	if err := app.repository.DeleteChatRoom(roomName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func (app *Application) acceptChatConnection(w http.ResponseWriter, r *http.Request) {
	// Create websocket connection
	conn, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("User connected from " + conn.RemoteAddr().String())

	// Get user info if possible, and send an error message if not
	userInfo, err := app.authService.GetUserInfo(r)
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
	roomModel, err := app.repository.FindChatRoomByName(roomName)
	if err != nil {
		reason := defaultErrorMessage
		if err == sql.ErrNoRows {
			reason = "Could not find room with that name"
		}
		conn.WriteJSON(&models.WsServerMessage{
			Error:  true,
			Reason: reason,
		})
		conn.Close()
		return
	}
	chatHistory, err := app.repository.GetChatMessagesByRoomId(roomModel.Id)
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
	chatRoom, ok := app.chatRoomDirectory[roomName]
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
	app.chatRoomDirectory[roomName] = chatRoom
	go app.handleChatSession(newChatSession, roomName)
}

func (app *Application) handleChatSession(chatSession *ChatSession, roomName string) {
	defer chatSession.UserConn.Close()
	for {
		clientMessage := &models.ChatMessage{}
		err := chatSession.UserConn.ReadJSON(clientMessage)
		if err != nil {
			// TODO: remove user from pool if disconnected, to prevent sending messages to closed sockets
			log.Println("chatUser.userConn.ReadJSON: ", err)
			break
		}

		if room, ok := app.chatRoomDirectory[roomName]; ok {
			if err := app.repository.InsertChatMessage(room.roomId, clientMessage); err != nil {
				log.Println("Unable to insert chat message: " + err.Error())
			}
		}
		body := []*models.ChatMessage{clientMessage}
		app.broadcastMessage(roomName, chatSession.UserName, body)
	}
}

func (app *Application) broadcastMessage(roomName, sender string, body []*models.ChatMessage) {
	chatRoom, ok := app.chatRoomDirectory[roomName]
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
