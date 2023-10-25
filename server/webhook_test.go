package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthenticateWebhook(t *testing.T) {
	plugin := Plugin{
		configuration: &configuration{},
	}

	t.Run("no auth set, header not defined", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)

		require.NoError(t, plugin.authenticateWebhook(request))
	})

	t.Run("no auth set, header defined as empty", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)
		request.Header.Add(authHeaderKey, "")

		require.NoError(t, plugin.authenticateWebhook(request))
	})

	t.Run("no auth set, header defined as wrong value", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)
		request.Header.Add(authHeaderKey, "test")

		require.EqualError(t, plugin.authenticateWebhook(request), "unauthorized")
	})

	plugin.configuration.ProvisioningServerWebhookSecret = "secret1"

	t.Run("auth set, header not defined", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)

		require.EqualError(t, plugin.authenticateWebhook(request), "unauthorized")
	})

	t.Run("auth set, header defined as empty", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)
		request.Header.Add(authHeaderKey, "")

		require.EqualError(t, plugin.authenticateWebhook(request), "unauthorized")
	})

	t.Run("auth set, header defined as wrong value", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)
		request.Header.Add(authHeaderKey, "test")

		require.EqualError(t, plugin.authenticateWebhook(request), "unauthorized")
	})

	t.Run("auth set, header defined as right value", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, "test.domain.com", nil)
		require.NoError(t, err)
		request.Header.Add(authHeaderKey, "secret1")

		require.NoError(t, plugin.authenticateWebhook(request))
	})
}
