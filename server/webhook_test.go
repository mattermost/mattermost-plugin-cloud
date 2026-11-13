package main

import (
	"net/http"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
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

func TestProcessWebhookEventPublishesInstallationState(t *testing.T) {
	install := serviceTestInstall("install-id", "Install", "owner")
	plugin, _, api := newServiceTestPlugin(t, []*Installation{install})
	api.On("LogDebug", mock.Anything).Return()
	api.On("PublishWebSocketEvent", websocketEventInstallationUpdated, mock.Anything, mock.Anything).Return()

	plugin.processWebhookEvent(&cloud.WebhookPayload{
		Type:     cloud.TypeInstallation,
		ID:       "install-id",
		OldState: cloud.InstallationStateUpdateRequested,
		NewState: cloud.InstallationStateUpdateInProgress,
	})

	api.AssertCalled(t, "PublishWebSocketEvent", websocketEventInstallationUpdated,
		map[string]any{"installation_id": "install-id", "owner_id": "owner", "state": cloud.InstallationStateUpdateInProgress, "shared": false},
		&model.WebsocketBroadcast{UserId: "owner"})
}
