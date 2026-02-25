package handlers

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// AppVersion is set at startup from the embedded VERSION file in main.go.
// Do not edit this manually — update the VERSION file instead.
var AppVersion = "dev"

// latestVersion holds the newest version string fetched from GitHub.
// Empty means no upgrade is available (or check hasn't run yet).
var latestVersion atomic.Value // stores string

// LatestVersion returns the latest available version if newer than AppVersion, else "".
func LatestVersion() string {
	v, _ := latestVersion.Load().(string)
	return v
}

// StartUpgradeChecker launches a background goroutine that checks GitHub every 5 minutes
// for a newer version. buildTime must be a non-empty, non-"unknown" value to activate.
func StartUpgradeChecker(buildTime string) {
	if buildTime == "" || buildTime == "unknown" {
		return
	}
	go func() {
		checkForUpgrade()
		for range time.Tick(5 * time.Minute) {
			checkForUpgrade()
		}
	}()
}

func checkForUpgrade() {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/bardenit/Bard/main/VERSION")
	if err != nil {
		log.Printf("Upgrade check failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	fetched := strings.TrimSpace(string(body))
	if fetched == "" {
		return
	}

	if fetched != AppVersion {
		latestVersion.Store(fetched)
	} else {
		latestVersion.Store("")
	}
}
