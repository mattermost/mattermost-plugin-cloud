package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
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

	t.Run("no installation found with the given name", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		response, _, err := plugin.runDeletionLockCommand([]string{"test_installation_name"}, &model.CommandArgs{UserId: "test_user_id"})

		require.EqualError(t, err, "no installation with the name test_installation_name found")
		require.Nil(t, response)
	})

	t.Run("lockForDeletion error", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"JoramsInstall\"}]"), nil)

		response, _, err := plugin.runDeletionLockCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})

		require.EqualError(t, err, "test error")
		require.Equal(t, &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "test error",
		}, response)
	})

}
