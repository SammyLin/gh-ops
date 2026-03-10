package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/google/go-github/v60/github"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

type contextKey string

const (
	sessionName  = "gh-ops"
	tokenKey     = "oauth_token"
	userKey      = "github_user"
	returnURLKey = "return_url"
	stateKey     = "oauth_state"

	ContextTokenKey contextKey = "token"
	ContextUserKey  contextKey = "user"
)

// Auth manages GitHub OAuth flow and session state.
type Auth struct {
	oauth *oauth2.Config
	store sessions.Store
}

// New creates an Auth handler with encrypted cookie sessions.
func New(clientID, clientSecret, baseURL, sessionSecret string) *Auth {
	authKey := sha256.Sum256([]byte("auth:" + sessionSecret))
	encKey := sha256.Sum256([]byte("enc:" + sessionSecret))

	store := sessions.NewCookieStore(authKey[:], encKey[:32])
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	return &Auth{
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{"repo"},
			Endpoint:     githubOAuth.Endpoint,
			RedirectURL:  baseURL + "/auth/callback",
		},
		store: store,
	}
}

// RequireAuth is middleware that redirects unauthenticated users to GitHub OAuth.
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := a.store.Get(r, sessionName)

		token, ok := session.Values[tokenKey].(string)
		if !ok || token == "" {
			session.Values[returnURLKey] = r.URL.String()
			_ = session.Save(r, w)
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		user, _ := session.Values[userKey].(string)

		ctx := context.WithValue(r.Context(), ContextTokenKey, token)
		ctx = context.WithValue(ctx, ContextUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoginHandler initiates the GitHub OAuth authorization flow.
func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.store.Get(r, sessionName)

	state := generateState()
	session.Values[stateKey] = state
	_ = session.Save(r, w)

	http.Redirect(w, r, a.oauth.AuthCodeURL(state), http.StatusFound)
}

// CallbackHandler exchanges the authorization code for a token and stores it.
func (a *Auth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.store.Get(r, sessionName)

	expectedState, ok := session.Values[stateKey].(string)
	if !ok || r.URL.Query().Get("state") != expectedState {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	delete(session.Values, stateKey)

	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "OAuth exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch authenticated GitHub user
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token.AccessToken})
	ghClient := github.NewClient(oauth2.NewClient(r.Context(), ts))

	ghUser, _, err := ghClient.Users.Get(r.Context(), "")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values[tokenKey] = token.AccessToken
	session.Values[userKey] = ghUser.GetLogin()

	returnURL := "/"
	if u, ok := session.Values[returnURLKey].(string); ok && u != "" {
		returnURL = u
		delete(session.Values, returnURLKey)
	}

	_ = session.Save(r, w)
	http.Redirect(w, r, returnURL, http.StatusFound)
}

// LogoutHandler clears the session.
func (a *Auth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.store.Get(r, sessionName)
	session.Options.MaxAge = -1
	_ = session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// TokenFromContext extracts the OAuth token from the request context.
func TokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextTokenKey).(string)
	return v
}

// UserFromContext extracts the GitHub username from the request context.
func UserFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextUserKey).(string)
	return v
}

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
