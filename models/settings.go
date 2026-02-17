package models

import (
	"database/sql"
	"strconv"

	"github.com/bardenit/Bard/db"
)

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
