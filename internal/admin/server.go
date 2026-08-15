package admin

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	store    *Store
	views    *Views
	sessions *SessionManager
	logger   *log.Logger
}

func NewServer(store *Store, views *Views, sessions *SessionManager, logger *log.Logger) *Server {
	return &Server{store: store, views: views, sessions: sessions, logger: logger}
}

func (s *Server) Routes(static, public http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", static))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/play/festival-hadiah-ceria/wheel", http.StatusSeeOther)
	})
	mux.Handle("/play/", public)
	mux.Handle("/api/", public)
	mux.HandleFunc("GET /admin/login", s.loginPage)
	mux.HandleFunc("POST /admin/login", s.login)
	mux.HandleFunc("GET /admin/setup", s.setupPage)
	mux.HandleFunc("POST /admin/setup", s.setup)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/admin/", http.StatusSeeOther) })
	protected.HandleFunc("GET /admin/", s.dashboard)
	protected.HandleFunc("POST /admin/logout", s.logout)
	protected.HandleFunc("GET /admin/game-types", s.gameTypes)
	protected.HandleFunc("POST /admin/game-types/save", s.saveGameType)
	protected.HandleFunc("GET /admin/campaigns", s.campaigns)
	protected.HandleFunc("POST /admin/campaigns/save", s.saveCampaign)
	protected.HandleFunc("GET /admin/prizes", s.prizes)
	protected.HandleFunc("POST /admin/prizes/save", s.savePrize)
	protected.HandleFunc("GET /admin/access-codes", s.accessCodes)
	protected.HandleFunc("POST /admin/access-codes/add", s.addAccessCodes)
	protected.HandleFunc("POST /admin/access-codes/{id}/status", s.setAccessCodeStatus)
	protected.HandleFunc("GET /admin/sessions", s.gameSessions)
	protected.HandleFunc("GET /admin/results", s.results)
	protected.HandleFunc("POST /admin/results/{id}/status", s.setClaimStatus)
	protected.HandleFunc("GET /admin/admins", s.admins)
	protected.HandleFunc("POST /admin/admins/create", s.createAdmin)
	protected.HandleFunc("POST /admin/admins/{id}/active", s.setAdminActive)
	mux.Handle("/admin", s.sessions.RequireAuth(protected))
	mux.Handle("/admin/", s.sessions.RequireAuth(protected))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; font-src 'self'; form-action 'self'; frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/admin") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

type loginData struct{ Setup bool }

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.sessions.FromRequest(r); ok {
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
		return
	}
	count, err := s.store.AdminCount(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if count == 0 {
		http.Redirect(w, r, "/admin/setup", http.StatusSeeOther)
		return
	}
	s.views.RenderLogin(w, ViewData{PageTitle: "Login Admin", Error: r.URL.Query().Get("error"), Notice: r.URL.Query().Get("notice"), Data: loginData{}})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil {
		redirectMessage(w, r, "/admin/login", "error", "Form tidak valid")
		return
	}
	user, err := s.store.Authenticate(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		time.Sleep(400 * time.Millisecond)
		redirectMessage(w, r, "/admin/login", "error", err.Error())
		return
	}
	if _, err := s.sessions.Create(w, user); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) setupPage(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.AdminCount(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if count > 0 {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}
	s.views.RenderLogin(w, ViewData{PageTitle: "Buat Admin Pertama", Error: r.URL.Query().Get("error"), Data: loginData{Setup: true}})
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.AdminCount(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if count > 0 {
		http.Error(w, "Setup sudah ditutup", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil {
		redirectMessage(w, r, "/admin/setup", "error", "Form tidak valid")
		return
	}
	password := r.FormValue("password")
	if password != r.FormValue("password_confirmation") {
		redirectMessage(w, r, "/admin/setup", "error", "Konfirmasi password tidak sama")
		return
	}
	if err := s.store.CreateAdmin(r.Context(), r.FormValue("username"), password); err != nil {
		redirectMessage(w, r, "/admin/setup", "error", friendlyDBError(err).Error())
		return
	}
	user, err := s.store.Authenticate(r.Context(), r.FormValue("username"), password)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if _, err := s.sessions.Create(w, user); err != nil {
		s.internalError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Destroy(w, r)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

type dashboardData struct {
	Stats     DashboardStats
	Results   []GameResult
	Campaigns []Campaign
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	stats, results, err := s.store.Dashboard(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	campaigns, err := s.store.Campaigns(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.views.Render(w, "dashboard", viewData(r, "Dashboard", "dashboard", dashboardData{Stats: stats, Results: results, Campaigns: campaigns}))
}

type gameTypesData struct{ Items []GameType }

func (s *Server) gameTypes(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.GameTypes(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.views.Render(w, "game_types", viewData(r, "Jenis Game", "game-types", gameTypesData{items}))
}
func (s *Server) saveGameType(w http.ResponseWriter, r *http.Request) {
	item := GameType{Code: r.FormValue("code"), Name: r.FormValue("name"), FrontendModule: r.FormValue("frontend_module"), IsActive: formBool(r, "is_active")}
	if err := s.store.SaveGameType(r.Context(), item); err != nil {
		redirectMessage(w, r, "/admin/game-types", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/game-types", "notice", "Jenis game berhasil disimpan")
}

type campaignsData struct {
	Items     []Campaign
	GameTypes []GameType
	Edit      Campaign
}

func (s *Server) campaigns(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Campaigns(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	types, err := s.store.GameTypes(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	var edit Campaign
	if id := queryID(r, "edit"); id > 0 {
		edit, err = s.store.Campaign(r.Context(), id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.internalError(w, err)
			return
		}
	}
	if edit.GameConfig == "" {
		edit.GameConfig = "{}"
	}
	s.views.Render(w, "campaigns", viewData(r, "Campaign", "campaigns", campaignsData{items, types, edit}))
}
func (s *Server) saveCampaign(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectMessage(w, r, "/admin/campaigns", "error", "Form campaign tidak valid")
		return
	}
	item := Campaign{ID: formID(r, "id"), GameCodes: r.Form["game_type_codes"], Name: r.FormValue("name"), Slug: r.FormValue("slug"), GameConfig: r.FormValue("game_config"), StartsAt: r.FormValue("starts_at"), EndsAt: r.FormValue("ends_at"), IsActive: formBool(r, "is_active")}
	if err := s.store.SaveCampaign(r.Context(), item); err != nil {
		redirectMessage(w, r, "/admin/campaigns", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/campaigns", "notice", "Campaign berhasil disimpan")
}

type prizesData struct {
	Items      []Prize
	Campaigns  []Campaign
	Edit       Prize
	CampaignID int64
}

func (s *Server) prizes(w http.ResponseWriter, r *http.Request) {
	campaignID := queryID(r, "campaign")
	items, err := s.store.Prizes(r.Context(), campaignID)
	if err != nil {
		s.internalError(w, err)
		return
	}
	campaigns, err := s.store.Campaigns(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	var edit Prize
	if id := queryID(r, "edit"); id > 0 {
		edit, err = s.store.Prize(r.Context(), id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.internalError(w, err)
			return
		}
	} else {
		edit.Weight = 1
		edit.RequiresClaim = true
		edit.IsActive = true
		edit.CampaignID = campaignID
	}
	s.views.Render(w, "prizes", viewData(r, "Hadiah", "prizes", prizesData{items, campaigns, edit, campaignID}))
}
func (s *Server) savePrize(w http.ResponseWriter, r *http.Request) {
	item := Prize{ID: formID(r, "id"), CampaignID: formID(r, "campaign_id"), Name: r.FormValue("name"), Description: r.FormValue("description"), ImagePath: r.FormValue("existing_image_path"), Color: r.FormValue("color"), Weight: formFloat(r, "weight"), InitialStock: formInt(r, "initial_stock"), RemainingStock: formInt(r, "remaining_stock"), IsUnlimited: formBool(r, "is_unlimited"), RequiresClaim: formBool(r, "requires_claim"), DisplayOrder: formInt(r, "display_order"), IsActive: formBool(r, "is_active")}
	var uploadedDiskPath string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		item.ImagePath, uploadedDiskPath, err = savePrizeImage(file, header, item.Name)
		if err != nil {
			redirectMessage(w, r, "/admin/prizes", "error", err.Error())
			return
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		redirectMessage(w, r, "/admin/prizes", "error", "Upload gambar tidak dapat diproses")
		return
	}
	if err := s.store.SavePrize(r.Context(), item); err != nil {
		if uploadedDiskPath != "" {
			_ = os.Remove(uploadedDiskPath)
		}
		redirectMessage(w, r, "/admin/prizes", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/prizes", "notice", "Hadiah berhasil disimpan")
}

type accessCodesData struct {
	Items      []AccessCode
	Campaigns  []Campaign
	CampaignID int64
}

func (s *Server) accessCodes(w http.ResponseWriter, r *http.Request) {
	campaignID := queryID(r, "campaign")
	items, err := s.store.AccessCodes(r.Context(), campaignID)
	if err != nil {
		s.internalError(w, err)
		return
	}
	campaigns, err := s.store.Campaigns(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.views.Render(w, "access_codes", viewData(r, "Kode Akses", "access-codes", accessCodesData{items, campaigns, campaignID}))
}
func (s *Server) addAccessCodes(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.AddAccessCodes(r.Context(), formID(r, "campaign_id"), r.FormValue("codes"))
	if err != nil {
		redirectMessage(w, r, "/admin/access-codes", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/access-codes", "notice", strconv.Itoa(count)+" kode berhasil ditambahkan")
}
func (s *Server) setAccessCodeStatus(w http.ResponseWriter, r *http.Request) {
	if err := s.store.SetAccessCodeStatus(r.Context(), pathID(r), r.FormValue("status")); err != nil {
		redirectMessage(w, r, "/admin/access-codes", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/access-codes", "notice", "Status kode diperbarui")
}

type sessionsData struct{ Items []GameSession }

func (s *Server) gameSessions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Sessions(r.Context(), 500)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.views.Render(w, "sessions", viewData(r, "Sesi Permainan", "sessions", sessionsData{items}))
}

type resultsData struct{ Items []GameResult }

func (s *Server) results(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Results(r.Context(), 500)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.views.Render(w, "results", viewData(r, "Hasil & Klaim", "results", resultsData{items}))
}
func (s *Server) setClaimStatus(w http.ResponseWriter, r *http.Request) {
	if err := s.store.SetClaimStatus(r.Context(), pathID(r), r.FormValue("status")); err != nil {
		redirectMessage(w, r, "/admin/results", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/results", "notice", "Status klaim diperbarui")
}

type adminsData struct{ Items []AdminUser }

func (s *Server) admins(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Admins(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.views.Render(w, "admins", viewData(r, "Admin", "admins", adminsData{items}))
}
func (s *Server) createAdmin(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("password") != r.FormValue("password_confirmation") {
		redirectMessage(w, r, "/admin/admins", "error", "Konfirmasi password tidak sama")
		return
	}
	if err := s.store.CreateAdmin(r.Context(), r.FormValue("username"), r.FormValue("password")); err != nil {
		redirectMessage(w, r, "/admin/admins", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/admins", "notice", "Admin baru berhasil dibuat")
}
func (s *Server) setAdminActive(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	active := r.FormValue("active") == "1"
	if id == currentAdmin(r).AdminID && !active {
		redirectMessage(w, r, "/admin/admins", "error", "Anda tidak dapat menonaktifkan akun yang sedang digunakan")
		return
	}
	if err := s.store.SetAdminActive(r.Context(), id, active); err != nil {
		redirectMessage(w, r, "/admin/admins", "error", friendlyDBError(err).Error())
		return
	}
	redirectMessage(w, r, "/admin/admins", "notice", "Status admin diperbarui")
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.logger.Printf("admin error: %v", err)
	http.Error(w, "Terjadi kesalahan internal", http.StatusInternalServerError)
}
func formBool(r *http.Request, name string) bool {
	return r.FormValue(name) == "1" || r.FormValue(name) == "on" || r.FormValue(name) == "true"
}
func formID(r *http.Request, name string) int64 {
	value, _ := strconv.ParseInt(r.FormValue(name), 10, 64)
	return value
}
func formInt(r *http.Request, name string) int {
	value, _ := strconv.Atoi(r.FormValue(name))
	return value
}
func formFloat(r *http.Request, name string) float64 {
	value, _ := strconv.ParseFloat(r.FormValue(name), 64)
	return value
}
func queryID(r *http.Request, name string) int64 {
	value, _ := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	return value
}
func pathID(r *http.Request) int64 {
	value, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return value
}
