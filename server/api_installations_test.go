package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func postInstallationAction(plugin *Plugin, userID, action, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/installations/"+action, strings.NewReader(body))
	if userID != "" {
		req.Header.Set("Mattermost-User-ID", userID)
	}
	rec := httptest.NewRecorder()
	plugin.handleInstallationAction(rec, req, action)
	return rec
}

func expectInstallationEvents(api interface {
	On(methodName string, arguments ...any) *mock.Call
}) {
	api.On("PublishWebSocketEvent", websocketEventInstallationUpdated, mock.Anything, mock.Anything).Return()
}

func TestInstallationActionHandlersRejectBadRequests(t *testing.T) {
	plugin, _, api := newServiceTestPlugin(t, nil)
	api.On("LogError", mock.AnythingOfType("string")).Return(nil)

	t.Run("missing user", func(t *testing.T) {
		rec := postInstallationAction(plugin, "", "restart", `{"installation_id":"id"}`)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/installations/restart", nil)
		req.Header.Set("Mattermost-User-ID", "owner")
		rec := httptest.NewRecorder()
		plugin.handleInstallationAction(rec, req, "restart")
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("unknown action", func(t *testing.T) {
		rec := postInstallationAction(plugin, "owner", "explode", `{}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		rec := postInstallationAction(plugin, "owner", "update", `{`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	for _, action := range []string{"update", "restart", "hibernate", "wake", "delete", "share"} {
		t.Run("null body for "+action, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			require.NotPanics(t, func() {
				rec = postInstallationAction(plugin, "owner", action, "null")
			})
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "installation_id")
		})
	}
}

func TestInstallationCreateHandler(t *testing.T) {
	t.Run("creates installation and publishes event", func(t *testing.T) {
		plugin, cloudClient, api := newServiceTestPlugin(t, nil)
		plugin.dockerClient = &MockedDockerClient{tagExists: true, digest: "sha256:digest"}
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "create", `{"name":"Example","version":"9.5.0","env":{"ENV1":"value1"}}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.NotNil(t, cloudClient.creationRequest)
		assert.Equal(t, "example", cloudClient.creationRequest.Name)
		assert.Contains(t, rec.Body.String(), `"status":"create_requested"`)
		assert.NotContains(t, rec.Body.String(), "value1")
	})

	t.Run("invalid name is a bad request", func(t *testing.T) {
		plugin, _, _ := newServiceTestPlugin(t, nil)

		rec := postInstallationAction(plugin, "owner", "create", `{"name":"not valid!"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "only letters, numbers, and hyphens")
	})

	t.Run("missing name is a bad request", func(t *testing.T) {
		plugin, _, _ := newServiceTestPlugin(t, nil)

		rec := postInstallationAction(plugin, "owner", "create", `{}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "must provide an installation name")
	})
}

func TestInstallationUpdateHandler(t *testing.T) {
	t.Run("owner updates installation", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{install})
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "update", `{"installation_id":"install-id","size":"miniHA"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, cloudClient.patchRequest)
		assert.Equal(t, "miniHA", *cloudClient.patchRequest.Size)
		assert.Equal(t, "install-id", cloudClient.patchInstallationID)
	})

	t.Run("empty update is a bad request", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, _, _ := newServiceTestPlugin(t, []*Installation{install})

		rec := postInstallationAction(plugin, "owner", "update", `{"installation_id":"install-id"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "must specify at least one option")
	})

	t.Run("unknown installation is a bad request", func(t *testing.T) {
		plugin, _, _ := newServiceTestPlugin(t, nil)

		rec := postInstallationAction(plugin, "owner", "update", `{"installation_id":"missing","size":"miniHA"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "no installation with the id missing found")
	})

	t.Run("shared installation without update permission is hidden", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		install.Shared = true
		plugin, _, _ := newServiceTestPlugin(t, []*Installation{install})

		rec := postInstallationAction(plugin, "other", "update", `{"installation_id":"install-id","size":"miniHA"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("shared update by another user notifies the owner", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		install.Shared = true
		install.AllowSharedUpdates = true
		plugin, _, api := newServiceTestPlugin(t, []*Installation{install})
		expectInstallationEvents(api)
		api.On("GetUser", "other").Return(&model.User{Id: "other", Username: "other-user"}, nil)
		api.On("GetDirectChannel", mock.Anything, mock.Anything).Return(&model.Channel{Id: "dm"}, nil).Maybe()
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Maybe()
		api.On("GetUser", "owner").Return(&model.User{Id: "owner"}, nil).Maybe()

		rec := postInstallationAction(plugin, "other", "update", `{"installation_id":"install-id","size":"miniHA"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		api.AssertCalled(t, "GetUser", "other")
	})
}

func TestInstallationLifecycleHandlers(t *testing.T) {
	t.Run("hibernate stable installation", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{install})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(install)
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "hibernate", `{"installation_id":"install-id"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "install-id", cloudClient.hibernatedInstallationID)
	})

	t.Run("wake requires hibernating state", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{install})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(install)

		rec := postInstallationAction(plugin, "owner", "wake", `{"installation_id":"install-id"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "to wake up")
	})

	t.Run("restart patches the installation", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{install})
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "restart", `{"installation_id":"install-id"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "install-id", cloudClient.patchInstallationID)
	})

	t.Run("delete requires matching name", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{install})

		rec := postInstallationAction(plugin, "owner", "delete", `{"installation_id":"install-id","confirm_name":"other"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Empty(t, cloudClient.deletedInstallationID)
	})

	t.Run("delete with matching name", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{install})
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "delete", `{"installation_id":"install-id","confirm_name":"Install"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "install-id", cloudClient.deletedInstallationID)
		api.AssertCalled(t, "PublishWebSocketEvent", websocketEventInstallationUpdated, mock.MatchedBy(func(payload map[string]any) bool {
			return payload["state"] == "deleted"
		}), mock.Anything)
	})

	t.Run("share updates sharing flags", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{install})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(install)
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "share", `{"installation_id":"install-id","shared":true,"allow_updates":true}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"shared":true`)
		assert.Contains(t, rec.Body.String(), `"allow_shared_updates":true`)
	})

	t.Run("unsharing notifies other users", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		install.Shared = true
		plugin, cloudClient, api := newServiceTestPlugin(t, []*Installation{install})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(install)
		expectInstallationEvents(api)

		rec := postInstallationAction(plugin, "owner", "share", `{"installation_id":"install-id","shared":false}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		api.AssertNumberOfCalls(t, "PublishWebSocketEvent", 2)
		api.AssertCalled(t, "PublishWebSocketEvent", websocketEventInstallationUpdated, mock.Anything,
			&model.WebsocketBroadcast{OmitUsers: map[string]bool{"owner": true}})
	})
}

func TestPublishInstallationEvent(t *testing.T) {
	t.Run("private installation goes to the owner only", func(t *testing.T) {
		plugin, _, api := newServiceTestPlugin(t, nil)
		expectInstallationEvents(api)

		plugin.publishInstallationEvent(InstallationEvent{InstallationID: "id", OwnerID: "owner", State: "stable"})

		api.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
		api.AssertCalled(t, "PublishWebSocketEvent", websocketEventInstallationUpdated,
			map[string]any{"installation_id": "id", "owner_id": "owner", "state": "stable", "shared": false},
			&model.WebsocketBroadcast{UserId: "owner"})
	})

	t.Run("shared installation also goes to other users", func(t *testing.T) {
		plugin, _, api := newServiceTestPlugin(t, nil)
		expectInstallationEvents(api)

		plugin.publishInstallationEvent(InstallationEvent{InstallationID: "id", OwnerID: "owner", State: "stable", Shared: true})

		api.AssertNumberOfCalls(t, "PublishWebSocketEvent", 2)
		api.AssertCalled(t, "PublishWebSocketEvent", websocketEventInstallationUpdated, mock.Anything,
			&model.WebsocketBroadcast{OmitUsers: map[string]bool{"owner": true}})
	})

	t.Run("event without installation id is dropped", func(t *testing.T) {
		plugin, _, api := newServiceTestPlugin(t, nil)

		plugin.publishInstallationEvent(InstallationEvent{OwnerID: "owner"})

		api.AssertNotCalled(t, "PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("webhook event is published for tracked installation only", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, _, api := newServiceTestPlugin(t, []*Installation{install})
		expectInstallationEvents(api)

		plugin.publishInstallationWebhookEvent("unknown-id", "stable")
		api.AssertNotCalled(t, "PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything)

		plugin.publishInstallationWebhookEvent("install-id", "update-in-progress")
		api.AssertCalled(t, "PublishWebSocketEvent", websocketEventInstallationUpdated,
			map[string]any{"installation_id": "install-id", "owner_id": "owner", "state": "update-in-progress", "shared": false},
			&model.WebsocketBroadcast{UserId: "owner"})
	})
}

func TestConfigResponseOptions(t *testing.T) {
	response := (&configuration{DeletionLockInstallationsAllowedPerPerson: "2"}).ToConfigResponse()

	assert.Equal(t, "2", response.DeletionLockInstallationsAllowedPerPerson)
	assert.Equal(t, validInstallationSizes, response.Options.Sizes)
	assert.Equal(t, validLicenseOptions, response.Options.Licenses)
	assert.Equal(t, dockerRepoWhitelist, response.Options.Images)
	assert.Contains(t, response.Options.Databases, response.Options.DefaultDatabase)
	assert.Contains(t, response.Options.Filestores, response.Options.DefaultFilestore)
	assert.Contains(t, response.Options.Affinities, response.Options.DefaultAffinity)
	assert.Equal(t, "miniSingleton", response.Options.DefaultSize)
	assert.Equal(t, defaultImage, response.Options.DefaultImage)
}

func TestDeletionLockLimitIsBadRequest(t *testing.T) {
	first := serviceTestInstall("first-id", "First", "owner")
	first.DeletionLocked = true
	second := serviceTestInstall("second-id", "Second", "owner")
	second.DeletionLocked = true
	third := serviceTestInstall("third-id", "Third", "owner")
	plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{first, second, third})
	cloudClient.mockedCloudInstallationsDTO = serviceDTOs(first, second, third)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deletion-lock", strings.NewReader(`{"installation_id":"third-id"}`))
	req.Header.Set("Mattermost-User-ID", "owner")
	rec := httptest.NewRecorder()

	plugin.handleDeletionLock(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "you may only have at most 2 installations locked")
}
