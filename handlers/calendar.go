package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bardenit/Bard/models"
)

func CalendarHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Parse optional month parameter: ?month=2026-03
	if m := r.URL.Query().Get("month"); m != "" {
		t, err := time.Parse("2006-01", m)
		if err == nil {
			year = t.Year()
			month = int(t.Month())
		}
	}

	// Also support ?year=2026&month=3
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}
	if m := r.URL.Query().Get("m"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed >= 1 && parsed <= 12 {
			month = parsed
		}
	}

	days, err := models.BuildCalendarMonth(year, time.Month(month))
	if err != nil {
		log.Printf("Error building calendar: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Previous and next month links
	current := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	prev := current.AddDate(0, -1, 0)
	next := current.AddDate(0, 1, 0)

	data := PageData{
		Title:     fmt.Sprintf("Calendar — %s %d", current.Month().String(), year),
		ActiveNav: "calendar",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Days":       days,
		"MonthName":  current.Format("January 2006"),
		"PrevMonth":  prev.Format("2006-01"),
		"NextMonth":  next.Format("2006-01"),
		"TodayMonth": now.Format("2006-01"),
	}

	RenderTemplate(w, "calendar.html", data)
}
