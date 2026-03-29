package telemetry

import (
	"context"
	"fmt"
)

var (
	// GMVTotal tracks the Total Gross Merchandise Value.
	GMVTotal *Counter
	// SettlementPayout tracks the total settlement payouts.
	SettlementPayout *Counter
	// ActiveUsersGauge tracks the current number of active users.
	ActiveUsersGauge *Gauge
	// BetPlacementLatency tracks the latency of bet placements.
	BetPlacementLatency *Histogram
	// AuthLoginTotal tracks the total number of logins.
	AuthLoginTotal *Counter
	// AuthSignupTotal tracks the total number of new signups.
	AuthSignupTotal *Counter
	// PositionsPlacedTotal tracks the total number of positions placed.
	PositionsPlacedTotal *Counter
	// WalletTransactionsTotal tracks the total number of wallet transactions.
	WalletTransactionsTotal *Counter
	// WalletTransactionLatency tracks the latency of wallet transactions.
	WalletTransactionLatency *Histogram
)

// InitBusinessMetrics initializes the business-specific metrics.
func InitBusinessMetrics() error {
	var err error

	GMVTotal, err = NewCounter("modulith_gmv_total", "Total Gross Merchandise Volume (staked amount)")
	if err != nil {
		return fmt.Errorf("failed to create gmv_total metric: %w", err)
	}

	SettlementPayout, err = NewCounter("modulith_settlement_payout_total", "Total amount paid out to winners")
	if err != nil {
		return fmt.Errorf("failed to create settlement_payout_total metric: %w", err)
	}

	ActiveUsersGauge, err = NewGauge("modulith_active_users", "Number of active users in the last 5 minutes")
	if err != nil {
		return fmt.Errorf("failed to create active_users metric: %w", err)
	}

	BetPlacementLatency, err = NewHistogram("modulith_bet_placement_duration_seconds", "Duration of bet placement operations", "s")
	if err != nil {
		return fmt.Errorf("failed to create bet_placement_duration metric: %w", err)
	}

	AuthLoginTotal, err = NewCounter("modulith_auth_login_total", "Total number of successful logins")
	if err != nil {
		return fmt.Errorf("failed to create auth_login_total metric: %w", err)
	}

	AuthSignupTotal, err = NewCounter("modulith_auth_signup_total", "Total number of new signups")
	if err != nil {
		return fmt.Errorf("failed to create auth_signup_total metric: %w", err)
	}

	PositionsPlacedTotal, err = NewCounter("modulith_positions_placed_total", "Total number of positions placed")
	if err != nil {
		return fmt.Errorf("failed to create positions_placed_total metric: %w", err)
	}

	WalletTransactionsTotal, err = NewCounter("modulith_wallet_transactions_total", "Total number of wallet transactions")
	if err != nil {
		return fmt.Errorf("failed to create wallet_transactions_total metric: %w", err)
	}

	WalletTransactionLatency, err = NewHistogram("modulith_wallet_transaction_duration_seconds", "Duration of wallet transaction operations", "s")
	if err != nil {
		return fmt.Errorf("failed to create wallet_transaction_duration metric: %w", err)
	}

	return nil
}

// TrackGMV records a staked amount in the GMV metric.
func TrackGMV(ctx context.Context, amount int64, currency string) {
	if GMVTotal != nil {
		GMVTotal.WithAttributes(Attr("currency", currency)).Add(ctx, amount)
	}
}

// TrackPayout records a payout amount.
func TrackPayout(ctx context.Context, amount int64, currency string) {
	if SettlementPayout != nil {
		SettlementPayout.WithAttributes(Attr("currency", currency)).Add(ctx, amount)
	}
}
