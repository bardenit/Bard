package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bardenit/Bard/models"
)

func IncomeRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/income":
		if r.Method == http.MethodGet {
			IncomeHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	case path == "/income/create" && r.Method == http.MethodPost:
		IncomeCreateHandler(w, r)
	case strings.HasSuffix(path, "/edit"):
		IncomeEditHandler(w, r)
	case strings.HasSuffix(path, "/update"):
		IncomeUpdateHandler(w, r)
	case strings.HasSuffix(path, "/delete"):
		IncomeDeleteHandler(w, r)
	case strings.HasSuffix(path, "/set-primary"):
		IncomeSetPrimaryHandler(w, r)
	case strings.HasSuffix(path, "/record"):
		if r.Method == http.MethodGet {
			IncomeRecordFormHandler(w, r)
		} else if r.Method == http.MethodPost {
			IncomeRecordHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func IncomeHandler(w http.ResponseWriter, r *http.Request) {
	items, err := models.ListIncome()
	if err != nil {
		log.Printf("Error listing income: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:     "Income",
		ActiveNav: "income",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Income": items,
	}
	RenderTemplate(w, "income.html", data)
}

func IncomeCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/income", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	amountStr := r.FormValue("amount")
	depositDate := r.FormValue("deposit_date")
	recurrence := r.FormValue("recurrence")

	amount, err := parseCents(amountStr)
	if err != nil || name == "" || depositDate == "" {
		SetFlash(w, "Please fill in all fields with valid values.")
		http.Redirect(w, r, "/income", http.StatusSeeOther)
		return
	}

	if err := models.CreateIncome(name, amount, depositDate, recurrence); err != nil {
		log.Printf("Error creating income: %v", err)
		SetFlash(w, "Failed to create income entry.")
		http.Redirect(w, r, "/income", http.StatusSeeOther)
		return
	}

	SetFlash(w, "Income entry created successfully.")
	http.Redirect(w, r, "/income", http.StatusSeeOther)
}

func IncomeEditHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/income/")
	idStr = strings.TrimSuffix(idStr, "/edit")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	item, err := models.GetIncome(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := PageData{
		Title:     "Edit Income",
		ActiveNav: "income",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"IncomeItem": item,
		"Editing":    true,
	}
	RenderTemplate(w, "income.html", data)
}

func IncomeUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/income", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/income/")
	idStr = strings.TrimSuffix(idStr, "/update")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	amountStr := r.FormValue("amount")
	depositDate := r.FormValue("deposit_date")
	recurrence := r.FormValue("recurrence")
	isActive := r.FormValue("is_active") == "on"

	amount, err := parseCents(amountStr)
	if err != nil || name == "" || depositDate == "" {
		SetFlash(w, "Please fill in all fields with valid values.")
		http.Redirect(w, r, "/income/"+strconv.Itoa(id)+"/edit", http.StatusSeeOther)
		return
	}

	if err := models.UpdateIncome(id, name, amount, depositDate, recurrence, isActive); err != nil {
		log.Printf("Error updating income: %v", err)
		SetFlash(w, "Failed to update income entry.")
	} else {
		SetFlash(w, "Income entry updated successfully.")
	}

	http.Redirect(w, r, "/income", http.StatusSeeOther)
}

func IncomeSetPrimaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/income", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/income/")
	idStr = strings.TrimSuffix(idStr, "/set-primary")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.SetPrimaryIncome(id); err != nil {
		log.Printf("SetPrimaryIncome error: %v", err)
		SetFlash(w, "Failed to set primary income.")
	} else {
		SetFlash(w, "Primary income source updated.")
	}
	http.Redirect(w, r, "/income", http.StatusSeeOther)
}

func IncomeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/income", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/income/")
	idStr = strings.TrimSuffix(idStr, "/delete")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.DeleteIncome(id); err != nil {
		log.Printf("Error deleting income: %v", err)
		SetFlash(w, "Failed to delete income entry.")
	} else {
		SetFlash(w, "Income entry deleted.")
	}

	http.Redirect(w, r, "/income", http.StatusSeeOther)
}

// IncomeRecordFormHandler shows the form to record an actual amount received for an income occurrence.
func IncomeRecordFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/income/")
	idStr = strings.TrimSuffix(idStr, "/record")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	item, err := models.GetIncome(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	occurrenceDate := r.URL.Query().Get("date")
	returnMonth := r.URL.Query().Get("return")
	if returnMonth == "" && occurrenceDate != "" && len(occurrenceDate) >= 7 {
		returnMonth = occurrenceDate[:7]
	}

	existing, hasActual, _ := models.GetIncomeActual(id, occurrenceDate)

	data := PageData{
		Title:     "Record Income — " + item.Name,
		ActiveNav: "income",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Income":         item,
		"OccurrenceDate": occurrenceDate,
		"ReturnMonth":    returnMonth,
		"HasActual":      hasActual,
		"PriorActual":    existing,
	}
	RenderTemplate(w, "income_record.html", data)
}

// IncomeRecordHandler saves the actual amount received for an income occurrence.
func IncomeRecordHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/income/")
	idStr = strings.TrimSuffix(idStr, "/record")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	occurrenceDate := r.FormValue("occurrence_date")
	returnMonth := r.FormValue("return_month")
	amountStr := r.FormValue("amount_actual")
	notes := strings.TrimSpace(r.FormValue("notes"))

	amountActual, err := parseCents(amountStr)
	if err != nil || occurrenceDate == "" {
		SetFlash(w, "Please enter a valid amount.")
		redirectTo := "/income/" + strconv.Itoa(id) + "/record?date=" + occurrenceDate + "&return=" + returnMonth
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
		return
	}

	if err := models.RecordIncomeActual(id, occurrenceDate, amountActual, notes); err != nil {
		log.Printf("Error recording income actual: %v", err)
		SetFlash(w, "Failed to record income.")
	} else {
		SetFlash(w, "Income recorded.")
	}

	if returnMonth != "" {
		http.Redirect(w, r, "/calendar?month="+returnMonth, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/calendar", http.StatusSeeOther)
	}
}
