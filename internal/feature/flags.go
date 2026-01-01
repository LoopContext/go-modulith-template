// Package feature provides feature flag management for gradual rollouts
// and A/B testing. It supports multiple backends (in-memory, config file,
// or external services like LaunchDarkly).
package feature

import (
	"context"
	"sync"
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

// hashToBucket converts a string to a bucket number (0-100) for consistent hashing.
func hashToBucket(s string) int {
	if s == "" {
		return 0
	}

	// Simple hash function for bucket assignment
	var hash uint32

	for _, c := range s {
		hash = hash*31 + uint32(c)
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

