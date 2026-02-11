package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bardenit/Bard/db"
	"github.com/bardenit/Bard/handlers"
)

func main() {
	db.Init()
	defer db.DB.Close()

	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "templates"
	}
	handlers.InitTemplates(templateDir)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "static"
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Dashboard
	mux.HandleFunc("GET /{$}", handlers.DashboardHandler)

	// Bills — single handler routes all /bills paths
	mux.HandleFunc("/bills", handlers.BillsRouter)
	mux.HandleFunc("/bills/{path...}", handlers.BillsRouter)

	// Income
	mux.HandleFunc("/income", handlers.IncomeRouter)
	mux.HandleFunc("/income/{path...}", handlers.IncomeRouter)

	// Budgets
	mux.HandleFunc("/budgets", handlers.BudgetsRouter)
	mux.HandleFunc("/budgets/{path...}", handlers.BudgetsRouter)

	// Calendar
	mux.HandleFunc("GET /calendar", handlers.CalendarHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

