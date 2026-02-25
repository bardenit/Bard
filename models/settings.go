package models

import (
	"database/sql"
	"strconv"
	"time"

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

// GetManualBalanceWithDate returns the stored balance, the date it was set, and whether it exists.
func GetManualBalanceWithDate() (cents int, asOf time.Time, hasBalance bool, err error) {
	val, err := GetSetting("account_balance")
	if err != nil || val == "" {
		return 0, time.Time{}, false, err
	}
	cents, err = strconv.Atoi(val)
	if err != nil {
		return 0, time.Time{}, false, nil
	}
	dateStr, _ := GetSetting("account_balance_date")
	if dateStr != "" {
		asOf, _ = time.Parse("2006-01-02", dateStr)
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}
	return cents, asOf, true, nil
}

// SetManualBalance stores the account balance in cents and records today as the balance date.
func SetManualBalance(cents int) error {
	if err := SetSetting("account_balance", strconv.Itoa(cents)); err != nil {
		return err
	}
	return SetSetting("account_balance_date", time.Now().Format("2006-01-02"))
}

// GetProjectedBalance computes the current estimated balance by applying all scheduled
// bill and income events that occurred between when the balance was set and today.
func GetProjectedBalance() (int, bool, string, error) {
	storedCents, asOf, hasBalance, err := GetManualBalanceWithDate()
	if err != nil || !hasBalance {
		return 0, false, "", err
	}

	today := time.Now()
	asOfDay := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.Local)
	todayDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	asOfLabel := asOf.Format("Jan 2")

	// No time has passed since balance was set
	if !asOfDay.Before(todayDay) {
		return storedCents, true, asOfLabel, nil
	}

	// Range: day after balance was set through today
	rangeStart := asOfDay.AddDate(0, 0, 1)
	rangeEnd := todayDay

	bills, err := ListActiveBills()
	if err != nil {
		return storedCents, true, asOfLabel, err
	}

	billDeductions := 0
	for _, bill := range bills {
		anchor, err := time.Parse("2006-01-02", bill.DueDate)
		if err != nil {
			continue
		}
		occurrences := ExpandOccurrences(anchor, bill.Recurrence, rangeStart, rangeEnd)
		billDeductions += bill.Amount * len(occurrences)
	}

	incomeItems, err := ListActiveIncome()
	if err != nil {
		return storedCents - billDeductions, true, asOfLabel, err
	}

	incomeAdditions := 0
	for _, item := range incomeItems {
		anchor, err := time.Parse("2006-01-02", item.DepositDate)
		if err != nil {
			continue
		}
		occurrences := ExpandOccurrences(anchor, item.Recurrence, rangeStart, rangeEnd)
		incomeAdditions += item.Amount * len(occurrences)
	}

	projected := storedCents - billDeductions + incomeAdditions
	return projected, true, asOfLabel, nil
}
