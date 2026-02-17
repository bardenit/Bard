package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// AppVersion is the current application version, displayed in the nav.
const AppVersion = "1.2"

// upgradeAvailable is set to 1 by the background checker when a newer commit exists on GitHub.
var upgradeAvailable atomic.Bool

// StartUpgradeChecker launches a background goroutine that periodically checks GitHub
// for newer commits. buildTime must be an RFC3339 timestamp injected at build time;
// if it is "unknown" or empty the checker does nothing (local dev mode).
func StartUpgradeChecker(buildTime string) {
	if buildTime == "" || buildTime == "unknown" {
		return
	}
	go func() {
		// Check immediately, then every hour.
		checkForUpgrade(buildTime)
		for range time.Tick(1 * time.Hour) {
			checkForUpgrade(buildTime)
		}
	}()
}

func checkForUpgrade(buildTime string) {
	built, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/bardenit/Bard/commits/main", nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Upgrade check failed: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	latest, err := time.Parse(time.RFC3339, result.Commit.Committer.Date)
	if err != nil {
		return
	}

	upgradeAvailable.Store(latest.After(built))
}
