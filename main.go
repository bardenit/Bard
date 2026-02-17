package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bardenit/Bard/db"
	"github.com/bardenit/Bard/handlers"
)

// BuildTime is injected at build time via -ldflags "-X main.BuildTime=<RFC3339 timestamp>".
var BuildTime = "unknown"

func main() {
	db.Init()
	defer db.DB.Close()

	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "templates"
	}
	handlers.InitTemplates(templateDir)

	// Start background upgrade checker using the embedded build timestamp.
	handlers.StartUpgradeChecker(BuildTime)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "static"
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Dashboard
	mux.HandleFunc("GET /{$}", handlers.DashboardHandler)

	// Bills
	mux.HandleFunc("/bills", handlers.BillsRouter)
	mux.HandleFunc("/bills/{path...}", handlers.BillsRouter)

	// Income
	mux.HandleFunc("/income", handlers.IncomeRouter)
	mux.HandleFunc("/income/{path...}", handlers.IncomeRouter)

	// Budgets
	mux.HandleFunc("/budgets", handlers.BudgetsRouter)
	mux.HandleFunc("/budgets/{path...}", handlers.BudgetsRouter)

	// Expenditures
	mux.HandleFunc("/expenditures", handlers.ExpendituresRouter)
	mux.HandleFunc("/expenditures/{path...}", handlers.ExpendituresRouter)

	// Calendar
	mux.HandleFunc("GET /calendar", handlers.CalendarHandler)

	// Transactions
	mux.HandleFunc("/transactions", handlers.TransactionsRouter)
	mux.HandleFunc("/transactions/{path...}", handlers.TransactionsRouter)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s (v%s, built %s)", port, handlers.AppVersion, BuildTime)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
