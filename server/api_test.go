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

// Regression: JSON null must not nil the decoded pointer and panic on field access.
func TestJSONNullBodyDoesNotPanic(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		handler func(*Plugin, http.ResponseWriter, *http.Request)
	}{
		{
			name:    "userinstalls",
			path:    "/api/v1/userinstalls",
			handler: (*Plugin).handleUserInstalls,
		},
		{
			name:    "deletion-lock",
			path:    "/api/v1/deletion-lock",
			handler: (*Plugin).handleDeletionLock,
		},
		{
			name:    "deletion-unlock",
			path:    "/api/v1/deletion-unlock",
			handler: (*Plugin).handleDeletionUnlock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, _, _ := newServiceTestPlugin(t, nil)

			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("null"))
			req.Header.Set("Mattermost-User-ID", "user1")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() {
				tt.handler(plugin, rec, req)
			})
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "Please provide a JSON object")
		})
	}
}
