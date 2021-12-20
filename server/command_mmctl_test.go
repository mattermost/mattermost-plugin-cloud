package main

import (
	"strings"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMmctlCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	plugin.SetAPI(api)

	ci1 := &cloud.ClusterInstallation{
		ID: cloud.NewID(),
	}
	mockedCloudClient.mockedCloudClusterInstallations = []*cloud.ClusterInstallation{ci1}

	t.Run("run command successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runMmctlCommand([]string{"gabesinstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "mocked command output")
	})

	t.Run("run command successfully with caps in name to show name is case insensitive", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runMmctlCommand([]string{"GabesInstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "mocked command output")
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runMmctlCommand([]string{}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runMmctlCommand([]string{""}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no mmctl subcommand", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runMmctlCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide a mmctl command")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runMmctlCommand([]string{"gabesinstall2", "version"}, &model.CommandArgs{UserId: "gabeid2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no cluster installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
		mockedCloudClient.mockedCloudClusterInstallations = []*cloud.ClusterInstallation{}

		resp, isUserError, err := plugin.runMmctlCommand([]string{"gabesinstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no cluster installations found for installation")
		assert.False(t, isUserError)
		assert.Nil(t, resp)
	})
}
