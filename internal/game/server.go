package game

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
)

type Server struct {
	store  *Store
	page   *template.Template
	logger *log.Logger
}

type pageData struct{ Campaign Campaign }

func NewServer(store *Store, logger *log.Logger) (*Server, error) {
	page, err := template.ParseFiles("templates/game/wheel.html")
	if err != nil {
		return nil, err
	}
	return &Server{store: store, page: page, logger: logger}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /play/{slug}", s.wheelPage)
	mux.HandleFunc("GET /api/campaign/{slug}", s.campaign)
	mux.HandleFunc("POST /api/game/session", s.createSession)
	mux.HandleFunc("POST /api/game/play", s.play)
	return mux
}

func (s *Server) wheelPage(w http.ResponseWriter, r *http.Request) {
	campaign, err := s.store.Campaign(r.Context(), r.PathValue("slug"))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	if campaign.GameType != "wheel" {
		http.Error(w, "Tampilan game ini belum tersedia", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.page.ExecuteTemplate(w, "wheel.html", pageData{campaign}); err != nil {
		s.logger.Printf("render wheel: %v", err)
	}
}

func (s *Server) campaign(w http.ResponseWriter, r *http.Request) {
	campaign, err := s.store.Campaign(r.Context(), r.PathValue("slug"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		s.apiError(w, err)
		return
	}
	campaign.Config = map[string]any{}
	if err := json.Unmarshal([]byte(campaign.GameConfig), &campaign.Config); err != nil {
		s.apiError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaign)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CampaignSlug string `json:"campaignSlug"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "request tidak valid")
		return
	}
	token, err := s.store.CreateSession(r.Context(), input.CampaignSlug)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		s.apiError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"sessionToken": token})
}

func (s *Server) play(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "request tidak valid")
		return
	}
	result, err := s.store.Play(r.Context(), input.SessionToken)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) apiError(w http.ResponseWriter, err error) {
	s.logger.Printf("game api: %v", err)
	writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
}
func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.logger.Printf("game page: %v", err)
	http.Error(w, "Terjadi kesalahan internal", http.StatusInternalServerError)
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
