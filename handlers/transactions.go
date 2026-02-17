package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bardenit/Bard/models"
)

// TransactionsRouter dispatches all /transactions/* requests.
func TransactionsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/transactions" && r.Method == http.MethodGet:
		TransactionsHandler(w, r)
	case path == "/transactions/import" && r.Method == http.MethodPost:
		TransactionImportHandler(w, r)
	case path == "/transactions/confirm-all-auto" && r.Method == http.MethodPost:
		TransactionConfirmAllAutoHandler(w, r)
	case path == "/transactions/dismiss-all-credits" && r.Method == http.MethodPost:
		TransactionDismissAllCreditsHandler(w, r)
	case path == "/transactions/rules/delete" && r.Method == http.MethodPost:
		TransactionRuleDeleteHandler(w, r)
	case strings.HasPrefix(path, "/transactions/rules/") && strings.HasSuffix(path, "/update") && r.Method == http.MethodPost:
		TransactionRuleUpdateHandler(w, r)
	case strings.HasSuffix(path, "/confirm") && r.Method == http.MethodPost:
		TransactionConfirmHandler(w, r)
	case strings.HasSuffix(path, "/dismiss") && r.Method == http.MethodPost:
		TransactionDismissHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

// TransactionsHandler renders the transactions page.
func TransactionsHandler(w http.ResponseWriter, r *http.Request) {
	pending, err := models.ListPendingTransactions()
	if err != nil {
		log.Printf("Error listing pending transactions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	rules, err := models.ListRules()
	if err != nil {
		log.Printf("Error listing rules: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	categories, err := models.ListCategories()
	if err != nil {
		log.Printf("Error listing categories: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	confirmed, err := models.ListConfirmedTransactions(50)
	if err != nil {
		log.Printf("Error listing confirmed transactions: %v", err)
		confirmed = nil
	}

	balance, _, err := models.GetManualBalance()
	if err != nil {
		log.Printf("Error getting balance: %v", err)
	}

	data := PageData{
		Title:     "Transactions",
		ActiveNav: "transactions",
		Flash:     GetFlash(w, r),
	}
	data.Extra = map[string]interface{}{
		"Pending":    pending,
		"Rules":      rules,
		"Categories": categories,
		"Confirmed":  confirmed,
		"Balance":    balance,
	}
	RenderTemplate(w, "transactions.html", data)
}

// TransactionImportHandler parses an uploaded CSV and imports transactions.
func TransactionImportHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		SetFlash(w, "Failed to parse upload.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}

	file, header, err := r.FormFile("csv_file")
	if err != nil {
		SetFlash(w, "Please select a CSV file to upload.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}
	defer file.Close()

	sourceFile := header.Filename

	cr := csv.NewReader(file)
	cr.LazyQuotes = true

	// Skip header row.
	if _, err := cr.Read(); err != nil {
		SetFlash(w, "Failed to read CSV header.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}

	imported := 0
	skipped := 0

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("CSV read error: %v", err)
			continue
		}
		if len(record) < 6 {
			continue
		}

		accountName := strings.TrimSpace(record[0])
		processedDate := strings.TrimSpace(record[1])
		description := strings.TrimSpace(record[2])
		// record[3] is check# — skip
		creditOrDebit := strings.TrimSpace(record[4])
		amountStr := strings.TrimSpace(record[5])

		amount, err := parseCents(amountStr)
		if err != nil || description == "" || processedDate == "" {
			continue
		}

		// Skip rows that aren't clearly Credit or Debit.
		if creditOrDebit != "Credit" && creditOrDebit != "Debit" {
			continue
		}

		// Parse optional balance column (column 6).
		var balance *int
		if len(record) >= 7 {
			if b, err := parseCents(strings.TrimSpace(record[6])); err == nil {
				balance = &b
			}
		}

		// Check for import duplicate (same description+date+amount already imported).
		isDupImport, err := models.CheckImportDuplicate(description, processedDate, amount)
		if err != nil {
			log.Printf("CheckImportDuplicate error: %v", err)
		}
		if isDupImport {
			skipped++
			continue
		}

		// Auto-categorize.
		autoCatID, err := models.AutoCategorize(description)
		if err != nil {
			log.Printf("AutoCategorize error: %v", err)
		}

		// Flag as potential duplicate if same amount exists in expenditures.
		isDup := false
		if creditOrDebit == "Debit" {
			isDup, err = models.CheckDuplicate(amount)
			if err != nil {
				log.Printf("CheckDuplicate error: %v", err)
			}
		}

		if _, err := models.CreateImportedTransaction(accountName, processedDate, description, creditOrDebit, amount, balance, autoCatID, isDup, sourceFile); err != nil {
			log.Printf("CreateImportedTransaction error: %v", err)
			continue
		}
		imported++
	}

	msg := fmt.Sprintf("Imported %d transaction(s)", imported)
	if skipped > 0 {
		msg += fmt.Sprintf(", skipped %d duplicate(s)", skipped)
	}
	SetFlash(w, msg)
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

// TransactionConfirmHandler confirms a single transaction, creating an expenditure.
func TransactionConfirmHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/transactions/")
	idStr = strings.TrimSuffix(idStr, "/confirm")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categoryIDStr := r.FormValue("category_id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil || categoryID == 0 {
		SetFlash(w, "Please select a valid category.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}

	tx, err := models.GetTransaction(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.ConfirmTransaction(id, categoryID); err != nil {
		log.Printf("ConfirmTransaction error: %v", err)
		SetFlash(w, "Failed to confirm transaction.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}

	// Learn rule when user changes or sets a category on a debit.
	if tx.CreditOrDebit == "Debit" {
		shouldLearn := tx.AutoCategoryID == nil || *tx.AutoCategoryID != categoryID
		if shouldLearn {
			keyword := models.ExtractKeyword(tx.Description)
			if keyword != "" {
				if err := models.SaveRule(keyword, categoryID); err != nil {
					log.Printf("SaveRule error (non-fatal): %v", err)
				}
			}
		}
	}

	SetFlash(w, "Transaction confirmed.")
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

// TransactionDismissHandler dismisses a single transaction.
func TransactionDismissHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/transactions/")
	idStr = strings.TrimSuffix(idStr, "/dismiss")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := models.DismissTransaction(id); err != nil {
		log.Printf("DismissTransaction error: %v", err)
		SetFlash(w, "Failed to dismiss transaction.")
	}
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

// TransactionConfirmAllAutoHandler confirms all pending debits with an auto-category.
func TransactionConfirmAllAutoHandler(w http.ResponseWriter, r *http.Request) {
	n, err := models.ConfirmAllAuto()
	if err != nil {
		log.Printf("ConfirmAllAuto error: %v", err)
		SetFlash(w, "Failed to confirm transactions.")
	} else {
		SetFlash(w, fmt.Sprintf("Confirmed %d transaction(s).", n))
	}
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

// TransactionDismissAllCreditsHandler dismisses all pending credit transactions.
func TransactionDismissAllCreditsHandler(w http.ResponseWriter, r *http.Request) {
	n, err := models.DismissAllCredits()
	if err != nil {
		log.Printf("DismissAllCredits error: %v", err)
		SetFlash(w, "Failed to dismiss credits.")
	} else {
		SetFlash(w, fmt.Sprintf("Dismissed %d credit(s).", n))
	}
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

// TransactionRuleDeleteHandler deletes a categorization rule.
func TransactionRuleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ruleIDStr := r.FormValue("rule_id")
	ruleID, err := strconv.Atoi(ruleIDStr)
	if err != nil || ruleID == 0 {
		SetFlash(w, "Invalid rule ID.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}

	if err := models.DeleteRule(ruleID); err != nil {
		log.Printf("DeleteRule error: %v", err)
		SetFlash(w, "Failed to delete rule.")
	} else {
		SetFlash(w, "Rule deleted.")
	}
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}

// TransactionRuleUpdateHandler updates the category of an existing rule.
func TransactionRuleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// Path: /transactions/rules/{id}/update
	path := strings.TrimPrefix(r.URL.Path, "/transactions/rules/")
	path = strings.TrimSuffix(path, "/update")
	ruleID, err := strconv.Atoi(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categoryIDStr := r.FormValue("category_id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil || categoryID == 0 {
		SetFlash(w, "Please select a valid category.")
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
		return
	}

	if err := models.UpdateRule(ruleID, categoryID); err != nil {
		log.Printf("UpdateRule error: %v", err)
		SetFlash(w, "Failed to update rule.")
	} else {
		SetFlash(w, "Rule updated.")
	}
	http.Redirect(w, r, "/transactions", http.StatusSeeOther)
}
