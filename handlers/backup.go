package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bardenit/Bard/models"
)

// BackupData is the JSON structure used for export and import.
type BackupData struct {
	Version             string                      `json:"version"`
	ExportedAt          string                      `json:"exported_at"`
	IncludeTransactions bool                        `json:"include_transactions"`
	Bills               []models.BackupBill         `json:"bills"`
	Income              []models.BackupIncome       `json:"income"`
	Categories          []models.BackupCategory     `json:"budget_categories"`
	Budgets             []models.BackupBudget       `json:"budgets"`
	Expenditures        []models.BackupExpenditure  `json:"expenditures"`
	Rules               []models.BackupRule         `json:"transaction_rules"`
	Transactions        []models.BackupTransaction  `json:"imported_transactions,omitempty"`
	Settings            map[string]string           `json:"settings"`
}

// BackupRouter dispatches /backup/* requests.
func BackupRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/backup" && r.Method == http.MethodGet:
		BackupPageHandler(w, r)
	case path == "/backup/export" && r.Method == http.MethodGet:
		BackupExportHandler(w, r)
	case path == "/backup/restore" && r.Method == http.MethodPost:
		BackupRestoreHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

// BackupPageHandler renders the backup/restore page.
func BackupPageHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:     "Backup & Restore",
		ActiveNav: "backup",
		Flash:     GetFlash(w, r),
	}
	RenderTemplate(w, "backup.html", data)
}

// BackupExportHandler generates a JSON backup file and sends it as a download.
// Query param: include_transactions=1 to include imported_transactions table.
func BackupExportHandler(w http.ResponseWriter, r *http.Request) {
	includeTxns := r.URL.Query().Get("include_transactions") == "1"

	bd := &BackupData{
		Version:             AppVersion,
		ExportedAt:          time.Now().Format(time.RFC3339),
		IncludeTransactions: includeTxns,
	}

	var err error
	if bd.Bills, err = models.ExportAllBills(); err != nil {
		log.Printf("BackupExport bills: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}
	if bd.Income, err = models.ExportAllIncome(); err != nil {
		log.Printf("BackupExport income: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}
	if bd.Categories, err = models.ExportAllCategories(); err != nil {
		log.Printf("BackupExport categories: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}
	if bd.Budgets, err = models.ExportAllBudgets(); err != nil {
		log.Printf("BackupExport budgets: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}
	if bd.Expenditures, err = models.ExportAllExpenditures(); err != nil {
		log.Printf("BackupExport expenditures: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}
	if bd.Rules, err = models.ExportAllRules(); err != nil {
		log.Printf("BackupExport rules: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}
	if includeTxns {
		if bd.Transactions, err = models.ExportAllTransactions(); err != nil {
			log.Printf("BackupExport transactions: %v", err)
			http.Error(w, "Export failed", http.StatusInternalServerError)
			return
		}
	}
	if bd.Settings, err = models.ExportAllSettings(); err != nil {
		log.Printf("BackupExport settings: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}

	payload, err := json.MarshalIndent(bd, "", "  ")
	if err != nil {
		log.Printf("BackupExport marshal: %v", err)
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}

	suffix := "no-transactions"
	if includeTxns {
		suffix = "full"
	}
	filename := fmt.Sprintf("budget-backup-%s-%s.json", time.Now().Format("2006-01-02"), suffix)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(payload)
}

// BackupRestoreHandler accepts a JSON backup file upload and restores the database.
func BackupRestoreHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		SetFlash(w, "Failed to parse upload.")
		http.Redirect(w, r, "/backup", http.StatusSeeOther)
		return
	}

	file, _, err := r.FormFile("backup_file")
	if err != nil {
		SetFlash(w, "Please select a backup file.")
		http.Redirect(w, r, "/backup", http.StatusSeeOther)
		return
	}
	defer file.Close()

	var bd BackupData
	if err := json.NewDecoder(file).Decode(&bd); err != nil {
		SetFlash(w, "Invalid backup file: "+err.Error())
		http.Redirect(w, r, "/backup", http.StatusSeeOther)
		return
	}

	// Validate that the backup file looks reasonable.
	if !strings.HasPrefix(bd.Version, "2.") {
		SetFlash(w, fmt.Sprintf("Backup version %q may not be compatible — proceeding anyway.", bd.Version))
	}

	includeTxns := bd.IncludeTransactions && len(bd.Transactions) > 0

	if err := models.RestoreData(
		bd.Categories,
		bd.Budgets,
		bd.Bills,
		bd.Income,
		bd.Expenditures,
		bd.Rules,
		bd.Transactions,
		bd.Settings,
		includeTxns,
	); err != nil {
		log.Printf("RestoreData error: %v", err)
		SetFlash(w, "Restore failed: "+err.Error())
		http.Redirect(w, r, "/backup", http.StatusSeeOther)
		return
	}

	msg := fmt.Sprintf("Restore complete — %d bills, %d income, %d categories, %d expenditures",
		len(bd.Bills), len(bd.Income), len(bd.Categories), len(bd.Expenditures))
	if includeTxns {
		msg += fmt.Sprintf(", %d transactions", len(bd.Transactions))
	}
	SetFlash(w, msg)
	http.Redirect(w, r, "/backup", http.StatusSeeOther)
}
