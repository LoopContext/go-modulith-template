package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

// Handler handles OAuth HTTP requests.
type Handler struct {
	registry     *Registry
	stateManager *StateManager
	stateStore   StateStore
	onComplete   CompleteCallback
}

// StateStore defines the interface for persisting OAuth state tokens.
type StateStore interface {
	SaveState(ctx context.Context, data *StateData) error
	GetState(ctx context.Context, state string) (*StateData, error)
	DeleteState(ctx context.Context, state string) error
}

// CompleteCallback is called when OAuth flow completes successfully.
type CompleteCallback func(ctx context.Context, userInfo UserInfo, stateData *StateData) (*Result, error)

// Result contains the result of a successful OAuth flow.
type Result struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IsNewUser    bool   `json:"is_new_user"`
	UserID       string `json:"user_id"`
}

// NewHandler creates a new OAuth handler.
func NewHandler(registry *Registry, stateManager *StateManager, stateStore StateStore, onComplete CompleteCallback) *Handler {
	return &Handler{
		registry:     registry,
		stateManager: stateManager,
		stateStore:   stateStore,
		onComplete:   onComplete,
	}
}

// BeginAuth starts the OAuth flow for a provider.
func (h *Handler) BeginAuth(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if provider == "" {
		provider = r.URL.Query().Get("provider")
	}

	if provider == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)

		return
	}

	if !h.registry.IsProviderEnabled(provider) {
		http.Error(w, "provider not enabled", http.StatusBadRequest)

		return
	}

	// Get redirect URL from query
	redirectURL := r.URL.Query().Get("redirect_url")

	// Get user ID from context if this is a linking operation
	userID := getUserIDFromContext(r.Context())

	action := ActionLogin
	if userID != "" {
		action = ActionLink
	}

	// Create state data
	stateData, err := h.stateManager.CreateStateData(provider, redirectURL, userID, action)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to create state", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	// Save state to store
	if err := h.stateStore.SaveState(r.Context(), stateData); err != nil {
		slog.ErrorContext(r.Context(), "Failed to save state", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	// Set state in query for gothic
	q := r.URL.Query()
	q.Set("state", stateData.State)
	r.URL.RawQuery = q.Encode()

	// Add provider to request context for gothic
	r = r.WithContext(context.WithValue(r.Context(), gothic.ProviderParamKey, provider))

	// Begin OAuth flow
	gothic.BeginAuthHandler(w, r)
}

// Callback handles the OAuth callback from the provider.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	// Get state from query
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "state is required", http.StatusBadRequest)

		return
	}

	// Validate state token
	if !h.stateManager.ValidateState(state) {
		http.Error(w, "invalid state", http.StatusBadRequest)

		return
	}

	// Get state data from store
	stateData, err := h.stateStore.GetState(r.Context(), state)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to get state", "error", err)
		http.Error(w, "invalid or expired state", http.StatusBadRequest)

		return
	}

	// Delete state after use (one-time use)
	defer func() {
		if delErr := h.stateStore.DeleteState(r.Context(), state); delErr != nil {
			slog.WarnContext(r.Context(), "Failed to delete state", "error", delErr)
		}
	}()

	// Complete OAuth flow with gothic
	r = r.WithContext(context.WithValue(r.Context(), gothic.ProviderParamKey, stateData.Provider))

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to complete OAuth", "error", err)
		h.redirectWithError(w, r, stateData.RedirectURL, "oauth_failed", err.Error())

		return
	}

	// Convert goth user to our UserInfo
	userInfo := FromGothUser(gothUser)

	// Call the completion callback
	result, err := h.onComplete(r.Context(), userInfo, stateData)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to complete OAuth callback", "error", err)
		h.redirectWithError(w, r, stateData.RedirectURL, "auth_failed", err.Error())

		return
	}

	// Redirect with tokens
	h.redirectWithSuccess(w, r, stateData.RedirectURL, result)
}

// GetProviders returns the list of enabled OAuth providers.
func (h *Handler) GetProviders(w http.ResponseWriter, _ *http.Request) {
	providers := h.registry.GetEnabledProviders()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": providers,
	}); err != nil {
		slog.Error("Failed to encode providers response", "error", err)
	}
}

// redirectWithError redirects to the redirect URL with error parameters.
func (h *Handler) redirectWithError(w http.ResponseWriter, r *http.Request, redirectURL, errorCode, errorMsg string) {
	if redirectURL == "" {
		http.Error(w, fmt.Sprintf("%s: %s", errorCode, errorMsg), http.StatusBadRequest)

		return
	}

	// Append error to redirect URL
	redirectURL = fmt.Sprintf("%s?error=%s&error_description=%s", redirectURL, errorCode, errorMsg)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// redirectWithSuccess redirects to the redirect URL with tokens.
func (h *Handler) redirectWithSuccess(w http.ResponseWriter, r *http.Request, redirectURL string, result *Result) {
	if redirectURL == "" {
		// Return JSON response if no redirect URL
		w.Header().Set("Content-Type", "application/json")

		//nolint:gosec
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Error("Failed to encode result", "error", err)
		}

		return
	}

	// Redirect with tokens in query params (for mobile apps, etc.)
	redirectURL = fmt.Sprintf("%s?access_token=%s&refresh_token=%s&expires_in=%d&is_new_user=%t",
		redirectURL, result.AccessToken, result.RefreshToken, result.ExpiresIn, result.IsNewUser)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// RegisterRoutes registers the OAuth HTTP routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/auth/oauth/providers", h.GetProviders)
	mux.HandleFunc("GET /v1/auth/oauth/{provider}/start", h.BeginAuth)
	mux.HandleFunc("POST /v1/auth/oauth/{provider}/start", h.BeginAuth)
	mux.HandleFunc("GET /v1/auth/oauth/callback", h.Callback)
	mux.HandleFunc("GET /v1/auth/oauth/{provider}/link", h.BeginAuth) // Same handler, but requires auth
}

// getUserIDFromContext extracts user ID from context (set by auth middleware).
func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}

	return ""
}

// GetGothProvider returns the goth provider by name (exposed for testing).
func GetGothProvider(name string) (goth.Provider, error) {
	provider, err := goth.GetProvider(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get goth provider %s: %w", name, err)
	}

	return provider, nil
}
