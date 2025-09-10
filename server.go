package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"
)

//go:embed static/style.css
var styleCSS string

//go:embed static/crypto.js
var cryptoJS string

type Server struct {
	db        *sql.DB
	tmplIndex *template.Template
	tmplShow  *template.Template
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// Inline CSS/JS for the tiny UI, so allow 'unsafe-inline' here.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		db:        db,
		tmplIndex: template.Must(template.New("index").Parse(indexHTML)),
		tmplShow:  template.Must(template.New("show").Parse(showHTML)),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// GUI
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /{id}", s.handleShow)

	// API
	mux.HandleFunc("POST /api/message", s.handleCreateMessage)
	mux.HandleFunc("GET /api/message/{id}", s.handleGetMessage)

	// Health
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return securityHeaders(mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// _ = s.tmplIndex.Execute(w, nil)
	_ = s.tmplIndex.Execute(w, struct {
		Style  template.CSS
		Script template.JS
	}{
		Style:  template.CSS(styleCSS),
		Script: template.JS(cryptoJS),
	})
}

func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "api" || id == "" {
		http.NotFound(w, r)
		return
	}
	// _ = s.tmplShow.Execute(w, struct{ ID string }{ID: id})
	_ = s.tmplShow.Execute(w, struct {
		Style  template.CSS
		Script template.JS
		ID     string
	}{
		Style:  template.CSS(styleCSS),
		Script: template.JS(cryptoJS),
		ID:     id,
	})
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB

	var req struct {
		Ciphertext string `json:"ciphertext"`
		Nonce      string `json:"nonce"`
		TTLSeconds int    `json:"ttl_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Ciphertext) == "" || strings.TrimSpace(req.Nonce) == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if ttl > 7*24*time.Hour {
		ttl = 7 * 24 * time.Hour
	}

	id, err := NewID()
	if err != nil {
		http.Error(w, "id generation failed", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(ttl).UTC()

	if err := s.dbInsertMessage(id, req.Ciphertext, req.Nonce, expiresAt); err != nil {
		http.Error(w, "storage failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          id,
		"view_url":    "/" + id,
		"expires_at":  expiresAt.Format(time.RFC3339),
		"ttl_seconds": int(ttl.Seconds()),
	})
}

func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	msg, err := s.dbGetMessage(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if time.Now().UTC().After(msg.ExpiresAt.UTC()) {
		_ = s.dbDeleteMessage(id)
		http.Error(w, "expired", http.StatusGone)
		return
	}

	// one-time: delete after fetch
	_ = s.dbDeleteMessage(id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"ciphertext": msg.Ciphertext,
		"nonce":      msg.Nonce,
		"filename":   msg.Filename,
	})
}

func (s *Server) StartCleanup() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for range t.C {
		s.dbCleanupExpired()
	}
}
