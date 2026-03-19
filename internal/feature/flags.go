// Package feature provides feature flag management for gradual rollouts
// and A/B testing. It supports multiple backends (in-memory, config file,
// or external services like LaunchDarkly).
package feature

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	flagTrue  = "true"
	flagFalse = "false"
)

// Flag represents a feature flag with its configuration.
type Flag struct {
	// Name is the unique identifier for the flag.
	Name string
	// Description describes what the flag controls.
	Description string
	// DefaultValue is the value when not explicitly set.
	DefaultValue bool
	// Enabled is the current state of the flag.
	Enabled bool
	// Percentage is the rollout percentage (0-100) for gradual rollouts.
	Percentage int
	// Rules define conditions for enabling the flag.
	Rules []Rule
}

// Rule defines a condition for enabling a flag.
type Rule struct {
	// Attribute is the context attribute to check (e.g., "user_id", "email").
	Attribute string
	// Operator is the comparison operator (e.g., "equals", "contains", "in").
	Operator string
	// Value is the value to compare against.
	Value interface{}
}

// Context holds information used to evaluate feature flags.
type Context struct {
	// UserID is the unique identifier of the user.
	UserID string
	// Email is the user's email address.
	Email string
	// Attributes holds additional custom attributes.
	Attributes map[string]interface{}
}

// Manager provides feature flag operations.
type Manager interface {
	// IsEnabled checks if a feature flag is enabled.
	IsEnabled(ctx context.Context, flagName string) bool

	// IsEnabledFor checks if a feature flag is enabled for a specific context.
	IsEnabledFor(ctx context.Context, flagName string, featureCtx Context) bool

	// GetFlag returns the full flag configuration.
	GetFlag(ctx context.Context, flagName string) (*Flag, bool)

	// SetFlag updates or creates a flag.
	SetFlag(ctx context.Context, flag Flag) error

	// ListFlags returns all registered flags.
	ListFlags(ctx context.Context) []Flag
}

// InMemoryManager is an in-memory implementation of feature flag management.
// Suitable for development and small-scale deployments.
type InMemoryManager struct {
	mu    sync.RWMutex
	flags map[string]*Flag
}

// NewInMemoryManager creates a new in-memory feature flag manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		flags: make(map[string]*Flag),
	}
}

// IsEnabled checks if a feature flag is enabled globally.
func (m *InMemoryManager) IsEnabled(_ context.Context, flagName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[flagName]
	if !ok {
		return false
	}

	return flag.Enabled
}

// IsEnabledFor checks if a feature flag is enabled for a specific context.
func (m *InMemoryManager) IsEnabledFor(_ context.Context, flagName string, featureCtx Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[flagName]
	if !ok {
		return false
	}

	// If globally disabled, return false
	if !flag.Enabled {
		return false
	}

	// Check percentage-based rollout
	if flag.Percentage > 0 && flag.Percentage < 100 {
		// Use user ID hash for consistent bucketing
		bucket := hashToBucket(featureCtx.UserID)
		if bucket > flag.Percentage {
			return false
		}
	}

	// Check rules
	for _, rule := range flag.Rules {
		if !evaluateRule(rule, featureCtx) {
			return false
		}
	}

	return true
}

// GetFlag returns the full flag configuration.
func (m *InMemoryManager) GetFlag(_ context.Context, flagName string) (*Flag, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[flagName]
	if !ok {
		return nil, false
	}

	// Return a copy to prevent mutation
	flagCopy := *flag

	return &flagCopy, true
}

// SetFlag updates or creates a flag.
func (m *InMemoryManager) SetFlag(_ context.Context, flag Flag) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.flags[flag.Name] = &flag

	return nil
}

// ListFlags returns all registered flags.
func (m *InMemoryManager) ListFlags(_ context.Context) []Flag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make([]Flag, 0, len(m.flags))
	for _, flag := range m.flags {
		flags = append(flags, *flag)
	}

	return flags
}

// RegisterFlag is a convenience method to register a new flag with a name and default value.
func (m *InMemoryManager) RegisterFlag(name, description string, enabled bool) {
	_ = m.SetFlag(context.Background(), Flag{
		Name:         name,
		Description:  description,
		DefaultValue: enabled,
		Enabled:      enabled,
		Percentage:   100,
	})
}

// SQLManager is a database-backed implementation of feature flag management.
type SQLManager struct {
	db *pgxpool.Pool
	// tableMap maps module/context to table name
	tableMap map[string]string
}

// NewSQLManager creates a new SQLManager.
func NewSQLManager(db *pgxpool.Pool) *SQLManager {
	return &SQLManager{
		db: db,
		tableMap: map[string]string{
			"system":     "admin.system_config",
			"feeds":      "feeds.feed_config",
			"auth":       "auth.auth_config",
			"bets":       "bets.bets_config",
			"events":     "events.events_config",
			"kyc":        "kyc.kyc_config",
			"wallet":     "wallet.wallet_config",
			"settlement": "settlement.settlement_config",
			"admin":      "admin.admin_config",
		},
	}
}

// IsEnabled checks if a feature flag is enabled globally.
// It assumes the flag is in the "system" context.
func (m *SQLManager) IsEnabled(ctx context.Context, flagName string) bool {
	return m.IsEnabledFor(ctx, flagName, Context{Attributes: map[string]interface{}{"context": "system"}})
}

// IsEnabledFor checks if a feature flag is enabled for a specific context.
// The context should contain a "context" attribute mapping to a module name.
func (m *SQLManager) IsEnabledFor(ctx context.Context, flagName string, featureCtx Context) bool {
	module, ok := featureCtx.Attributes["context"].(string)
	if !ok {
		module = "system"
	}

	tableName, ok := m.tableMap[module]
	if !ok {
		return false
	}

	query := "SELECT value FROM " + tableName + " WHERE key = $1"

	var val string

	err := m.db.QueryRow(ctx, query, flagName).Scan(&val)
	if err != nil {
		return false
	}

	return val == flagTrue
}

// GetFlag returns the full flag configuration (simplified for SQLManager).
func (m *SQLManager) GetFlag(ctx context.Context, flagName string) (*Flag, bool) {
	// For now, only basic Enabled check is implemented in SQLManager
	enabled := m.IsEnabled(ctx, flagName)
	return &Flag{Name: flagName, Enabled: enabled}, true
}

// SetFlag updates or creates a flag in the "system" context.
func (m *SQLManager) SetFlag(ctx context.Context, flag Flag) error {
	query := `
		INSERT INTO admin.system_config (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`

	val := flagFalse

	if flag.Enabled {
		val = flagTrue
	}

	if _, err := m.db.Exec(ctx, query, flag.Name, val); err != nil {
		return fmt.Errorf("failed to set feature flag in database: %w", err)
	}

	return nil
}

// ListFlags returns all registered flags from the "system" context.
func (m *SQLManager) ListFlags(ctx context.Context) []Flag {
	query := "SELECT key, value FROM admin.system_config"

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil
	}

	defer rows.Close()

	var flags []Flag

	for rows.Next() {
		var key, val string

		if err := rows.Scan(&key, &val); err == nil {
			flags = append(flags, Flag{Name: key, Enabled: val == flagTrue})
		}
	}

	return flags
}

// hashToBucket converts a string to a bucket number (0-100) for consistent hashing.
func hashToBucket(s string) int {
	if s == "" {
		return 0
	}

	// Simple hash function for bucket assignment
	var hash uint32

	for _, b := range []byte(s) {
		hash = hash*31 + uint32(b)
	}

	return int(hash % 100)
}

// evaluateRule evaluates a single rule against the feature context.
//
//nolint:cyclop // Rule evaluation requires multiple operator checks
func evaluateRule(rule Rule, ctx Context) bool {
	value := getAttributeValue(rule.Attribute, ctx)

	return evaluateOperator(rule.Operator, value, rule.Value)
}

// getAttributeValue extracts the attribute value from the context.
func getAttributeValue(attr string, ctx Context) interface{} {
	switch attr {
	case "user_id":
		return ctx.UserID
	case "email":
		return ctx.Email
	default:
		if ctx.Attributes != nil {
			return ctx.Attributes[attr]
		}

		return nil
	}
}

// evaluateOperator evaluates the operator against the value.
func evaluateOperator(operator string, value, ruleValue interface{}) bool {
	switch operator {
	case "equals":
		return value == ruleValue
	case "not_equals":
		return value != ruleValue
	case "contains":
		return evaluateContains(value, ruleValue)
	case "in":
		return evaluateIn(value, ruleValue)
	default:
		return false
	}
}

// evaluateContains checks if value contains ruleValue.
func evaluateContains(value, ruleValue interface{}) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}

	v, ok := ruleValue.(string)
	if !ok {
		return false
	}

	return contains(s, v)
}

// evaluateIn checks if value is in ruleValue list.
func evaluateIn(value, ruleValue interface{}) bool {
	list, ok := ruleValue.([]string)
	if !ok {
		return false
	}

	s, ok := value.(string)
	if !ok {
		return false
	}

	for _, item := range list {
		if item == s {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
