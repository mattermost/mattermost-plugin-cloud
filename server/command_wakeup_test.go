package main

import (
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWakeUpCommand(t *testing.T) {
	plugin := Plugin{}
	mockedCloudClient := &MockClient{}
	mockedCloudClient.overrideGetInstallationDTO = &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid", State: cloud.InstallationStateHibernating}}
	plugin.cloudClient = mockedCloudClient

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)
	plugin.SetAPI(api)

	t.Run("wake up installation successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)

		resp, isUserError, err := plugin.runWakeUpCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation joramsinstall is waking up.")
	})

	t.Run("hibernate installation successfully with caps in name to demonstrate name case insensitivity", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"JoramsInstall\"}]"), nil)

		resp, isUserError, err := plugin.runWakeUpCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation joramsinstall is waking up.")
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runWakeUpCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid2"})
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "no installation with the name joramsinstall found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runWakeUpCommand([]string{}, &model.CommandArgs{UserId: "joramid2"})
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("installation is not hibernating", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"joramsinstall\"}]"), nil)
		mockedCloudClient.overrideGetInstallationDTO = &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid", State: cloud.InstallationStateStable}}

		resp, isUserError, err := plugin.runWakeUpCommand([]string{"joramsinstall"}, &model.CommandArgs{UserId: "joramid"})
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "installation state is currently")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})
}
