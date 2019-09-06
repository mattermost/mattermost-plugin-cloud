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

func (mc *MockClient) UpgradeInstallation(installataionID, version, license string) error {
	return nil
}

func (mc *MockClient) DeleteInstallation(installationID string) error {
	return nil
}

func TestCreateCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)

	plugin.SetAPI(api)

	t.Run("create installation successfully", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation being created.")
	})

	t.Run("invalid license", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--license", "e30"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid license option")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("missing installation name", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{""}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Equal(t, "must provide an installation name", err.Error())
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runCreateCommand([]string{"--blargh"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Equal(t, "must provide an installation name", err.Error())
		assert.True(t, isUserError)
		assert.Nil(t, resp)
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
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

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

func TestUpgradeCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	plugin.SetAPI(api)

	t.Run("upgrade installation successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Upgrade of installation")
	})

	t.Run("no version", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify a version")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("invalid license", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.13.1", "--license", "e30"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid license option")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall2", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runUpgradeCommand([]string{}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runUpgradeCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
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
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

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
