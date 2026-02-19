package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"html/template"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed static/style.css
var styleCSS string

//go:embed static/crypto.js
var cryptoJS string

type Server struct {
	db             *sql.DB
	defaultTTL     int
	maxUpload      int64
	tmplIndex      *template.Template
	tmplShow       *template.Template
	anonymizeIP    bool
	TrustedProxies []*net.IPNet
	logger         *slog.Logger
}

type TTLOption struct {
	Value int
	Label string
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

func NewServer(db *sql.DB, config *Config, logger *slog.Logger) *Server {
	var trustedNets []*net.IPNet
	for _, cidr := range config.TrustedProxies {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Fatalf("invalid CIDR in trusted proxies: %s", cidr)
		}
		trustedNets = append(trustedNets, ipNet)
	}
	return &Server{
		db:             db,
		defaultTTL:     config.DefaultTTL,
		maxUpload:      config.MaxUploadSize,
		anonymizeIP:    config.AnonymizeIP,
		TrustedProxies: trustedNets,
		tmplIndex:      template.Must(template.New("index").Parse(indexHTML)),
		tmplShow:       template.Must(template.New("show").Parse(showHTML)),
		logger:         logger,
	}
}

func getClientIP(r *http.Request, trustedProxies []*net.IPNet, anonym bool) string {
	remoteIPStr, _, _ := net.SplitHostPort(r.RemoteAddr)
	remoteIP := net.ParseIP(remoteIPStr)

	finalIP := remoteIPStr

	// Trusted Proxy Check (IPv4 & IPv6) - Check if the remote IP is in any of the trusted proxy ranges
	if len(trustedProxies) > 0 {
		isTrusted := false
		for _, ipNet := range trustedProxies {
			if ipNet.Contains(remoteIP) {
				isTrusted = true
				break
			}
		}

		if isTrusted {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				ips := strings.Split(xff, ",")
				finalIP = strings.TrimSpace(ips[0])
			}
		}
	}

	// Anonymization
	if anonym {
		parsedFinal := net.ParseIP(finalIP)
		if parsedFinal == nil {
			return "0.0.0.0"
		}

		if ipv4 := parsedFinal.To4(); ipv4 != nil {
			return ipv4.Mask(net.CIDRMask(24, 32)).String()
		}

		return parsedFinal.Mask(net.CIDRMask(64, 128)).String()
	}

	return finalIP
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
	err := s.tmplIndex.Execute(w, struct {
		Style       template.CSS
		Script      template.JS
		TTLOptions  []TTLOption
		SelectedTTL int
	}{
		Style:  template.CSS(styleCSS),
		Script: template.JS(cryptoJS),
		TTLOptions: []TTLOption{
			{Value: 3600, Label: "1 hour"},
			{Value: 86400, Label: "1 day"},
			{Value: 604800, Label: "1 week"},
		},
		SelectedTTL: s.defaultTTL,
	})
	if err != nil {
		s.logger.Error("template error", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	// log request path for debugging with ip and user agent
	s.logger.Info("http request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("ip", getClientIP(r, s.TrustedProxies, s.anonymizeIP)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int("status", 200), // Falls du den Status trackst
	)
}

func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "api" || id == "" {
		http.NotFound(w, r)
		return
	}

	err := s.tmplShow.Execute(w, struct {
		Style  template.CSS
		Script template.JS
		ID     string
	}{
		Style:  template.CSS(styleCSS),
		Script: template.JS(cryptoJS),
		ID:     id,
	})
	if err != nil {
		s.logger.Error("template error", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	// log request path for debugging with ip and user agent
	s.logger.Info("http request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("ip", getClientIP(r, s.TrustedProxies, s.anonymizeIP)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int("status", 200), // Falls du den Status trackst
	)
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	// r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload)

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
	// log request path for debugging with ip and user agent
	s.logger.Info("http request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("ip", getClientIP(r, s.TrustedProxies, s.anonymizeIP)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int("status", 200), // Falls du den Status trackst
	)
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
	// log request path for debugging with ip and user agent
	s.logger.Info("http request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("ip", getClientIP(r, s.TrustedProxies, s.anonymizeIP)),
		slog.String("user_agent", r.UserAgent()),
		slog.Int("status", 200), // Falls du den Status trackst
	)
}

func (s *Server) StartCleanup() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for range t.C {
		s.dbCleanupExpired()
	}
}
