package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeletionLockHandlers(t *testing.T) {
	t.Run("lock returns created with request body", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{target})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(target)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/deletion-lock", strings.NewReader(`{"installation_id":"target-id"}`))
		req.Header.Set("Mattermost-User-ID", "owner")
		rec := httptest.NewRecorder()

		plugin.handleDeletionLock(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.JSONEq(t, `{"installation_id":"target-id"}`, rec.Body.String())
		assert.Equal(t, "target-id", cloudClient.lockedInstallationID)
	})

	t.Run("unlock returns created with request body", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		target.DeletionLocked = true
		plugin, cloudClient, _ := newServiceTestPlugin(t, []*Installation{target})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(target)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/deletion-unlock", strings.NewReader(`{"installation_id":"target-id"}`))
		req.Header.Set("Mattermost-User-ID", "owner")
		rec := httptest.NewRecorder()

		plugin.handleDeletionUnlock(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.JSONEq(t, `{"installation_id":"target-id"}`, rec.Body.String())
		assert.Equal(t, "target-id", cloudClient.unlockedInstallationID)
	})

	t.Run("wrong owner keeps existing internal error behavior", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		plugin, _, api := newServiceTestPlugin(t, []*Installation{target})
		api.On("LogError", mock.AnythingOfType("string")).Return(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/deletion-lock", strings.NewReader(`{"installation_id":"target-id"}`))
		req.Header.Set("Mattermost-User-ID", "other")
		rec := httptest.NewRecorder()

		plugin.handleDeletionLock(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Internal server error")
	})
}
