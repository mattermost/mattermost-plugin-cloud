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

func TestShareCommand(t *testing.T) {
	dockerClient := &MockedDockerClient{tagExists: true}
	mockCloudClient := &MockClient{
		mockedCloudInstallationsDTO: []*cloud.InstallationDTO{
			{Installation: &cloud.Installation{ID: "someid", OwnerID: "gabeid"}},
		},
	}
	plugin := Plugin{
		cloudClient:  mockCloudClient,
		dockerClient: dockerClient,
	}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
	plugin.SetAPI(api)

	t.Run("share installation successfully", func(t *testing.T) {
		resp, isUserError, err := plugin.runShareInstallationCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation has been shared with other plugin users. Other plugin users are not permitted to update this installation.")
	})

	t.Run("share installation successfully with name with caps to demonstrate case insensitivity of name", func(t *testing.T) {
		resp, isUserError, err := plugin.runShareInstallationCommand([]string{"GabesInstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation has been shared with other plugin users. Other plugin users are not permitted to update this installation.")
	})

	t.Run("share installation successfully and allow updates", func(t *testing.T) {
		resp, isUserError, err := plugin.runShareInstallationCommand([]string{"gabesinstall", "--allow-updates"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation has been shared with other plugin users. Other plugin users will be allowed to update this installation.")
	})

	t.Run("missing installation name", func(t *testing.T) {
		resp, isUserError, err := plugin.runShareInstallationCommand([]string{}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("invalid installation name", func(t *testing.T) {
		resp, isUserError, err := plugin.runShareInstallationCommand([]string{"gabesinstall2"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})
}

func TestUnshareCommand(t *testing.T) {
	dockerClient := &MockedDockerClient{tagExists: true}
	mockCloudClient := &MockClient{
		mockedCloudInstallationsDTO: []*cloud.InstallationDTO{
			{Installation: &cloud.Installation{ID: "someid", OwnerID: "gabeid"}},
		},
	}
	plugin := Plugin{
		cloudClient:  mockCloudClient,
		dockerClient: dockerClient,
	}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
	plugin.SetAPI(api)

	t.Run("unshare installation successfully", func(t *testing.T) {
		resp, isUserError, err := plugin.runUnshareInstallationCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation has been unshared.")
	})

	t.Run("share installation successfully with name with caps to demonstrate case insensitivity of name", func(t *testing.T) {
		resp, isUserError, err := plugin.runUnshareInstallationCommand([]string{"GabesInstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation has been unshared.")
	})

	t.Run("missing installation name", func(t *testing.T) {
		resp, isUserError, err := plugin.runUnshareInstallationCommand([]string{}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("invalid installation name", func(t *testing.T) {
		resp, isUserError, err := plugin.runUnshareInstallationCommand([]string{"gabesinstall2"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})
}
