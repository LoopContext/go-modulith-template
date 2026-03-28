package testutil_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	authv1 "github.com/LoopContext/go-modulith-template/gen/go/proto/auth/v1"
	"github.com/LoopContext/go-modulith-template/internal/testutil"
	"github.com/LoopContext/go-modulith-template/modules/auth"
)

func TestSetupCrossModuleTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	setup, err := testutil.SetupCrossModuleTest(ctx, t, auth.NewModule())
	require.NoError(t, err)
	require.NotNil(t, setup.Client())
	require.NotNil(t, setup.Registry)
	require.NotNil(t, setup.Pool)

	authClient := testutil.NewServiceClient(setup, authv1.NewAuthServiceClient)

	resp, err := authClient.RequestLogin(ctx, &authv1.RequestLoginRequest{
		ContactInfo: &authv1.RequestLoginRequest_Email{
			Email: "integration@example.com",
		},
	})
	require.NoError(t, err)
	require.True(t, resp.Success)
}
