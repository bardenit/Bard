package models

import (
	"database/sql"
	"strconv"

	"github.com/bardenit/Bard/db"
)

// GetAuthUsername returns the locally-stored auth username.
func GetAuthUsername() (string, error) {
	return GetSetting("auth_username")
}

// GetAuthPasswordHash returns the locally-stored bcrypt password hash.
func GetAuthPasswordHash() (string, error) {
	return GetSetting("auth_password_hash")
}

// GetMustChangePassword returns true if the user must change their password on next login.
func GetMustChangePassword() (bool, error) {
	v, err := GetSetting("auth_must_change_password")
	return v == "1", err
}

// SetMustChangePassword updates the must-change-password flag in the DB.
func SetMustChangePassword(must bool) error {
	v := "0"
	if must {
		v = "1"
	}
	return SetSetting("auth_must_change_password", v)
}

// GetOIDCSettings returns all OIDC-related settings from the DB.
func GetOIDCSettings() (issuer, clientID, clientSecret, redirectURL string, enabled bool, err error) {
	issuer, err = GetSetting("oidc_issuer")
	if err != nil {
		return
	}
	clientID, err = GetSetting("oidc_client_id")
	if err != nil {
		return
	}
	clientSecret, err = GetSetting("oidc_client_secret")
	if err != nil {
		return
	}
	redirectURL, err = GetSetting("oidc_redirect_url")
	if err != nil {
		return
	}
	var enabledStr string
	enabledStr, err = GetSetting("oidc_enabled")
	enabled = enabledStr == "1"
	return
}

// SetOIDCSettings persists all OIDC-related settings to the DB.
func SetOIDCSettings(issuer, clientID, clientSecret, redirectURL string, enabled bool) error {
	enabledVal := "0"
	if enabled {
		enabledVal = "1"
	}
	pairs := map[string]string{
		"oidc_issuer":        issuer,
		"oidc_client_id":     clientID,
		"oidc_client_secret": clientSecret,
		"oidc_redirect_url":  redirectURL,
		"oidc_enabled":       enabledVal,
	}
	for k, v := range pairs {
		if err := SetSetting(k, v); err != nil {
			return err
		}
	}
	return nil
}

// GetSetting returns a setting value by key, or "" if not set.
func GetSetting(key string) (string, error) {
	var value string
	err := db.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting inserts or replaces a setting value.
func SetSetting(key, value string) error {
	_, err := db.DB.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetManualBalance returns the manually set account balance in cents.
// Returns 0, false if no balance has been set yet.
func GetManualBalance() (int, bool, error) {
	val, err := GetSetting("account_balance")
	if err != nil || val == "" {
		return 0, false, err
	}
	cents, err := strconv.Atoi(val)
	if err != nil {
		return 0, false, nil
	}
	return cents, true, nil
}

// SetManualBalance stores the account balance in cents.
func SetManualBalance(cents int) error {
	return SetSetting("account_balance", strconv.Itoa(cents))
}
