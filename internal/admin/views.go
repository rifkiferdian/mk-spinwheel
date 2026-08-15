package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type ViewData struct {
	PageTitle string
	ActiveNav string
	Admin     LoginSession
	CSRF      string
	Notice    string
	Error     string
	Data      any
}

type Views struct {
	pages map[string]*template.Template
	login *template.Template
}

func NewViews() (*Views, error) {
	funcs := template.FuncMap{
		"date": formatDate, "datetimeLocal": datetimeLocal, "jsonPretty": jsonPretty,
		"short": shortText, "statusClass": statusClass, "add": func(a, b int) int { return a + b },
	}
	base := "templates/admin/base.html"
	files, err := filepath.Glob("templates/admin/pages/*.html")
	if err != nil {
		return nil, err
	}
	views := &Views{pages: make(map[string]*template.Template)}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		t, err := template.New("base.html").Funcs(funcs).ParseFiles(base, file)
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", name, err)
		}
		views.pages[name] = t
	}
	views.login, err = template.New("login.html").Funcs(funcs).ParseFiles("templates/admin/login.html")
	if err != nil {
		return nil, err
	}
	return views, nil
}

func (v *Views) Render(w http.ResponseWriter, name string, data ViewData) {
	t, ok := v.pages[name]
	if !ok {
		http.Error(w, "template tidak ditemukan", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Views) RenderLogin(w http.ResponseWriter, data ViewData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.login.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewData(r *http.Request, title, nav string, data any) ViewData {
	admin := currentAdmin(r)
	return ViewData{PageTitle: title, ActiveNav: nav, Admin: admin, CSRF: admin.CSRF, Notice: r.URL.Query().Get("notice"), Error: r.URL.Query().Get("error"), Data: data}
}
func redirectMessage(w http.ResponseWriter, r *http.Request, path, key, message string) {
	http.Redirect(w, r, path+"?"+key+"="+url.QueryEscape(message), http.StatusSeeOther)
}

func formatDate(value string) string {
	if value == "" {
		return "—"
	}
	formats := []string{time.RFC3339Nano, "2006-01-02T15:04", "2006-01-02 15:04:05"}
	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Local().Format("02 Jan 2006 15:04")
		}
	}
	return value
}
func datetimeLocal(value string) string {
	if value == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t.Local().Format("2006-01-02T15:04")
	}
	if len(value) >= 16 {
		return value[:16]
	}
	return value
}
func jsonPretty(value string) string {
	var data any
	if json.Unmarshal([]byte(value), &data) != nil {
		return value
	}
	pretty, _ := json.MarshalIndent(data, "", "  ")
	return string(pretty)
}
func shortText(value string, size int) string {
	if len(value) <= size {
		return value
	}
	return value[:size] + "…"
}
func statusClass(status any) string {
	s := fmt.Sprint(status)
	switch s {
	case "true", "active", "completed", "claimed", "used":
		return "badge badge-success"
	case "pending", "playing", "created", "unused":
		return "badge badge-warning"
	case "false", "disabled", "expired", "cancelled":
		return "badge badge-muted"
	default:
		return "badge badge-info"
	}
}
