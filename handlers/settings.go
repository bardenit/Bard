package handlers

import (
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/bardenit/Bard/models"
)

// SettingsPageHandler handles GET /settings.
func SettingsPageHandler(w http.ResponseWriter, r *http.Request) {
	authMu.RLock()
	pwEnvOverride := authConfig.pwEnvOverride
	oidcEnvOverride := authConfig.oidcEnvOverride
	mustChange := authConfig.mustChangePassword
	authMu.RUnlock()

	// Always read DB values for display (env var notice will indicate if they're overridden).
	dbUsername, _ := models.GetAuthUsername()
	dbIssuer, dbClientID, dbClientSecret, dbRedirectURL, dbOIDCEnabled, _ := models.GetOIDCSettings()

	RenderTemplate(w, "settings.html", PageData{
		Title:     "Settings",
		ActiveNav: "settings",
		Flash:     GetFlash(w, r),
		Extra: map[string]interface{}{
			"Username":        dbUsername,
			"PWEnvOverride":   pwEnvOverride,
			"MustChange":      mustChange,
			"OIDCEnabled":     dbOIDCEnabled,
			"OIDCIssuer":      dbIssuer,
			"OIDCClientID":    dbClientID,
			"OIDCClientSecret": dbClientSecret,
			"OIDCRedirectURL": dbRedirectURL,
			"OIDCEnvOverride": oidcEnvOverride,
		},
	})
}

// SettingsPasswordHandler handles POST /settings/password.
func SettingsPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	newUsername := strings.TrimSpace(r.FormValue("username"))
	newPassword := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if newUsername == "" {
		SetFlash(w, "Username cannot be empty.")
		http.Redirect(w, r, "/settings", http.StatusFound)
		return
	}
	if len(newPassword) < 8 {
		SetFlash(w, "Password must be at least 8 characters.")
		http.Redirect(w, r, "/settings", http.StatusFound)
		return
	}
	if newPassword != confirmPassword {
		SetFlash(w, "Passwords do not match.")
		http.Redirect(w, r, "/settings", http.StatusFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Settings: bcrypt failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := models.SetSetting("auth_username", newUsername); err != nil {
		log.Printf("Settings: save username failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := models.SetSetting("auth_password_hash", string(hash)); err != nil {
		log.Printf("Settings: save password hash failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := models.SetMustChangePassword(false); err != nil {
		log.Printf("Settings: save must_change_password failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update live authConfig if not overridden by env vars.
	authMu.RLock()
	pwEnvOverride := authConfig.pwEnvOverride
	authMu.RUnlock()
	if !pwEnvOverride {
		authMu.Lock()
		authConfig.username = newUsername
		authConfig.passwordHash = hash
		authConfig.mustChangePassword = false
		AuthEnabled = true
		authMu.Unlock()
	}

	// Clear session → force re-login with new credentials.
	clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// SettingsOIDCHandler handles POST /settings/oidc.
func SettingsOIDCHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	enabled := r.FormValue("oidc_enabled") == "1"
	issuer := strings.TrimSpace(r.FormValue("oidc_issuer"))
	clientID := strings.TrimSpace(r.FormValue("oidc_client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("oidc_client_secret"))
	redirectURL := strings.TrimSpace(r.FormValue("oidc_redirect_url"))

	if err := models.SetOIDCSettings(issuer, clientID, clientSecret, redirectURL, enabled); err != nil {
		log.Printf("Settings: save OIDC failed: %v", err)
		SetFlash(w, "Failed to save OIDC settings.")
		http.Redirect(w, r, "/settings", http.StatusFound)
		return
	}

	// Hot-reload auth config so OIDC takes effect immediately.
	if err := ReloadAuth(); err != nil {
		SetFlash(w, "OIDC settings saved, but provider init failed: "+err.Error())
	} else {
		SetFlash(w, "Settings saved.")
	}
	http.Redirect(w, r, "/settings", http.StatusFound)
}
