package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bardenit/Bard/models"
)

func ExpendituresRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/expenditures":
		if r.Method == http.MethodGet {
			ExpendituresHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	case path == "/expenditures/create" && r.Method == http.MethodPost:
		ExpenditureCreateHandler(w, r)
	case strings.HasSuffix(path, "/edit"):
		ExpenditureEditHandler(w, r)
	case strings.HasSuffix(path, "/update"):
		ExpenditureUpdateHandler(w, r)
	case strings.HasSuffix(path, "/delete"):
		ExpenditureDeleteHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func ExpendituresHandler(w http.ResponseWriter, r *http.Request) {
	filterMonth := r.URL.Query().Get("month")
	filterCatID, _ := strconv.Atoi(r.URL.Query().Get("category_id"))

	expenditures, err := models.ListExpendituresFiltered(filterMonth, filterCatID)
	if err != nil {
		log.Printf("Error listing expenditures: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	categories, err := models.ListCategories()
	if err != nil {
		log.Printf("Error listing categories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	monthsWithData, err := models.MonthsWithSpending()
	if err != nil {
		log.Printf("Error getting months with spending: %v", err)
	}

	// Build a total for the current filter view.
	total := 0
	for _, e := range expenditures {
		total += e.Amount
	}

	data := PageData{
		Title:     "Expenditures",
		ActiveNav: "expenditures",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Expenditures":    expenditures,
		"Categories":      categories,
		"MonthsWithData":  monthsWithData,
		"FilterMonth":     filterMonth,
		"FilterCategoryID": filterCatID,
		"FilterTotal":     total,
	}
	RenderTemplate(w, "expenditures.html", data)
}

func ExpenditureCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	amountStr := r.FormValue("amount")
	categoryIDStr := r.FormValue("category_id")
	date := r.FormValue("date")

	amount, err := parseCents(amountStr)
	if err != nil || description == "" || date == "" {
		SetFlash(w, "Please fill in all fields with valid values.")
		http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
		return
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		SetFlash(w, "Please select a valid category.")
		http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
		return
	}

	if err := models.CreateExpenditure(description, amount, categoryID, date); err != nil {
		log.Printf("Error creating expenditure: %v", err)
		SetFlash(w, "Failed to create expenditure.")
		http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
		return
	}

	SetFlash(w, "Expenditure created successfully.")
	http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
}

func ExpenditureEditHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/expenditures/")
	idStr = strings.TrimSuffix(idStr, "/edit")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	expenditure, err := models.GetExpenditure(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categories, err := models.ListCategories()
	if err != nil {
		log.Printf("Error listing categories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:     "Edit Expenditure",
		ActiveNav: "expenditures",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Expenditure": expenditure,
		"Categories":  categories,
		"Editing":     true,
	}
	RenderTemplate(w, "expenditures.html", data)
}

func ExpenditureUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/expenditures/")
	idStr = strings.TrimSuffix(idStr, "/update")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	amountStr := r.FormValue("amount")
	categoryIDStr := r.FormValue("category_id")
	date := r.FormValue("date")

	amount, err := parseCents(amountStr)
	if err != nil || description == "" || date == "" {
		SetFlash(w, "Please fill in all fields with valid values.")
		http.Redirect(w, r, "/expenditures/"+strconv.Itoa(id)+"/edit", http.StatusSeeOther)
		return
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		SetFlash(w, "Please select a valid category.")
		http.Redirect(w, r, "/expenditures/"+strconv.Itoa(id)+"/edit", http.StatusSeeOther)
		return
	}

	if err := models.UpdateExpenditure(id, description, amount, categoryID, date); err != nil {
		log.Printf("Error updating expenditure: %v", err)
		SetFlash(w, "Failed to update expenditure.")
	} else {
		SetFlash(w, "Expenditure updated successfully.")
	}

	http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
}

func ExpenditureDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/expenditures/")
	idStr = strings.TrimSuffix(idStr, "/delete")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.DeleteExpenditure(id); err != nil {
		log.Printf("Error deleting expenditure: %v", err)
		SetFlash(w, "Failed to delete expenditure.")
	} else {
		SetFlash(w, "Expenditure deleted.")
	}

	http.Redirect(w, r, "/expenditures", http.StatusSeeOther)
}
