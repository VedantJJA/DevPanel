package docker

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/VedantJJA/devpnl/internal/db"
)

// Session token is stored in the database for persistence

// Helper to hash passwords using SHA-256 with a simple salt.
// We use a fixed salt for simplicity since this is a single-admin system,
// but in a multi-tenant system bcrypt/argon2 should be used.
const fixedSalt = "devpanel_salt_v1_8237"

func hashPassword(password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + fixedSalt))
	return hex.EncodeToString(hasher.Sum(nil))
}

func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "devpanel_session",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// AuthMiddleware wraps an http.Handler and enforces authentication
func AuthMiddleware(database *db.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("devpanel_session")
		dbSession, _ := database.GetSetting(r.Context(), "admin_session_token")
		if err != nil || cookie.Value == "" || dbSession == "" || cookie.Value != dbSession {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	}
}

// HandleAuthStatus returns the current auth state (needs_setup, authenticated)
func HandleAuthStatus(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash, _ := database.GetSetting(r.Context(), "admin_password_hash")
		dbSession, _ := database.GetSetting(r.Context(), "admin_session_token")
		
		authenticated := false
		if cookie, err := r.Cookie("devpanel_session"); err == nil && cookie.Value != "" && dbSession != "" && cookie.Value == dbSession {
			authenticated = true
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"needs_setup":   hash == "",
			"authenticated": authenticated,
		})
	}
}

// HandleAuthSetup initializes the admin password if not set
func HandleAuthSetup(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		existingHash, _ := database.GetSetting(r.Context(), "admin_password_hash")
		if existingHash != "" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Setup already completed"})
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Password is required"})
			return
		}

		hash := hashPassword(req.Password)
		if err := database.SetSetting(r.Context(), "admin_password_hash", hash); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to save password"})
			return
		}

		// Auto login
		sessionToken := generateSessionToken()
		_ = database.SetSetting(r.Context(), "admin_session_token", sessionToken)
		setSessionCookie(w, sessionToken)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// HandleAuthLogin verifies password and issues a session cookie
func HandleAuthLogin(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash, err := database.GetSetting(r.Context(), "admin_password_hash")
		if err != nil || hash == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "System not set up yet"})
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request"})
			return
		}

		expectedHash := hashPassword(req.Password)
		if subtle.ConstantTimeCompare([]byte(expectedHash), []byte(hash)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid password"})
			return
		}

		sessionToken := generateSessionToken()
		_ = database.SetSetting(r.Context(), "admin_session_token", sessionToken)
		setSessionCookie(w, sessionToken)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// HandleAuthLogout clears the session cookie
func HandleAuthLogout(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = database.SetSetting(r.Context(), "admin_session_token", "")
		http.SetCookie(w, &http.Cookie{
			Name:     "devpanel_session",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
