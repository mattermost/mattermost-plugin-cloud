package main

import (
	"strings"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockClient struct{}

func (mc *MockClient) CreateInstallation(request *cloud.CreateInstallationRequest) (*cloud.Installation, error) {
	return &cloud.Installation{ID: "someid"}, nil
}

func (mc *MockClient) GetInstallation(installataionID string) (*cloud.Installation, error) {
	return &cloud.Installation{ID: "someid", OwnerID: "joramid"}, nil
}

func TestCreateCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return([]byte{}, nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)

	plugin.SetAPI(api)

	t.Run("create installation successfully", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest"}, &model.CommandArgs{})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "Installation being created."))
	})

	t.Run("missing installation name", func(t *testing.T) {
		_, isUserError, err := plugin.runCreateCommand([]string{""}, &model.CommandArgs{})
		require.NotNil(t, err)
		assert.Equal(t, "must provide an installation name", err.Error())
		assert.True(t, isUserError)

		_, isUserError, err = plugin.runCreateCommand([]string{"--blargh"}, &model.CommandArgs{})
		require.NotNil(t, err)
		assert.Equal(t, "must provide an installation name", err.Error())
		assert.True(t, isUserError)
	})
}

func TestListCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	plugin.SetAPI(api)

	t.Run("list installations successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\"}]"), nil)

		resp, isUserError, err := plugin.runListCommand([]string{}, &model.CommandArgs{UserId: "joramid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "someid"))
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte{}, nil)

		resp, isUserError, err := plugin.runListCommand([]string{}, &model.CommandArgs{})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.False(t, strings.Contains(resp.Text, "someid"))
	})

	t.Run("no installations for current user", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\"}]"), nil)

		resp, isUserError, err := plugin.runListCommand([]string{}, &model.CommandArgs{UserId: "joramid2"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.False(t, strings.Contains(resp.Text, "someid"))
	})
}

func TestDeleteCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	plugin.SetAPI(api)

	t.Run("delete installation successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		resp, isUserError, err := plugin.runDeleteCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "Installation joramsinstall deleted."))
	})

	t.Run("don't delete with wrong owner", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		resp, isUserError, err := plugin.runDeleteCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "no installation with the name joramsinstall found"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte{}, nil)

		resp, isUserError, err := plugin.runDeleteCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "no installation with the name joramsinstall found"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runDeleteCommand([]string{}, &model.CommandArgs{UserId: "joramid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runDeleteCommand([]string{""}, &model.CommandArgs{UserId: "joramid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})
}
