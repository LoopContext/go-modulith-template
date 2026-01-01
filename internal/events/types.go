// Package events provides typed event constants for the event bus.
package events

// Event type constants for type-safe event publishing and subscription.
// Each module should define its own event constants here.
const (
	// Auth module events
	EventAuthMagicCodeRequested = "auth.magic_code_requested"
	EventAuthUserCreated        = "auth.user.created"
	EventAuthUserLoggedIn       = "auth.user.logged_in"
	EventAuthUserLoggedOut      = "auth.user.logged_out"
	EventAuthSessionCreated     = "auth.session.created"
	EventAuthSessionRevoked     = "auth.session.revoked"
	EventAuthProfileUpdated     = "auth.profile.updated"
	EventAuthEmailChanged       = "auth.email.changed"
	EventAuthPhoneChanged       = "auth.phone.changed"

	// OAuth events
	EventOAuthAccountLinked   = "auth.oauth.account_linked"
	EventOAuthAccountUnlinked = "auth.oauth.account_unlinked"

	// Generic user events (can be used by any module)
	EventUserCreated = "user.created"
	EventUserUpdated = "user.updated"
	EventUserDeleted = "user.deleted"
)

// EventPayload provides type-safe payload construction helpers.
type EventPayload map[string]interface{}

// NewUserCreatedPayload creates a payload for user.created events.
func NewUserCreatedPayload(userID, email string) EventPayload {
	return EventPayload{
		"user_id": userID,
		"email":   email,
	}
}

// NewMagicCodeRequestedPayload creates a payload for auth.magic_code_requested events.
func NewMagicCodeRequestedPayload(email, phone, code string) EventPayload {
	payload := EventPayload{}

	if email != "" {
		payload["email"] = email
	}

	if phone != "" {
		payload["phone"] = phone
	}

	payload["code"] = code

	return payload
}

// NewSessionCreatedPayload creates a payload for auth.session.created events.
func NewSessionCreatedPayload(userID, sessionID string) EventPayload {
	return EventPayload{
		"user_id":    userID,
		"session_id": sessionID,
	}
}

// NewProfileUpdatedPayload creates a payload for auth.profile.updated events.
func NewProfileUpdatedPayload(userID, displayName, avatarURL string) EventPayload {
	return EventPayload{
		"user_id":      userID,
		"display_name": displayName,
		"avatar_url":   avatarURL,
	}
}

// NewOAuthAccountLinkedPayload creates a payload for auth.oauth.account_linked events.
func NewOAuthAccountLinkedPayload(userID, provider, providerUserID string) EventPayload {
	return EventPayload{
		"user_id":          userID,
		"provider":         provider,
		"provider_user_id": providerUserID,
	}
}

