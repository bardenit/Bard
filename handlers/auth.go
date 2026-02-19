package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	oidcpkg "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"github.com/bardenit/Bard/models"
)

// AuthEnabled is true when at least one auth method is configured.
var AuthEnabled bool

// authMu protects authConfig for safe hot-reload via ReloadAuth.
var authMu sync.RWMutex

var authConfig struct {
	username           string
	passwordHash       []byte
	mustChangePassword bool
	oidcEnabled        bool
	oauth2Config       oauth2.Config
	oidcVerifier       *oidcpkg.IDTokenVerifier
	sessionSecret      []byte
	redirectURL        string
	pwEnvOverride      bool // true when AUTH_USERNAME + AUTH_PASSWORD_HASH env vars are set
	oidcEnvOverride    bool // true when all OIDC_* env vars are set
}

type sessionPayload struct {
	U string `json:"u"`           // username or OIDC sub
	E int64  `json:"e"`           // expiry unix timestamp
	T string `json:"t,omitempty"` // token type: "pw" or "oidc"; empty treated as "pw"
}

const sessionCookieName = "session"
const sessionDuration = 7 * 24 * time.Hour

var publicPaths = []string{"/login", "/static/", "/sw.js"}

// InitAuth reads DB + env vars and configures auth. Called once at startup.
func InitAuth() {
	if err := reloadAuthCore(); err != nil {
		log.Fatalf("Auth: init failed: %v", err)
	}
}

// ReloadAuth re-reads DB + env vars and hot-reloads auth config. Safe to call at runtime.
func ReloadAuth() error {
	return reloadAuthCore()
}

// reloadAuthCore does the actual work; holds the write lock for the entire operation.
func reloadAuthCore() error {
	authMu.Lock()
	defer authMu.Unlock()

	// Session secret — set once, preserved across reloads to keep existing sessions valid.
	if len(authConfig.sessionSecret) == 0 {
		secretStr := os.Getenv("SESSION_SECRET")
		if secretStr == "" {
			secret := make([]byte, 32)
			if _, err := rand.Read(secret); err != nil {
				return fmt.Errorf("failed to generate session secret: %w", err)
			}
			authConfig.sessionSecret = secret
			log.Println("Auth: SESSION_SECRET not set — sessions will be lost on restart")
		} else {
			authConfig.sessionSecret = []byte(secretStr)
		}
	}

	// Read DB settings.
	dbUsername, _ := models.GetAuthUsername()
	dbPasswordHash, _ := models.GetAuthPasswordHash()
	dbIssuer, dbClientID, dbClientSecret, dbRedirectURL, dbOIDCEnabled, _ := models.GetOIDCSettings()

	// Read env vars.
	envUsername := os.Getenv("AUTH_USERNAME")
	envPasswordHash := os.Getenv("AUTH_PASSWORD_HASH")
	envIssuer := os.Getenv("OIDC_ISSUER")
	envClientID := os.Getenv("OIDC_CLIENT_ID")
	envClientSecret := os.Getenv("OIDC_CLIENT_SECRET")
	envRedirectURL := os.Getenv("OIDC_REDIRECT_URL")

	// Determine which source wins (env vars take priority).
	pwEnvOverride := envUsername != "" && envPasswordHash != ""
	oidcEnvOverride := envIssuer != "" && envClientID != "" && envClientSecret != "" && envRedirectURL != ""

	authConfig.pwEnvOverride = pwEnvOverride
	authConfig.oidcEnvOverride = oidcEnvOverride

	// Resolve active password credentials.
	var username, passwordHashStr string
	if pwEnvOverride {
		username = envUsername
		passwordHashStr = envPasswordHash
	} else {
		username = dbUsername
		passwordHashStr = dbPasswordHash
	}

	// Resolve active OIDC settings.
	var issuer, clientID, clientSecret, redirectURL string
	var oidcEnabled bool
	if oidcEnvOverride {
		issuer = envIssuer
		clientID = envClientID
		clientSecret = envClientSecret
		redirectURL = envRedirectURL
		oidcEnabled = true
	} else {
		issuer = dbIssuer
		clientID = dbClientID
		clientSecret = dbClientSecret
		redirectURL = dbRedirectURL
		oidcEnabled = dbOIDCEnabled
	}

	passwordEnabled := username != "" && passwordHashStr != ""

	if !passwordEnabled && !oidcEnabled {
		AuthEnabled = false
		authConfig.username = ""
		authConfig.passwordHash = nil
		authConfig.mustChangePassword = false
		authConfig.oidcEnabled = false
		authConfig.oidcVerifier = nil
		log.Println("Auth: disabled")
		return nil
	}

	AuthEnabled = true

	// Configure password auth.
	if passwordEnabled {
		authConfig.username = username
		authConfig.passwordHash = []byte(passwordHashStr)
		if pwEnvOverride {
			// Env-var accounts don't use the DB must-change flag.
			authConfig.mustChangePassword = false
		} else {
			mustChange, _ := models.GetMustChangePassword()
			authConfig.mustChangePassword = mustChange
		}
		log.Println("Auth: password login enabled")
	} else {
		authConfig.username = ""
		authConfig.passwordHash = nil
		authConfig.mustChangePassword = false
	}

	// Configure OIDC.
	if oidcEnabled && issuer != "" && clientID != "" && clientSecret != "" && redirectURL != "" {
		authConfig.redirectURL = redirectURL
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		provider, err := oidcpkg.NewProvider(ctx, issuer)
		if err != nil {
			authConfig.oidcEnabled = false
			authConfig.oidcVerifier = nil
			return fmt.Errorf("OIDC provider init failed: %w", err)
		}
		authConfig.oidcEnabled = true
		authConfig.oauth2Config = oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidcpkg.ScopeOpenID, "profile", "email"},
		}
		authConfig.oidcVerifier = provider.Verifier(&oidcpkg.Config{ClientID: clientID})
		log.Printf("Auth: OIDC enabled — register this redirect URI in Pocket ID: %s", redirectURL)
	} else {
		authConfig.oidcEnabled = false
		authConfig.oidcVerifier = nil
	}

	return nil
}

// signedCookieValue creates a signed cookie value: base64url(json) + "." + base64url(hmac).
// sessionSecret is write-once at startup, so no lock is needed here.
func signedCookieValue(payload sessionPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, authConfig.sessionSecret)
	mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig, nil
}

// parseSessionCookie validates and parses the session cookie. Returns nil if invalid or expired.
// sessionSecret is write-once at startup, so no lock is needed here.
func parseSessionCookie(r *http.Request) *sessionPayload {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	encoded, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, authConfig.sessionSecret)
	mac.Write([]byte(encoded))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}
	var payload sessionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}
	if time.Now().Unix() > payload.E {
		return nil
	}
	return &payload
}

func setSessionCookie(w http.ResponseWriter, username string, tokenType string) error {
	payload := sessionPayload{
		U: username,
		E: time.Now().Add(sessionDuration).Unix(),
		T: tokenType,
	}
	value, err := signedCookieValue(payload)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// AuthMiddleware wraps the mux and enforces authentication.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !AuthEnabled {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		for _, pub := range publicPaths {
			if path == pub || strings.HasPrefix(path, pub) {
				next.ServeHTTP(w, r)
				return
			}
		}

		session := parseSessionCookie(r)
		if session == nil {
			// Invalid/missing session — clear bad cookie and redirect to login.
			clearSessionCookie(w)
			nextURL := r.URL.RequestURI()
			if !strings.HasPrefix(nextURL, "/") || strings.HasPrefix(nextURL, "//") {
				nextURL = "/"
			}
			http.Redirect(w, r, "/login?next="+url.QueryEscape(nextURL), http.StatusFound)
			return
		}

		// Force-password-change gate.
		authMu.RLock()
		localUser := authConfig.username
		mustChange := authConfig.mustChangePassword
		authMu.RUnlock()

		isLocalPWSession := (session.T == "pw" || session.T == "") && session.U == localUser
		if mustChange && isLocalPWSession {
			// Allow the password-change page and logout through.
			allowed := path == "/logout" ||
				path == "/settings" ||
				strings.HasPrefix(path, "/settings/password")
			if !allowed {
				http.Redirect(w, r, "/settings", http.StatusFound)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

type loginPageData struct {
	Error           string
	Next            string
	OIDCEnabled     bool
	PasswordEnabled bool
	OIDCRedirectURL string
}

// LoginPageHandler handles GET /login.
func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if !AuthEnabled {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if parseSessionCookie(r) != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	nextURL := r.URL.Query().Get("next")
	if !strings.HasPrefix(nextURL, "/") || strings.HasPrefix(nextURL, "//") {
		nextURL = "/"
	}
	authMu.RLock()
	oidcEnabled := authConfig.oidcEnabled
	hasUsername := authConfig.username != ""
	redirectURL := authConfig.redirectURL
	authMu.RUnlock()
	renderLoginPage(w, loginPageData{
		Next:            nextURL,
		OIDCEnabled:     oidcEnabled,
		PasswordEnabled: hasUsername,
		OIDCRedirectURL: redirectURL,
	})
}

func renderLoginPage(w http.ResponseWriter, data loginPageData) {
	t, ok := pageCache["login.html"]
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "login.html", data); err != nil {
		log.Printf("Login template error: %v", err)
	}
}

// LoginPostHandler handles POST /login.
func LoginPostHandler(w http.ResponseWriter, r *http.Request) {
	authMu.RLock()
	enabled := AuthEnabled
	hasUsername := authConfig.username != ""
	oidcEnabled := authConfig.oidcEnabled
	redirectURL := authConfig.redirectURL
	username := authConfig.username
	pwHash := make([]byte, len(authConfig.passwordHash))
	copy(pwHash, authConfig.passwordHash)
	authMu.RUnlock()

	if !enabled || !hasUsername {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	r.ParseForm()
	formUsername := r.FormValue("username")
	password := r.FormValue("password")
	nextURL := r.FormValue("next")
	if !strings.HasPrefix(nextURL, "/") || strings.HasPrefix(nextURL, "//") {
		nextURL = "/"
	}

	// Always call bcrypt.CompareHashAndPassword to mitigate timing attacks.
	err := bcrypt.CompareHashAndPassword(pwHash, []byte(password))
	if formUsername != username || err != nil {
		renderLoginPage(w, loginPageData{
			Error:           "Invalid username or password.",
			Next:            nextURL,
			OIDCEnabled:     oidcEnabled,
			PasswordEnabled: true,
			OIDCRedirectURL: redirectURL,
		})
		return
	}

	if err := setSessionCookie(w, username, "pw"); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, nextURL, http.StatusFound)
}

// LogoutHandler handles GET /logout.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// OIDCLoginHandler handles GET /login/oidc.
func OIDCLoginHandler(w http.ResponseWriter, r *http.Request) {
	authMu.RLock()
	enabled := AuthEnabled
	oidcEnabled := authConfig.oidcEnabled
	oauth2Cfg := authConfig.oauth2Config
	secret := authConfig.sessionSecret
	authMu.RUnlock()

	if !enabled || !oidcEnabled {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	nextURL := r.URL.Query().Get("next")
	if !strings.HasPrefix(nextURL, "/") || strings.HasPrefix(nextURL, "//") {
		nextURL = "/"
	}

	// Generate random state bytes.
	stateBytes := make([]byte, 24)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	rawState := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Sign rawState for CSRF protection.
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(rawState))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Cookie stores rawState + "." + sig.
	cookieValue := rawState + "." + sig

	// OAuth2 state parameter embeds next URL: rawState + ":" + base64url(next).
	nextEncoded := base64.RawURLEncoding.EncodeToString([]byte(nextURL))
	oauthState := rawState + ":" + nextEncoded

	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_state",
		Value:    cookieValue,
		Path:     "/login/oidc/callback",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, oauth2Cfg.AuthCodeURL(oauthState), http.StatusFound)
}

// OIDCCallbackHandler handles GET /login/oidc/callback.
func OIDCCallbackHandler(w http.ResponseWriter, r *http.Request) {
	authMu.RLock()
	enabled := AuthEnabled
	oidcEnabled := authConfig.oidcEnabled
	oauth2Cfg := authConfig.oauth2Config
	verifier := authConfig.oidcVerifier
	secret := authConfig.sessionSecret
	authMu.RUnlock()

	if !enabled || !oidcEnabled {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	oauthState := r.URL.Query().Get("state")
	if oauthState == "" {
		http.Error(w, "Missing state", http.StatusBadRequest)
		return
	}

	// Split state into rawState and nextEncoded.
	stateParts := strings.SplitN(oauthState, ":", 2)
	rawState := stateParts[0]
	nextURL := "/"
	if len(stateParts) == 2 {
		if decoded, err := base64.RawURLEncoding.DecodeString(stateParts[1]); err == nil {
			nextURL = string(decoded)
		}
	}
	if !strings.HasPrefix(nextURL, "/") || strings.HasPrefix(nextURL, "//") {
		nextURL = "/"
	}

	// Validate state cookie.
	stateCookie, err := r.Cookie("oidc_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}
	cookieParts := strings.SplitN(stateCookie.Value, ".", 2)
	if len(cookieParts) != 2 {
		http.Error(w, "Invalid state cookie", http.StatusBadRequest)
		return
	}
	cookieRawState, cookieSig := cookieParts[0], cookieParts[1]

	// Verify HMAC signature.
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(cookieRawState))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(cookieSig), []byte(expectedSig)) {
		http.Error(w, "Invalid state signature", http.StatusBadRequest)
		return
	}

	// Verify rawState matches cookie.
	if rawState != cookieRawState {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Clear state cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_state",
		Value:    "",
		Path:     "/login/oidc/callback",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Exchange code for token.
	ctx := context.Background()
	token, err := oauth2Cfg.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("OIDC: token exchange failed: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Extract and verify ID token.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token in response", http.StatusInternalServerError)
		return
	}
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("OIDC: ID token verification failed: %v", err)
		http.Error(w, "Token verification failed", http.StatusInternalServerError)
		return
	}

	if err := setSessionCookie(w, idToken.Subject, "oidc"); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, nextURL, http.StatusFound)
}
