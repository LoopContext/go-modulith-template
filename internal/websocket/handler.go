package websocket

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/gorilla/websocket"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub            *Hub
	verifier       authn.Verifier
	allowedOrigins []string
	env            string // "dev" or "prod" - affects security strictness
}

// HandlerConfig configures the WebSocket handler.
type HandlerConfig struct {
	Hub            *Hub
	Verifier       authn.Verifier
	AllowedOrigins []string
	Env            string // "dev" or "prod"
}

// NewHandler creates a new WebSocket HTTP handler with security features.
func NewHandler(cfg HandlerConfig) *Handler {
	// Create upgrader with origin checking
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     createOriginChecker(cfg.AllowedOrigins, cfg.Env),
	}

	// Store upgrader in handler for potential future use
	_ = upgrader

	return &Handler{
		hub:            cfg.Hub,
		verifier:       cfg.Verifier,
		allowedOrigins: cfg.AllowedOrigins,
		env:            cfg.Env,
	}
}

// createOriginChecker returns a function that validates WebSocket origin headers.
func createOriginChecker(allowedOrigins []string, env string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		// In development, allow all origins if none specified
		if env == envDev && len(allowedOrigins) == 0 {
			return true
		}

		// If no origins configured in prod, deny all (fail secure)
		if len(allowedOrigins) == 0 {
			slog.Warn("WebSocket connection rejected: no allowed origins configured",
				"origin", r.Header.Get("Origin"))

			return false
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			return checkEmptyOrigin(allowedOrigins)
		}

		return checkOriginMatch(origin, allowedOrigins)
	}
}

func checkEmptyOrigin(allowedOrigins []string) bool {
	// Some clients don't send Origin header (e.g., native apps)
	// Allow if explicitly configured with "*"
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
	}

	return false
}

func checkOriginMatch(origin string, allowedOrigins []string) bool {
	// Check against allowed origins
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}

		if strings.EqualFold(origin, allowed) {
			return true
		}
	}

	slog.Warn("WebSocket connection rejected: origin not allowed",
		"origin", origin,
		"allowed", allowedOrigins)

	return false
}

// ServeHTTP handles the WebSocket upgrade request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate and extract user ID
	userID, err := h.authenticateRequest(r)
	if err != nil {
		slog.Warn("WebSocket authentication failed",
			"error", err,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)

		return
	}

	// Create upgrader with origin checking
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     createOriginChecker(h.allowedOrigins, h.env),
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)

		return
	}

	client := NewClient(h.hub, conn, userID)
	h.hub.register <- client

	slog.Info("WebSocket connection established",
		"user_id", userID,
		"remote_addr", r.RemoteAddr)

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}

// authenticateRequest authenticates the WebSocket request and returns the user ID.
// It supports multiple authentication methods:
// 1. JWT token from query parameter (token=...)
// 2. JWT token from cookie (auth_token)
// 3. JWT token from Authorization header (Bearer ...)
// 4. In dev mode: user_id query parameter (for testing)
func (h *Handler) authenticateRequest(r *http.Request) (string, error) {
	// In development mode, allow user_id query parameter for testing
	if h.env == envDev {
		userID := r.URL.Query().Get("user_id")
		if userID != "" {
			return userID, nil
		}
	}

	token := h.extractToken(r)
	if token == "" {
		return h.handleNoToken()
	}

	return h.verifyToken(r, token)
}

func (h *Handler) extractToken(r *http.Request) string {
	// 1. Try Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 2. Try query parameter
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	// 3. Try cookie
	cookie, err := r.Cookie("auth_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

func (h *Handler) handleNoToken() (string, error) {
	if h.verifier == nil {
		// In dev mode without verifier, allow anonymous (for testing)
		if h.env == envDev {
			return "", nil
		}

		return "", http.ErrNoCookie
	}

	return "", http.ErrNoCookie
}

func (h *Handler) verifyToken(r *http.Request, token string) (string, error) {
	if h.verifier == nil {
		// No verifier configured - deny in production
		if h.env != envDev {
			return "", http.ErrNoCookie
		}

		// Dev mode without verifier - allow (for local testing)
		return "", nil
	}

	claims, err := h.verifier.VerifyToken(r.Context(), token)
	if err != nil {
		return "", fmt.Errorf("failed to verify token: %w", err)
	}

	return claims.UserID, nil
}
