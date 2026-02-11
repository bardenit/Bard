package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/bardenit/Bard/models"
)

func BillsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/bills":
		if r.Method == http.MethodGet {
			BillsHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	case path == "/bills/create" && r.Method == http.MethodPost:
		BillCreateHandler(w, r)
	case strings.HasSuffix(path, "/edit"):
		BillEditHandler(w, r)
	case strings.HasSuffix(path, "/update"):
		BillUpdateHandler(w, r)
	case strings.HasSuffix(path, "/delete"):
		BillDeleteHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func BillsHandler(w http.ResponseWriter, r *http.Request) {
	bills, err := models.ListBills()
	if err != nil {
		log.Printf("Error listing bills: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:     "Bills",
		ActiveNav: "bills",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Bills": bills,
	}
	RenderTemplate(w, "bills.html", data)
}

func BillCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/bills", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	amountStr := r.FormValue("amount")
	dueDate := r.FormValue("due_date")
	recurrence := r.FormValue("recurrence")

	amount, err := parseCents(amountStr)
	if err != nil || name == "" || dueDate == "" {
		SetFlash(w, "Please fill in all fields with valid values.")
		http.Redirect(w, r, "/bills", http.StatusSeeOther)
		return
	}

	if err := models.CreateBill(name, amount, dueDate, recurrence); err != nil {
		log.Printf("Error creating bill: %v", err)
		SetFlash(w, "Failed to create bill.")
		http.Redirect(w, r, "/bills", http.StatusSeeOther)
		return
	}

	SetFlash(w, "Bill created successfully.")
	http.Redirect(w, r, "/bills", http.StatusSeeOther)
}

func BillEditHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/bills/")
	idStr = strings.TrimSuffix(idStr, "/edit")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	bill, err := models.GetBill(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := PageData{
		Title:     "Edit Bill",
		ActiveNav: "bills",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Bill":    bill,
		"Editing": true,
	}
	RenderTemplate(w, "bills.html", data)
}

func BillUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/bills", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/bills/")
	idStr = strings.TrimSuffix(idStr, "/update")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	amountStr := r.FormValue("amount")
	dueDate := r.FormValue("due_date")
	recurrence := r.FormValue("recurrence")
	isActive := r.FormValue("is_active") == "on"

	amount, err := parseCents(amountStr)
	if err != nil || name == "" || dueDate == "" {
		SetFlash(w, "Please fill in all fields with valid values.")
		http.Redirect(w, r, "/bills/"+strconv.Itoa(id)+"/edit", http.StatusSeeOther)
		return
	}

	if err := models.UpdateBill(id, name, amount, dueDate, recurrence, isActive); err != nil {
		log.Printf("Error updating bill: %v", err)
		SetFlash(w, "Failed to update bill.")
	} else {
		SetFlash(w, "Bill updated successfully.")
	}

	http.Redirect(w, r, "/bills", http.StatusSeeOther)
}

func BillDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/bills", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/bills/")
	idStr = strings.TrimSuffix(idStr, "/delete")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.DeleteBill(id); err != nil {
		log.Printf("Error deleting bill: %v", err)
		SetFlash(w, "Failed to delete bill.")
	} else {
		SetFlash(w, "Bill deleted.")
	}

	http.Redirect(w, r, "/bills", http.StatusSeeOther)
}

// parseCents converts a dollar string like "45.99" to cents (4599)
func parseCents(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return int(math.Round(f * 100)), nil
}
