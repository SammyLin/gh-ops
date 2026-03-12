package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/SammyLin/gh-ops/internal/actions"
	"github.com/SammyLin/gh-ops/internal/audit"
	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
	"github.com/SammyLin/gh-ops/internal/middleware"
)

// Run starts the gh-ops HTTP server.
func Run(configPath string, templateFS fs.FS) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	auditLogger, err := audit.New(cfg.Audit.DBPath)
	if err != nil {
		return fmt.Errorf("init audit log: %w", err)
	}
	defer func() {
		if err := auditLogger.Close(); err != nil {
			log.Printf("close audit logger: %v", err)
		}
	}()

	authHandler := auth.New(
		cfg.GitHub.ClientID,
		cfg.GitHub.ClientSecret,
		cfg.Server.BaseURL,
		cfg.Session.Secret,
	)

	tmpl, err := template.ParseFS(templateFS, "web/templates/*.html")
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	actionHandler := actions.NewHandler(auditLogger, tmpl, cfg.AllowedActions)

	r := chi.NewRouter()

	// Middleware stack
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
		MaxAge:         300,
	}))
	r.Use(middleware.RateLimit(60, time.Minute))

	// Public routes
	r.Get("/", homeHandler(tmpl, authHandler))
	r.Get("/health", healthHandler)
	r.Get("/auth/login", authHandler.LoginHandler)
	r.Get("/auth/callback", authHandler.CallbackHandler)
	r.Get("/auth/logout", authHandler.LogoutHandler)

	// Protected routes (require GitHub OAuth)
	r.Group(func(r chi.Router) {
		r.Use(authHandler.RequireAuth)
		r.Get("/action/{action}", actionHandler.HandleAction)
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("gh-ops starting on %s (%s)", addr, cfg.Server.BaseURL)
	return http.ListenAndServe(addr, r)
}

func homeHandler(tmpl *template.Template, authHandler *auth.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		data := map[string]string{}

		// Try to get user from session (non-blocking)
		if user := authHandler.UserFromRequest(r); user != "" {
			data["User"] = user
		}

		if err := tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}
