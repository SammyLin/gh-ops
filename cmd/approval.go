package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/SammyLin/gh-ops/internal/config"
)

type approvalResult struct {
	confirmed bool
}

func waitForApproval(cfg *config.ResolvedConfig, actionName string, params map[string]string, ghUser string) (bool, error) {
	token, err := generateToken()
	if err != nil {
		return false, fmt.Errorf("failed to generate approval token: %w", err)
	}

	ch := make(chan approvalResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/confirm", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != token {
			http.Error(w, "Invalid token", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodGet {
			renderApprovalPage(w, actionName, params, ghUser, token)
			return
		}
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprint(w, renderSuccessPage())
			ch <- approvalResult{confirmed: true}
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		return false, fmt.Errorf("failed to start approval server: %w", err)
	}

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()

	approvalURL := fmt.Sprintf("%s/confirm?token=%s", strings.TrimRight(cfg.Server.BaseURL, "/"), token)

	if jsonOutput {
		emitJSON(JSONEvent{Event: "approval_required", ApprovalURL: approvalURL})
	} else {
		fmt.Printf("\n  Confirm this action: %s\n\n", approvalURL)
	}

	result := <-ch
	_ = server.Close()

	return result.confirmed, nil
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const pageStyle = `
:root {
  --cream-100: #faf8f5;
  --cream-200: #f5f1eb;
  --cream-300: #ebe6dc;
  --ink-500: #6b6b6b;
  --ink-600: #434343;
  --ink-900: #1a1a1a;
  --border: #ebe6dc;
  --accent: #ca8a04;
  --accent-hover: #a16207;
  --highlight: #fbbf24;
  --success: #10B981;
  --warn: #f59e0b;
}
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif; background:var(--cream-100); color:var(--ink-900); min-height:100vh; display:flex; flex-direction:column; -webkit-text-size-adjust:100%%; }
header { background:#fff; border-bottom:1px solid var(--border); position:sticky; top:0; z-index:50; box-shadow:0 1px 3px rgba(0,0,0,0.04); }
.container { max-width:1024px; margin:0 auto; padding:0 20px; }
.header-inner { display:flex; justify-content:space-between; align-items:center; padding:14px 0; }
.logo { display:flex; align-items:center; gap:10px; }
.logo-icon { width:36px; height:36px; background:linear-gradient(135deg, var(--accent), var(--highlight)); border-radius:10px; display:flex; align-items:center; justify-content:center; color:#fff; flex-shrink:0; }
.logo-icon svg { width:20px; height:20px; }
.logo h1 { font-size:1.25rem; font-weight:700; color:var(--ink-900); }
.logo p { font-size:0.75rem; color:var(--ink-600); display:none; }
.version { font-size:0.75rem; color:var(--ink-600); background:var(--cream-200); padding:4px 8px; border-radius:6px; white-space:nowrap; }
main { flex:1; display:flex; justify-content:center; align-items:flex-start; padding:24px 16px; }
.card { background:#fff; border:1px solid var(--border); border-radius:20px; box-shadow:0 2px 8px rgba(0,0,0,0.06); padding:32px 24px; width:100%%; max-width:480px; }
.card-icon { display:flex; justify-content:center; margin-bottom:20px; }
.card-icon-circle { width:64px; height:64px; border-radius:50%%; display:flex; align-items:center; justify-content:center; }
.card-icon-circle.warn { background:#fef3c7; }
.card-icon-circle.warn svg { width:32px; height:32px; }
.card-icon-circle.success { background:#d1fae5; }
.card-icon-circle.success svg { width:32px; height:32px; }
.card-title { text-align:center; font-size:1.5rem; font-weight:700; margin-bottom:6px; }
.card-sub { text-align:center; font-size:1rem; color:var(--ink-500); margin-bottom:8px; }
.card-sub strong { color:var(--ink-900); }
.card-desc { text-align:center; font-size:0.875rem; color:var(--ink-500); margin-bottom:28px; line-height:1.5; }
.detail-box { background:var(--cream-100); border:1px solid var(--border); border-radius:14px; padding:4px 0; margin-bottom:28px; overflow:hidden; }
.detail-row { display:flex; justify-content:space-between; align-items:center; padding:14px 20px; font-size:0.9375rem; border-bottom:1px solid var(--cream-300); }
.detail-row:last-child { border-bottom:none; }
.detail-row .label { font-weight:600; color:var(--ink-500); font-size:0.8125rem; text-transform:uppercase; letter-spacing:0.02em; }
.detail-row .value { color:var(--ink-900); font-family:"SF Mono",Menlo,monospace; font-size:0.875rem; font-weight:500; text-align:right; word-break:break-all; }
.btn-confirm { display:block; width:100%%; padding:16px; background:var(--accent); color:#fff; border:none; border-radius:14px; font-size:1.125rem; font-weight:600; cursor:pointer; transition:all 0.2s; box-shadow:0 4px 12px rgba(202,138,4,0.3); -webkit-tap-highlight-color:transparent; }
.btn-confirm:hover { background:var(--accent-hover); transform:translateY(-2px); box-shadow:0 6px 16px rgba(202,138,4,0.35); }
.btn-confirm:active { transform:translateY(0); box-shadow:0 2px 6px rgba(202,138,4,0.2); }
footer { background:#fff; border-top:1px solid var(--border); padding:20px 16px; text-align:center; margin-top:auto; }
footer p { font-size:0.75rem; color:var(--ink-500); }
@media (min-width:480px) {
  .logo p { display:block; }
  .logo-icon { width:40px; height:40px; border-radius:12px; }
  .logo-icon svg { width:24px; height:24px; }
  .logo h1 { font-size:1.5rem; }
  main { padding:48px 24px; }
  .card { padding:40px 36px; }
  .card-title { font-size:1.75rem; }
}
`

func renderHeader() string {
	return fmt.Sprintf(`<header>
  <div class="container">
    <div class="header-inner">
      <div class="logo">
        <div class="logo-icon">
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
        </div>
        <div>
          <h1>gh-ops</h1>
          <p>One-click GitHub operations via OAuth</p>
        </div>
      </div>
      <div class="version">%s</div>
    </div>
  </div>
</header>`, Version)
}

const footerHTML = `<footer>
  <div class="container">
    <p>&copy; 2026 gh-ops by Sammy Lin. All rights reserved.</p>
  </div>
</footer>`

func renderApprovalPage(w http.ResponseWriter, actionName string, params map[string]string, ghUser, token string) {
	var detailRows strings.Builder
	for k, v := range params {
		if v != "" {
			fmt.Fprintf(&detailRows, `<div class="detail-row"><span class="label">%s</span><span class="value">%s</span></div>`, k, v)
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gh-ops — Confirm Action</title>
<style>%s</style>
</head>
<body>
%s
<main>
  <div class="card">
    <div class="card-icon">
      <div class="card-icon-circle warn">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#d97706" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 9v4"/><path d="M10.363 3.591l-8.106 13.534a1.914 1.914 0 001.636 2.871h16.214a1.914 1.914 0 001.636-2.871L13.637 3.591a1.914 1.914 0 00-3.274 0z"/><path d="M12 17h.01"/></svg>
      </div>
    </div>
    <div class="card-title">Confirm Action</div>
    <div class="card-sub">as <strong>%s</strong></div>
    <div class="card-desc">Review the details below and confirm to execute this GitHub operation.</div>
    <div class="detail-box">
      <div class="detail-row"><span class="label">Action</span><span class="value">%s</span></div>
      %s
    </div>
    <form method="POST" action="/confirm?token=%s">
      <button class="btn-confirm" type="submit">Confirm &amp; Execute</button>
    </form>
  </div>
</main>
%s
</body>
</html>`, pageStyle, renderHeader(), ghUser, actionName, detailRows.String(), token, footerHTML)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, html)
}

func renderSuccessPage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gh-ops — Action Confirmed</title>
<style>%s</style>
</head>
<body>
%s
<main>
  <div class="card">
    <div class="card-icon">
      <div class="card-icon-circle success">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#059669" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
      </div>
    </div>
    <div class="card-title" style="color:#059669;">Action Confirmed</div>
    <div class="card-sub">Executing... you can close this page.</div>
  </div>
</main>
%s
</body>
</html>`, pageStyle, renderHeader(), footerHTML)
}
