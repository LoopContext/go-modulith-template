package feature_test

import (
	"context"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/feature"
)

func TestInMemoryManager_BasicFlag(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	// Register a flag
	manager.RegisterFlag("new_dashboard", "Enable new dashboard UI", true)

	// Should be enabled
	if !manager.IsEnabled(ctx, "new_dashboard") {
		t.Error("expected new_dashboard to be enabled")
	}

	// Non-existent flag should be disabled
	if manager.IsEnabled(ctx, "non_existent") {
		t.Error("expected non_existent to be disabled")
	}
}

func TestInMemoryManager_DisabledFlag(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	// Register a disabled flag
	manager.RegisterFlag("legacy_feature", "Legacy feature to be removed", false)

	if manager.IsEnabled(ctx, "legacy_feature") {
		t.Error("expected legacy_feature to be disabled")
	}
}

func TestInMemoryManager_GetFlag(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	// Register a flag
	err := manager.SetFlag(ctx, feature.Flag{
		Name:        "test_flag",
		Description: "Test flag description",
		Enabled:     true,
		Percentage:  50,
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	// Get the flag
	flag, ok := manager.GetFlag(ctx, "test_flag")
	if !ok {
		t.Fatal("expected to find test_flag")
	}

	if flag.Name != "test_flag" {
		t.Errorf("expected name 'test_flag', got '%s'", flag.Name)
	}

	if flag.Percentage != 50 {
		t.Errorf("expected percentage 50, got %d", flag.Percentage)
	}
}

func TestInMemoryManager_ListFlags(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	// Register multiple flags
	manager.RegisterFlag("flag_a", "First flag", true)
	manager.RegisterFlag("flag_b", "Second flag", false)
	manager.RegisterFlag("flag_c", "Third flag", true)

	flags := manager.ListFlags(ctx)
	if len(flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(flags))
	}
}

func TestInMemoryManager_PercentageRollout(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	// Register a flag with 50% rollout
	err := manager.SetFlag(ctx, feature.Flag{
		Name:       "partial_rollout",
		Enabled:    true,
		Percentage: 50,
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	// Test with multiple users - should get roughly 50% enabled
	enabledCount := 0
	totalUsers := 100

	for i := 0; i < totalUsers; i++ {
		featureCtx := feature.Context{
			UserID: string(rune('A' + i%26)) + string(rune('0'+i)),
		}

		if manager.IsEnabledFor(ctx, "partial_rollout", featureCtx) {
			enabledCount++
		}
	}

	// Allow for some variance (30-70% range)
	if enabledCount < 30 || enabledCount > 70 {
		t.Errorf("expected roughly 50%% enabled, got %d/%d", enabledCount, totalUsers)
	}
}

func TestInMemoryManager_RuleEvaluation(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	// Register a flag with rules
	err := manager.SetFlag(ctx, feature.Flag{
		Name:       "beta_feature",
		Enabled:    true,
		Percentage: 100,
		Rules: []feature.Rule{
			{
				Attribute: "email",
				Operator:  "contains",
				Value:     "@beta.com",
			},
		},
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	// Beta user should have access
	betaCtx := feature.Context{
		UserID: "user_123",
		Email:  "user@beta.com",
	}

	if !manager.IsEnabledFor(ctx, "beta_feature", betaCtx) {
		t.Error("expected beta_feature to be enabled for beta user")
	}

	// Regular user should not have access
	regularCtx := feature.Context{
		UserID: "user_456",
		Email:  "user@example.com",
	}

	if manager.IsEnabledFor(ctx, "beta_feature", regularCtx) {
		t.Error("expected beta_feature to be disabled for regular user")
	}
}

func TestInMemoryManager_ConsistentHashing(t *testing.T) {
	manager := feature.NewInMemoryManager()
	ctx := context.Background()

	err := manager.SetFlag(ctx, feature.Flag{
		Name:       "consistent_flag",
		Enabled:    true,
		Percentage: 50,
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	featureCtx := feature.Context{
		UserID: "consistent_user",
	}

	// Same user should always get the same result
	firstResult := manager.IsEnabledFor(ctx, "consistent_flag", featureCtx)

	for i := 0; i < 10; i++ {
		result := manager.IsEnabledFor(ctx, "consistent_flag", featureCtx)
		if result != firstResult {
			t.Error("expected consistent result for same user")
		}
	}
}

