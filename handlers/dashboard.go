package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bardenit/Bard/models"
)

// SetBalanceHandler handles POST /balance to manually set the account balance.
func SetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	amountStr := r.FormValue("balance")
	cents, err := parseCents(amountStr)
	if err != nil {
		SetFlash(w, "Invalid balance amount.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := models.SetManualBalance(cents); err != nil {
		log.Printf("SetManualBalance error: %v", err)
		SetFlash(w, "Failed to save balance.")
	}
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

type MonthOption struct {
	Year     int
	Month    int
	Label    string
	Selected bool
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	// Parse optional year/month query params for budget status view.
	requestedYear := now.Year()
	requestedMonth := int(now.Month())
	if y := r.URL.Query().Get("year"); y != "" {
		if v, err := strconv.Atoi(y); err == nil {
			requestedYear = v
		}
	}
	if m := r.URL.Query().Get("month"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 1 && v <= 12 {
			requestedMonth = v
		}
	}

	// Build month options from months that actually have spending data.
	monthsWithData, err := models.MonthsWithSpending()
	if err != nil {
		log.Printf("Error getting months with spending: %v", err)
	}

	// Validate selected month; fall back to most recent month with data.
	selectedYear, selectedMonth := requestedYear, requestedMonth
	if len(monthsWithData) > 0 {
		found := false
		for _, t := range monthsWithData {
			if t.Year() == selectedYear && int(t.Month()) == selectedMonth {
				found = true
				break
			}
		}
		if !found {
			selectedYear = monthsWithData[0].Year()
			selectedMonth = int(monthsWithData[0].Month())
		}
	}

	var monthOptions []MonthOption
	for _, t := range monthsWithData {
		monthOptions = append(monthOptions, MonthOption{
			Year:     t.Year(),
			Month:    int(t.Month()),
			Label:    t.Format("January 2006"),
			Selected: t.Year() == selectedYear && int(t.Month()) == selectedMonth,
		})
	}

	upcomingBillsTotal, nextPaycheck, err := models.BillsDueBeforeNextPaycheck()
	if err != nil {
		log.Printf("Error calculating upcoming bills: %v", err)
	}

	totalBudgeted, err := models.TotalMonthlyBudgeted()
	if err != nil {
		log.Printf("Error calculating total budgeted: %v", err)
	}

	// Balance from manually entered account balance.
	balance, hasBalance, err := models.GetManualBalance()
	if err != nil {
		log.Printf("Error getting manual balance: %v", err)
	}
	unallocated := balance - totalBudgeted

	upcomingBills, err := models.UpcomingBillOccurrences()
	if err != nil {
		log.Printf("Error getting upcoming bills: %v", err)
	}

	upcomingDeposits, err := models.UpcomingIncomeOccurrences()
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

	var billsBefore, billsAfter []UpcomingItem
	for _, o := range upcomingBills {
		item := UpcomingItem{
			Name:   o.Name,
			Date:   o.Date.Format("Jan 2"),
			Amount: o.Amount,
		}
		if !o.Date.After(nextPaycheck) {
			billsBefore = append(billsBefore, item)
		} else {
			billsAfter = append(billsAfter, item)
		}
	}

	var depositItems []UpcomingItem
	for _, o := range upcomingDeposits {
		depositItems = append(depositItems, UpcomingItem{
			Name:   o.Name,
			Date:   o.Date.Format("Jan 2"),
			Amount: o.Amount,
		})
	}

	// Sum top-level (Depth==0) rows for the totals footer row.
	var budgetTotalAllocated, budgetTotalSpent int
	for _, bs := range budgetStatuses {
		if bs.Depth == 0 {
			budgetTotalAllocated += bs.Allocated
			budgetTotalSpent += bs.Spent
		}
	}
	budgetTotalRemaining := budgetTotalAllocated - budgetTotalSpent

	selectedLabel := time.Date(selectedYear, time.Month(selectedMonth), 1, 0, 0, 0, 0, time.Local).Format("January 2006")

	data := PageData{
		Title:     "Dashboard",
		ActiveNav: "dashboard",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"UpcomingBillsTotal":    upcomingBillsTotal,
		"NextPaycheckDate":      nextPaycheck.Format("Jan 2"),
		"Balance":               balance,
		"HasBalance":            hasBalance,
		"TotalBudgeted":         totalBudgeted,
		"Unallocated":           unallocated,
		"BillsBefore":           billsBefore,
		"BillsAfter":            billsAfter,
		"UpcomingDeposits":      depositItems,
		"BudgetStatuses":        budgetStatuses,
		"BudgetTotalAllocated":  budgetTotalAllocated,
		"BudgetTotalSpent":      budgetTotalSpent,
		"BudgetTotalRemaining":  budgetTotalRemaining,
		"MonthOptions":          monthOptions,
		"SelectedLabel":         selectedLabel,
		"SelectedYear":          fmt.Sprintf("%d", selectedYear),
		"SelectedMonth":         fmt.Sprintf("%d", selectedMonth),
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
