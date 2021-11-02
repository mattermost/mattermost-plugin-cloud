package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCommand(t *testing.T) {
	dockerClient := &MockedDockerClient{tagExists: true}
	plugin := Plugin{
		cloudClient:  &MockClient{},
		dockerClient: dockerClient,
	}

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

	t.Run("upgrade installation successfully with name with caps to demonstrate case insensitivity of name", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"GabesInstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Upgrade of installation")
	})

	t.Run("no version, license, or size", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify at least one option: version, license, image or size")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("version only", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Upgrade of installation")
	})

	t.Run("size only", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--size", "miniHA"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Upgrade of installation")
	})

	t.Run("licenses", func(t *testing.T) {

		t.Run("invalid", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.13.1", "--license", "e30"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid license option")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("e20", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--license", licenseOptionE20}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Upgrade of installation")
		})

		t.Run("e10", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--license", licenseOptionE10}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Upgrade of installation")
		})

		t.Run("te", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--license", licenseOptionTE}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Upgrade of installation")
		})

	})

	t.Run("version is equal to current version", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\", \"Version\": \"5.31.1\"}]"), nil)

		resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.31.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Upgrade of installation")
	})

	t.Run("docker tag", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Upgrade of installation")
		})

		t.Run("invalid", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			dockerClient.tagExists = false
			defer func() { dockerClient.tagExists = true }()

			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

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

	t.Run("image", func(t *testing.T) {

		t.Run("invalid image", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--image", "mattermost/randomimage"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid image name")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
		t.Run("valid image", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpgradeCommand([]string{"gabesinstall", "--image", "mattermost/mm-ee-test"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Upgrade of installation")
		})
	})

}
