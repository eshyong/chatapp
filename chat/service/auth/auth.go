package auth

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/eshyong/chatapp/chat/models"
	"github.com/eshyong/chatapp/chat/repository"
	"github.com/gorilla/securecookie"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieMaxAge = int((time.Hour * 24) / time.Second)
)

type AuthService struct {
	repo         *repository.Repository
	secureCookie *securecookie.SecureCookie
}

type ServiceError struct {
	Code    int
	Message string
}

func (serviceErr *ServiceError) Error() string {
	return fmt.Sprintf("HTTP Code: %s, Message: %s", serviceErr.Code, serviceErr.Message)
}

func NewAuthenticationService(secureCookie *securecookie.SecureCookie, repo *repository.Repository) *AuthService {
	return &AuthService{
		repo:         repo,
		secureCookie: secureCookie,
	}
}

func (service *AuthService) RegisterUser(request *models.RegisterRequest) *ServiceError {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return &ServiceError{
			Code:    http.StatusInternalServerError,
			Message: "bcrypt hash failed",
		}
	}

	err = service.repo.InsertUser(request.UserName, string(hashedPassword))
	if err == nil {
		return nil
	}

	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
		return &ServiceError{
			Code:    http.StatusBadRequest,
			Message: "A user with that name already exists",
		}
	}
	return &ServiceError{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}
}

func (service *AuthService) LoginUser(request *models.LoginRequest) *ServiceError {
	user, err := service.repo.FindUserByName(request.UserName)
	if err != nil {
		if err == sql.ErrNoRows {
			return &ServiceError{
				Code:    http.StatusBadRequest,
				Message: "No user found with that name",
			}
		}
		return &ServiceError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	storedPassword := user.Password
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(request.Password)); err != nil {
		return &ServiceError{
			Code:    http.StatusBadRequest,
			Message: "Password is incorrect",
		}
	}
	return nil
}

func (service *AuthService) CreateUserSession(userName string) (*http.Cookie, *ServiceError) {
	session := map[string]string{
		"userName":      userName,
		"authenticated": "true",
	}
	cookieName := "userSession"
	encoded, err := service.secureCookie.Encode(cookieName, session)
	if err != nil {
		return nil, &ServiceError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   cookieMaxAge,
	}
	return cookie, nil
}

func (service *AuthService) GetUserInfo(r *http.Request) (*models.UserInfo, error) {
	cookieName := "userSession"
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}
	var sessionValues map[string]string
	if err = service.secureCookie.Decode(cookieName, cookie.Value, &sessionValues); err != nil {
		return nil, err
	}

	return &models.UserInfo{
		Authenticated: sessionValues["authenticated"] == "true",
		UserName:      sessionValues["userName"],
	}, nil
}
