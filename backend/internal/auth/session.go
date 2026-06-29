package auth

import (
	"errors"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

const (
	SessionName = "isomap_session"
	UserIDKey   = "user_id"
)

var ErrUnauthenticated = errors.New("user is not authenticated")

func GetSession(r *http.Request) (*sessions.Session, error) {
	return cookieStore().Get(r, SessionName)
}

func SetUserID(w http.ResponseWriter, r *http.Request, userID uuid.UUID) error {
	session, err := GetSession(r)
	if err != nil {
		return err
	}

	session.Values[UserIDKey] = userID.String()
	return session.Save(r, w)
}

func Clear(w http.ResponseWriter, r *http.Request) error {
	session, err := GetSession(r)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func CurrentUserID(r *http.Request) (uuid.UUID, error) {
	session, err := GetSession(r)
	if err != nil {
		return uuid.Nil, err
	}

	rawUserID, ok := session.Values[UserIDKey].(string)
	if !ok || rawUserID == "" {
		return uuid.Nil, ErrUnauthenticated
	}

	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return uuid.Nil, ErrUnauthenticated
	}

	return userID, nil
}

func cookieStore() *sessions.CookieStore {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "development-only-change-me-before-production"
	}

	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
		SameSite: sameSite(),
		Secure:   isProduction(),
	}
	return store
}

func sameSite() http.SameSite {
	if isProduction() {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func isProduction() bool {
	return os.Getenv("APP_ENV") == "production" || os.Getenv("GO_ENV") == "production"
}
