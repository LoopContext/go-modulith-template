// Package events provides typed event constants for the event bus.
package events

// Event type constants for type-safe event publishing and subscription.
// Each module should define its own event constants here.
const (
	// Auth module events
	EventAuthMagicCodeRequested         = "auth.magic_code_requested"
	EventAuthUserCreated                = "auth.user.created"
	EventAuthUserRegistered             = "auth.user.registered"
	EventAuthUserLoggedIn               = "auth.user.logged_in"
	EventAuthUserLoggedOut              = "auth.user.logged_out"
	EventAuthSessionCreated             = "auth.session.created"
	EventAuthSessionRevoked             = "auth.session.revoked"
	EventAuthProfileUpdated             = "auth.profile.updated"
	EventAuthEmailChanged               = "auth.email.changed"
	EventAuthPhoneChanged               = "auth.phone.changed"
	EventAuthEmailVerificationRequested = "auth.email_verification_requested"

	// OAuth events
	EventOAuthAccountLinked   = "auth.oauth.account_linked"
	EventOAuthAccountUnlinked = "auth.oauth.account_unlinked"

	// Generic user events (can be used by any module)
	EventUserCreated = "user.created"
	EventUserUpdated = "user.updated"
	EventUserDeleted = "user.deleted"

	// Audit events
	EventAuditLogCreated = "audit.log.created"

	// KYC events
	EventKYCDocumentUploaded      = "kyc.document_uploaded"
	EventKYCVerificationInitiated = "kyc.verification_initiated"
	EventKYCUserScreened          = "kyc.user_screened"
	EventKYCVerified              = "kyc.verified"
	EventKYCScreeningMatch        = "kyc.screening_match"

	// Positions events
	EventPositionsOpened  = "positions.opened"
	EventPositionWon      = "position.won"
	EventPositionRefunded = "position.refunded"
	EventPositionsInsured = "positions.insured"

	// Derivatives events
	EventDerivativesContractCreated   = "derivatives.contract_created"
	EventDerivativesOptionBought      = "derivatives.option_bought"
	EventDerivativesContractFilled    = "derivatives.contract_filled"
	EventDerivativesContractSettled   = "derivatives.contract_settled"
	EventDerivativesContractCancelled = "derivatives.contract_cancelled"
	EventDerivativesContractExpired   = "derivatives.contract_expired"

	// Events module events
	EventEventCreated        = "events.created"
	EventEventUpdated        = "events.updated"
	EventEventRescheduled    = "events.rescheduled"
	EventEventCancelled      = "events.cancelled"
	EventEventResultProposed = "events.result_proposed"
	EventTemplateCreated     = "events.template_created"
	EventTemplateUpdated     = "events.template_updated"
	EventPoolUpdated         = "events.pool_updated"
	EventLiveUpdate          = "events.live_update"

	// Settlement events
	EventSettlementCompleted = "settlement.completed"
	EventSettlementRefunded  = "settlement.refunded"
	EventSettlementVoided    = "settlement.voided"

	// Dispute events
	EventDisputeCreated  = "dispute.created"
	EventDisputeResolved = "dispute.resolved"

	// Feeds events
	EventFeedsResultReported = "feeds.result_reported"

	// Wallet events
	EventWalletCurrencyExchanged     = "wallet.currency_exchanged"
	EventWalletIntegrityVerified     = "wallet.integrity_verified"
	EventWalletBalanceAdjusted       = "wallet.balance_adjusted"
	EventWalletConfigUpdated         = "wallet.config.updated"
	EventWalletIntentVerifyIntegrity = "wallet.intent.verify_integrity"

	// Messaging events
	EventMessagingProviderCreated = "messaging.provider.created"
	EventMessagingMessageSent     = "messaging.message.sent"
	EventMessagingMessageReceived = "messaging.message.received"
	EventMessagingStatusUpdated   = "messaging.status.updated"
	EventMessagingWebhookReceived = "messaging.webhook.received"
	EventMessagingSendCommand     = "messaging.send_command"

	// Bot Intent events (Decoupled requests)
	EventBotIntentBalanceRequested = "bot.intent.balance_requested"
	EventBotIntentProfileRequested = "bot.intent.profile_requested"
	EventBotIntentEventsRequested  = "bot.intent.events_requested"
	EventBotIntentBetRequested     = "bot.intent.bet_requested"
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

// NewUserRegisteredPayload creates a payload for auth.user.registered events.
func NewUserRegisteredPayload(userID, email, phone, displayName, nationality, docType, docNumber string) EventPayload {
	return EventPayload{
		"user_id":         userID,
		"email":           email,
		"phone":           phone,
		"display_name":    displayName,
		"nationality":     nationality,
		"document_type":   docType,
		"document_number": docNumber,
	}
}
