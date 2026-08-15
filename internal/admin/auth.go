package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const adminContextKey contextKey = "admin-session"

type LoginSession struct {
	AdminID  int64
	Username string
	CSRF     string
	Expires  time.Time
}

type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]LoginSession
	secureCookie bool
	lifetime     time.Duration
}

func NewSessionManager(secureCookie bool) *SessionManager {
	return &SessionManager{sessions: make(map[string]LoginSession), secureCookie: secureCookie, lifetime: 12 * time.Hour}
}

func (m *SessionManager) Create(w http.ResponseWriter, admin AdminUser) (LoginSession, error) {
	token, err := randomToken(32)
	if err != nil {
		return LoginSession{}, err
	}
	csrf, err := randomToken(24)
	if err != nil {
		return LoginSession{}, err
	}
	session := LoginSession{AdminID: admin.ID, Username: admin.Username, CSRF: csrf, Expires: time.Now().Add(m.lifetime)}
	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: token, Path: "/admin", HttpOnly: true, Secure: m.secureCookie, SameSite: http.SameSiteStrictMode, Expires: session.Expires, MaxAge: int(m.lifetime.Seconds())})
	return session, nil
}

func (m *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("admin_session"); err == nil {
		m.mu.Lock()
		delete(m.sessions, cookie.Value)
		m.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_session", Value: "", Path: "/admin", HttpOnly: true, Secure: m.secureCookie, SameSite: http.SameSiteStrictMode, Expires: time.Unix(1, 0), MaxAge: -1})
}

func (m *SessionManager) FromRequest(r *http.Request) (LoginSession, bool) {
	cookie, err := r.Cookie("admin_session")
	if err != nil {
		return LoginSession{}, false
	}
	m.mu.RLock()
	session, ok := m.sessions[cookie.Value]
	m.mu.RUnlock()
	if !ok {
		return LoginSession{}, false
	}
	if time.Now().After(session.Expires) {
		m.mu.Lock()
		delete(m.sessions, cookie.Value)
		m.mu.Unlock()
		return LoginSession{}, false
	}
	return session, true
}

func (m *SessionManager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := m.FromRequest(r)
		if !ok {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, maxPrizeUploadSize+(1<<20))
			var err error
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				err = r.ParseMultipartForm(1 << 20)
			} else {
				err = r.ParseForm()
			}
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
			if err != nil || r.Form.Get("csrf_token") != session.CSRF {
				http.Error(w, "Permintaan tidak valid (CSRF)", http.StatusForbidden)
				return
			}
		}
		ctx := context.WithValue(r.Context(), adminContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentAdmin(r *http.Request) LoginSession {
	session, _ := r.Context().Value(adminContextKey).(LoginSession)
	return session
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
