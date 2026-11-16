package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleUserInstallsProvisionerUnavailable(t *testing.T) {
	plugin, cloudClient, _ := newServiceTestPlugin(t, nil)
	cloudClient.listErr = errors.New("dial tcp: no such host")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/userinstalls", strings.NewReader(`{"user_id":"owner"}`))
	req.Header.Set("Mattermost-User-ID", "owner")
	rec := httptest.NewRecorder()

	plugin.handleUserInstalls(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "Provisioning server unavailable")
}

func TestHandleUserInstallsKVErrorStaysInternal(t *testing.T) {
	plugin, _, api := newServiceTestPlugin(t, nil)
	api.ExpectedCalls = nil
	api.On("KVGet", StoreInstallsKey).Return([]byte("not json"), nil)
	api.On("LogError", mock.AnythingOfType("string")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/userinstalls", strings.NewReader(`{"user_id":"owner"}`))
	req.Header.Set("Mattermost-User-ID", "owner")
	rec := httptest.NewRecorder()

	plugin.handleUserInstalls(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
