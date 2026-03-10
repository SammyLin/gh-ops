package cmd

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/SammyLin/gh-ops/internal/actions"
	"github.com/SammyLin/gh-ops/internal/audit"
	"github.com/SammyLin/gh-ops/internal/auth"
	"github.com/SammyLin/gh-ops/internal/config"
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
	defer auditLogger.Close()

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
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
		MaxAge:         300,
	}))
	r.Use(RateLimit(60, time.Minute))

	// Public routes
	r.Get("/", homeHandler)
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

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>gh-ops</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; background: #f6f8fa;
        }
        .card {
            background: #fff; border-radius: 12px; padding: 48px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06);
            text-align: center; max-width: 500px; width: 90%;
        }
        h1 { font-size: 28px; color: #1f2328; margin-bottom: 8px; font-weight: 700; }
        p { color: #656d76; font-size: 16px; }
    </style>
</head>
<body>
    <div class="card">
        <h1>gh-ops</h1>
        <p>GitHub Operations Web API</p>
    </div>
</body>
</html>`)
}
