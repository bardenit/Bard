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
	activeUsername := authConfig.username
	activeOIDCIssuer := authConfig.oidcIssuer
	activeOIDCClientID := authConfig.oidcClientID
	activeOIDCRedirectURL := authConfig.redirectURL
	activeOIDCEnabled := authConfig.oidcEnabled
	authMu.RUnlock()

	// Read DB values for the secret (we never expose env-var secrets in the UI).
	dbIssuer, dbClientID, dbClientSecret, dbRedirectURL, dbOIDCEnabled, _ := models.GetOIDCSettings()
	dbUsername, _ := models.GetAuthUsername()

	// Resolve display values: active (effective) for non-sensitive fields;
	// DB for sensitive (secret) or for reference when no env override.
	displayUsername := activeUsername
	if !pwEnvOverride {
		// When not overridden, show DB value (same as active, but kept up-to-date by SettingsPasswordHandler).
		displayUsername = dbUsername
	}

	var displayIssuer, displayClientID, displayRedirectURL string
	var displayOIDCEnabled bool
	var oidcSecretSet bool

	if oidcEnvOverride {
		// Show the active (env-var) values for non-sensitive fields.
		displayIssuer = activeOIDCIssuer
		displayClientID = activeOIDCClientID
		displayRedirectURL = activeOIDCRedirectURL
		displayOIDCEnabled = activeOIDCEnabled
		oidcSecretSet = false // don't reveal whether env var secret is set
	} else {
		// Show DB values (editable).
		displayIssuer = dbIssuer
		displayClientID = dbClientID
		displayRedirectURL = dbRedirectURL
		displayOIDCEnabled = dbOIDCEnabled
		oidcSecretSet = dbClientSecret != ""
	}

	RenderTemplate(w, "settings.html", PageData{
		Title:     "Settings",
		ActiveNav: "settings",
		Flash:     GetFlash(w, r),
		Extra: map[string]interface{}{
			"Username":        displayUsername,
			"PWEnvOverride":   pwEnvOverride,
			"MustChange":      mustChange,
			"OIDCEnabled":     displayOIDCEnabled,
			"OIDCIssuer":      displayIssuer,
			"OIDCClientID":    displayClientID,
			"OIDCSecretSet":   oidcSecretSet,
			"OIDCRedirectURL": displayRedirectURL,
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

	// Update live authConfig if env vars are not overriding it.
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

	// If secret left blank, preserve the existing DB value.
	if clientSecret == "" {
		_, _, existing, _, _, _ := models.GetOIDCSettings()
		clientSecret = existing
	}

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
