package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bardenit/Bard/models"
)

func BudgetsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/budgets":
		if r.Method == http.MethodGet {
			BudgetsHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	case path == "/budgets/category/create" && r.Method == http.MethodPost:
		CategoryCreateHandler(w, r)
	case path == "/budgets/set" && r.Method == http.MethodPost:
		BudgetSetHandler(w, r)
	case strings.HasSuffix(path, "/delete"):
		CategoryDeleteHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func BudgetsHandler(w http.ResponseWriter, r *http.Request) {
	tree, err := models.ListCategoryTree()
	if err != nil {
		log.Printf("Error listing categories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topLevel, err := models.ListTopLevelCategories()
	if err != nil {
		log.Printf("Error listing top-level categories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalBudgeted, err := models.TotalMonthlyBudgeted()
	if err != nil {
		log.Printf("Error calculating total budgeted: %v", err)
	}

	data := PageData{
		Title:     "Budgets",
		ActiveNav: "budgets",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Categories":    tree,
		"TopLevel":      topLevel,
		"TotalBudgeted": totalBudgeted,
	}
	RenderTemplate(w, "budgets.html", data)
}

func CategoryCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	parentIDStr := r.FormValue("parent_id")

	if name == "" {
		SetFlash(w, "Category name is required.")
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}

	var parentID *int
	if parentIDStr != "" {
		pid, err := strconv.Atoi(parentIDStr)
		if err == nil {
			parentID = &pid
		}
	}

	if err := models.CreateCategory(name, parentID); err != nil {
		log.Printf("Error creating category: %v", err)
		SetFlash(w, "Failed to create category.")
	} else {
		SetFlash(w, "Category created.")
	}

	http.Redirect(w, r, "/budgets", http.StatusSeeOther)
}

func CategoryDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/budgets/category/")
	idStr = strings.TrimSuffix(idStr, "/delete")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.DeleteCategory(id); err != nil {
		log.Printf("Error deleting category: %v", err)
		SetFlash(w, "Failed to delete category.")
	} else {
		SetFlash(w, "Category deleted.")
	}

	http.Redirect(w, r, "/budgets", http.StatusSeeOther)
}

func BudgetSetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}

	categoryIDStr := r.FormValue("category_id")
	amountStr := r.FormValue("amount")
	recurrence := r.FormValue("recurrence")

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		SetFlash(w, "Invalid category.")
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}

	amount, err := parseCents(amountStr)
	if err != nil {
		SetFlash(w, "Invalid amount.")
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}

	effectiveDate := time.Now().Format("2006-01-02")

	if err := models.SetBudget(categoryID, amount, recurrence, effectiveDate); err != nil {
		log.Printf("Error setting budget: %v", err)
		SetFlash(w, "Failed to set budget.")
	} else {
		SetFlash(w, "Budget updated.")
	}

	http.Redirect(w, r, "/budgets", http.StatusSeeOther)
}
