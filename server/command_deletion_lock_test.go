package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRunDeletionLockCommand(t *testing.T) {
	plugin := Plugin{}

	mockedCloudClient := &MockClient{}
	plugin.cloudClient = mockedCloudClient

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)
	plugin.SetAPI(api)
	t.Run("no installation name provided", func(t *testing.T) {
		response, _, err := plugin.runDeletionLockCommand([]string{}, &model.CommandArgs{UserId: "test_user_id"})

		require.EqualError(t, err, "must provide an installation name")
		require.Nil(t, response)
	})

	t.Run("Invalid config value", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		response, _, err := plugin.runDeletionLockCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})

		require.Contains(t, err.Error(), "invalid value for DeletionLockInstallationsAllowedPerPerson")
		assert.Contains(t, response.Text, "invalid value for DeletionLockInstallationsAllowedPerPerson")
	})

	t.Run("Exceeded deletion lock limit", func(t *testing.T) {
		plugin.configuration = &configuration{
			DeletionLockInstallationsAllowedPerPerson: "0",
		}
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		commandResponse, _, err := plugin.runDeletionLockCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})

		require.Error(t, err)
		assert.Contains(t, commandResponse.Text, "you may only have at most 0 installations locked for deletion at a time")
	})

	t.Run("No error", func(t *testing.T) {
		plugin.configuration = &configuration{
			DeletionLockInstallationsAllowedPerPerson: "1",
		}
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		commandResponse, _, err := plugin.runDeletionLockCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})

		require.NoError(t, err)
		assert.Contains(t, commandResponse.Text, "Deletion lock has been applied, your workspace will be preserved.")
	})

	t.Run("no installation found with the given name", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		response, _, err := plugin.runDeletionLockCommand([]string{"test_installation_name"}, &model.CommandArgs{UserId: "test_user_id"})

		require.EqualError(t, err, "no installation with the name test_installation_name found")
		require.Nil(t, response)
	})

}

func TestLockForDeletion(t *testing.T) {
	plugin := Plugin{}

	mockedCloudClient := &MockClient{}
	plugin.cloudClient = mockedCloudClient

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)
	plugin.SetAPI(api)

	t.Run("No installation ID provided", func(t *testing.T) {
		err := plugin.lockForDeletion("", "test_user_id")

		require.EqualError(t, err, "installationID must not be empty")
	})

	t.Run("Invalid config value", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		err := plugin.lockForDeletion("joramsinstall", "joramid")

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid value for DeletionLockInstallationsAllowedPerPerson")
	})

	t.Run("Exceeded deletion lock limit", func(t *testing.T) {
		plugin.configuration = &configuration{
			DeletionLockInstallationsAllowedPerPerson: "0",
		}
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		err := plugin.lockForDeletion("joramsinstall", "joramid")

		require.Error(t, err)
		require.Contains(t, err.Error(), "you may only have at most 0 installations locked for deletion at a time")
		plugin.configuration = &configuration{
			DeletionLockInstallationsAllowedPerPerson: "1",
		}
	})

	t.Run("No error", func(t *testing.T) {

		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		err := plugin.lockForDeletion("someid", "joramid")

		require.NoError(t, err)
	})

	t.Run("No installations to be locked", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		err := plugin.lockForDeletion("joramsinstall", "joramid")

		require.EqualError(t, err, "installation to be locked not found")
	})

	t.Run("No installations for provided User ID", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		err := plugin.lockForDeletion("test_installation_id", "test_user_id")

		require.EqualError(t, err, "no installations found for the given User ID")
	})

}

func TestRunDeletionUnlockCommand(t *testing.T) {
	plugin := Plugin{}

	mockedCloudClient := &MockClient{}
	plugin.cloudClient = mockedCloudClient

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)
	plugin.SetAPI(api)
	t.Run("no installation name provided", func(t *testing.T) {
		response, _, err := plugin.runDeletionUnlockCommand([]string{}, &model.CommandArgs{UserId: "test_user_id"})

		require.EqualError(t, err, "must provide an installation name")
		require.Nil(t, response)
	})

	t.Run("No error", func(t *testing.T) {
		plugin.configuration = &configuration{
			DeletionLockInstallationsAllowedPerPerson: "1",
		}
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		commandResponse, _, err := plugin.runDeletionUnlockCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})

		require.NoError(t, err)
		assert.Contains(t, commandResponse.Text, "Deletion lock has been removed, your workspace can now be deleted")
	})

	t.Run("no installation found with the given name", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		response, _, err := plugin.runDeletionUnlockCommand([]string{"test_installation_name"}, &model.CommandArgs{UserId: "test_user_id"})

		require.EqualError(t, err, "no installation with the name test_installation_name found")
		require.Nil(t, response)
	})
}

func TestUnlockForDeletion(t *testing.T) {
	plugin := Plugin{}

	mockedCloudClient := &MockClient{}
	plugin.cloudClient = mockedCloudClient

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)
	plugin.SetAPI(api)

	t.Run("No installation ID provided", func(t *testing.T) {
		err := plugin.unlockForDeletion("", "test_user_id")

		require.EqualError(t, err, "installationID must not be empty")
	})

	t.Run("No error", func(t *testing.T) {

		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		err := plugin.unlockForDeletion("someid", "joramid")

		require.NoError(t, err)
	})

	t.Run("No installations to be unlocked", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		err := plugin.unlockForDeletion("joramsinstall", "joramid")

		require.EqualError(t, err, "installation to be unlocked not found")
	})

	t.Run("No installations for provided User ID", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		err := plugin.unlockForDeletion("test_installation_id", "test_user_id")

		require.EqualError(t, err, "no installations found for the given User ID")
	})

}
