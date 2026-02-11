package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type PageData struct {
	Title     string
	ActiveNav string
	Flash     string
	Extra     map[string]interface{}
}

var (
	templateDir string
	funcMap     template.FuncMap
	pageCache   map[string]*template.Template
)

func InitTemplates(dir string) {
	templateDir = dir
	funcMap = template.FuncMap{
		"formatMoney":   formatMoney,
		"centsToDollar": centsToDollar,
		"dict":          dict,
		"multiply":      multiply,
	}

	// Pre-parse each page template paired with layout
	pages := []string{
		"dashboard.html",
		"bills.html",
		"income.html",
		"budgets.html",
		"expenditures.html",
		"calendar.html",
	}

	pageCache = make(map[string]*template.Template, len(pages))
	layoutPath := filepath.Join(dir, "layout.html")

	for _, page := range pages {
		pagePath := filepath.Join(dir, page)
		t := template.Must(
			template.New("").Funcs(funcMap).ParseFiles(layoutPath, pagePath),
		)
		pageCache[page] = t
	}
}

func RenderTemplate(w http.ResponseWriter, name string, data PageData) {
	t, ok := pageCache[name]
	if !ok {
		log.Printf("Template %s not found", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Merge Extra fields into a flat template data map
	td := map[string]interface{}{
		"Title":     data.Title,
		"ActiveNav": data.ActiveNav,
		"Flash":     data.Flash,
	}
	for k, v := range data.Extra {
		td[k] = v
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout.html", td); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func SetFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    msg,
		Path:     "/",
		MaxAge:   10,
		HttpOnly: true,
	})
}

func GetFlash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "flash",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return c.Value
}

func formatMoney(cents interface{}) string {
	var c int
	switch v := cents.(type) {
	case int:
		c = v
	case int64:
		c = int(v)
	default:
		return "$0.00"
	}

	negative := c < 0
	if negative {
		c = -c
	}

	dollars := c / 100
	remainder := c % 100

	s := fmt.Sprintf("$%d.%02d", dollars, remainder)
	if negative {
		s = "-" + s
	}

	// Add thousand separators
	parts := strings.Split(s, ".")
	intPart := parts[0]
	prefix := ""
	if strings.HasPrefix(intPart, "-$") {
		prefix = "-$"
		intPart = intPart[2:]
	} else {
		prefix = "$"
		intPart = intPart[1:]
	}

	if len(intPart) > 3 {
		var result []byte
		for i, d := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result = append(result, ',')
			}
			result = append(result, byte(d))
		}
		intPart = string(result)
	}

	return prefix + intPart + "." + parts[1]
}

// centsToDollar converts cents to a dollar string for form input values (e.g., 4599 -> "45.99")
func centsToDollar(cents interface{}) string {
	var c int
	switch v := cents.(type) {
	case int:
		c = v
	case int64:
		c = int(v)
	default:
		return "0.00"
	}
	return fmt.Sprintf("%.2f", float64(c)/100)
}

// multiply returns a * b as a float for use in style calculations
func multiply(a int, b float64) float64 {
	return float64(a) * b
}

// dict creates a map from key-value pairs for use in templates
func dict(values ...interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < len(values)-1; i += 2 {
		key, ok := values[i].(string)
		if ok {
			m[key] = values[i+1]
		}
	}
	return m
}
