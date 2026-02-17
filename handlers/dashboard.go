package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bardenit/Bard/models"
)

type MonthOption struct {
	Year     int
	Month    int
	Label    string
	Selected bool
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	// Parse optional year/month query params for budget status view.
	selectedYear := now.Year()
	selectedMonth := int(now.Month())
	if y := r.URL.Query().Get("year"); y != "" {
		if v, err := strconv.Atoi(y); err == nil {
			selectedYear = v
		}
	}
	if m := r.URL.Query().Get("month"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 1 && v <= 12 {
			selectedMonth = v
		}
	}

	// Summary cards always use current month.
	curYear, curMonth := now.Year(), now.Month()

	monthlyIncome, err := models.MonthlyIncomeTotal(curYear, curMonth)
	if err != nil {
		log.Printf("Error calculating monthly income: %v", err)
	}

	monthlyBills, err := models.MonthlyBillsTotal(curYear, curMonth)
	if err != nil {
		log.Printf("Error calculating monthly bills: %v", err)
	}

	totalBudgeted, err := models.TotalMonthlyBudgeted()
	if err != nil {
		log.Printf("Error calculating total budgeted: %v", err)
	}

	// Balance from latest imported transaction with a balance value.
	balance, hasBalance, err := models.GetLatestBalance()
	if err != nil {
		log.Printf("Error getting latest balance: %v", err)
	}
	unallocated := balance - totalBudgeted

	upcomingBills, err := models.UpcomingBillOccurrences(10)
	if err != nil {
		log.Printf("Error getting upcoming bills: %v", err)
	}

	upcomingDeposits, err := models.UpcomingIncomeOccurrences(5)
	if err != nil {
		log.Printf("Error getting upcoming deposits: %v", err)
	}

	// Budget status for the selected month.
	categoryTree, err := models.ListCategoryTree()
	if err != nil {
		log.Printf("Error getting category tree: %v", err)
	}

	spending, err := models.GetAllCategorySpending(selectedYear, time.Month(selectedMonth))
	if err != nil {
		log.Printf("Error getting category spending: %v", err)
	}

	var budgetStatuses []BudgetStatus
	for _, cat := range categoryTree {
		addBudgetStatuses(&budgetStatuses, cat, spending, 0)
	}

	// Build month options — current month plus previous 12.
	var monthOptions []MonthOption
	for i := 0; i < 13; i++ {
		t := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.Local)
		monthOptions = append(monthOptions, MonthOption{
			Year:     t.Year(),
			Month:    int(t.Month()),
			Label:    t.Format("January 2006"),
			Selected: t.Year() == selectedYear && int(t.Month()) == selectedMonth,
		})
	}

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

	selectedLabel := time.Date(selectedYear, time.Month(selectedMonth), 1, 0, 0, 0, 0, time.Local).Format("January 2006")

	data := PageData{
		Title:     "Dashboard",
		ActiveNav: "dashboard",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"MonthlyIncome":   monthlyIncome,
		"MonthlyBills":    monthlyBills,
		"Balance":         balance,
		"HasBalance":      hasBalance,
		"TotalBudgeted":   totalBudgeted,
		"Unallocated":     unallocated,
		"UpcomingBills":   billItems,
		"UpcomingDeposits": depositItems,
		"BudgetStatuses":  budgetStatuses,
		"MonthOptions":    monthOptions,
		"SelectedLabel":   selectedLabel,
		"SelectedYear":    fmt.Sprintf("%d", selectedYear),
		"SelectedMonth":   fmt.Sprintf("%d", selectedMonth),
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
	Depth        int
}

func categoryTotals(cat models.BudgetCategory, spending map[int]int) (allocated, spent int) {
	if cat.Budget != nil {
		allocated = cat.Budget.MonthlyAmount()
	}
	spent = spending[cat.ID]

	for _, child := range cat.Children {
		childAlloc, childSpent := categoryTotals(child, spending)
		allocated += childAlloc
		spent += childSpent
	}
	return
}

func addBudgetStatuses(statuses *[]BudgetStatus, cat models.BudgetCategory, spending map[int]int, depth int) {
	allocated, spent := categoryTotals(cat, spending)

	if allocated > 0 || spent > 0 {
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
			Depth:        depth,
		})
	}
	for _, child := range cat.Children {
		addBudgetStatuses(statuses, child, spending, depth+1)
	}
}
