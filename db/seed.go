package db

import (
	"log"
)

// SeedInitialData inserts default budget categories and transaction rules on a fresh DB.
// It is a no-op if any budget_categories rows already exist.
func SeedInitialData() {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM budget_categories`).Scan(&count); err != nil {
		log.Printf("Seed: failed to count categories: %v", err)
		return
	}
	if count > 0 {
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Seed: failed to begin transaction: %v", err)
		return
	}

	// cat inserts a budget category and returns its ID.
	// On the first error it short-circuits all subsequent calls.
	cat := func(name string, parentID *int64) int64 {
		if err != nil {
			return 0
		}
		var result interface{ LastInsertId() (int64, error) }
		if parentID != nil {
			r, e := tx.Exec(`INSERT INTO budget_categories (name, parent_id) VALUES (?, ?)`, name, *parentID)
			if e != nil {
				err = e
				return 0
			}
			result = r
		} else {
			r, e := tx.Exec(`INSERT INTO budget_categories (name) VALUES (?)`, name)
			if e != nil {
				err = e
				return 0
			}
			result = r
		}
		id, _ := result.LastInsertId()
		return id
	}

	// Insert parent categories.
	income := cat("Income", nil)
	housing := cat("Housing", nil)
	food := cat("Food & Dining", nil)
	transport := cat("Transportation", nil)
	health := cat("Health", nil)
	ent := cat("Entertainment", nil)
	shopping := cat("Shopping", nil)
	financial := cat("Financial", nil)
	other := cat("Other", nil)

	// Insert child categories; capture IDs needed for rules.
	payrollID := cat("Payroll", &income)
	cat("Other Income", &income)
	utilitiesID := cat("Utilities", &housing)
	insuranceID := cat("Insurance", &housing)
	groceriesID := cat("Groceries", &food)
	restaurantsID := cat("Restaurants", &food)
	coffeeID := cat("Coffee & Snacks", &food)
	gasID := cat("Gas", &transport)
	medicalID := cat("Medical", &health)
	lotteryID := cat("Lottery", &ent)
	wineID := cat("Wine & Spirits", &ent)
	homeImprovID := cat("Home Improvement", &shopping)
	generalShopID := cat("General Shopping", &shopping)
	creditCardID := cat("Credit Card Payments", &financial)
	loanPayID := cat("Loan Payments", &financial)
	savingsID := cat("Savings Transfers", &financial)

	if err != nil {
		log.Printf("Seed: category insert failed: %v", err)
		tx.Rollback()
		return
	}

	rules := []struct {
		keyword    string
		categoryID int64
	}{
		{"MEIJER", groceriesID},
		{"SAMSCLUB", groceriesID},
		{"COSTCO WHSE", groceriesID},
		{"JERSEY MIKE", restaurantsID},
		{"ARBYS", restaurantsID},
		{"PIZZA COMPANY", restaurantsID},
		{"JETS PIZZA", restaurantsID},
		{"NOODLES", restaurantsID},
		{"MCDONALD", restaurantsID},
		{"DETROIT WING", restaurantsID},
		{"FIREHOUSE SUBS", restaurantsID},
		{"JIMMY JOHN", restaurantsID},
		{"BIGGBY", coffeeID},
		{"STARBUCKS", coffeeID},
		{"COLEY VENDING", coffeeID},
		{"MICHIGANLOTTERY", lotteryID},
		{"MARATHON", gasID},
		{"HOME DEPOT", homeImprovID},
		{"FAMILY DOLLAR", generalShopID},
		{"ST. JULIAN", wineID},
		{"CONSUMERS ENERGY", utilitiesID},
		{"SPECTRUM", utilitiesID},
		{"NATIONWIDE", insuranceID},
		{"HURLEY MEDICAL CENTER", medicalID},
		{"HURLEY FOUNDATION", medicalID},
		{"AMERICAN EXPRESS", creditCardID},
		{"CITI CARD", creditCardID},
		{"APPLECARD", creditCardID},
		{"CHASE MASTERCARD", creditCardID},
		{"MSU CREDIT UNION", loanPayID},
		{"TRANSFER TO SHARE", savingsID},
		{"TRANSFER TO LOAN", loanPayID},
		{"VENMO", other},
		{"HURLEY MEDICAL C", payrollID},
		{"UTAC US", payrollID},
	}

	for _, rule := range rules {
		if rule.categoryID == 0 {
			continue
		}
		if _, e := tx.Exec(`INSERT OR IGNORE INTO transaction_rules (keyword, category_id) VALUES (?, ?)`, rule.keyword, rule.categoryID); e != nil {
			log.Printf("Seed: insert rule %q: %v", rule.keyword, e)
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Seed: commit failed: %v", err)
		return
	}
	log.Println("Seed: initial data seeded successfully")
}
