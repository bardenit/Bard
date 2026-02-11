package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/bardenit/Bard/models"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	year, month := now.Year(), now.Month()

	monthlyIncome, err := models.MonthlyIncomeTotal(year, month)
	if err != nil {
		log.Printf("Error calculating monthly income: %v", err)
	}

	monthlyBills, err := models.MonthlyBillsTotal(year, month)
	if err != nil {
		log.Printf("Error calculating monthly bills: %v", err)
	}

	netAvailable := monthlyIncome - monthlyBills

	totalBudgeted, err := models.TotalMonthlyBudgeted()
	if err != nil {
		log.Printf("Error calculating total budgeted: %v", err)
	}

	unallocated := netAvailable - totalBudgeted

	upcomingBills, err := models.UpcomingBillOccurrences(10)
	if err != nil {
		log.Printf("Error getting upcoming bills: %v", err)
	}

	upcomingDeposits, err := models.UpcomingIncomeOccurrences(5)
	if err != nil {
		log.Printf("Error getting upcoming deposits: %v", err)
	}

	// Budget status: per-category allocated vs spent
	categoryTree, err := models.ListCategoryTree()
	if err != nil {
		log.Printf("Error getting category tree: %v", err)
	}

	spending, err := models.GetAllCategorySpending(year, month)
	if err != nil {
		log.Printf("Error getting category spending: %v", err)
	}

	var budgetStatuses []BudgetStatus
	for _, cat := range categoryTree {
		addBudgetStatuses(&budgetStatuses, cat, spending)
	}

	// Convert to template-friendly format
	var billItems []UpcomingItem
	for _, o := range upcomingBills {
		billItems = append(billItems, UpcomingItem{
			Name:   o.Name,
			Date:   o.Date.Format("Jan 2"),
			Amount: o.Amount,
		})
	}

	var depositItems []UpcomingItem
	for _, o := range upcomingDeposits {
		depositItems = append(depositItems, UpcomingItem{
			Name:   o.Name,
			Date:   o.Date.Format("Jan 2"),
			Amount: o.Amount,
		})
	}

	data := PageData{
		Title:     "Dashboard",
		ActiveNav: "dashboard",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"MonthlyIncome":    monthlyIncome,
		"MonthlyBills":     monthlyBills,
		"NetAvailable":     netAvailable,
		"TotalBudgeted":    totalBudgeted,
		"Unallocated":      unallocated,
		"UpcomingBills":    billItems,
		"UpcomingDeposits": depositItems,
		"BudgetStatuses":   budgetStatuses,
	}

	RenderTemplate(w, "dashboard.html", data)
}

type UpcomingItem struct {
	Name   string
	Date   string
	Amount int
}

type BudgetStatus struct {
	CategoryName string
	Allocated    int
	Spent        int
	Remaining    int
	Percent      int
}

func addBudgetStatuses(statuses *[]BudgetStatus, cat models.BudgetCategory, spending map[int]int) {
	if cat.Budget != nil {
		allocated := cat.Budget.MonthlyAmount()
		spent := spending[cat.ID]
		remaining := allocated - spent
		pct := 0
		if allocated > 0 {
			pct = spent * 100 / allocated
			if pct > 100 {
				pct = 100
			}
		}
		*statuses = append(*statuses, BudgetStatus{
			CategoryName: cat.Name,
			Allocated:    allocated,
			Spent:        spent,
			Remaining:    remaining,
			Percent:      pct,
		})
	}
	for _, child := range cat.Children {
		addBudgetStatuses(statuses, child, spending)
	}
}
