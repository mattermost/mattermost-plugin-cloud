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

func TestUpdateCommand(t *testing.T) {
	dockerClient := &MockedDockerClient{tagExists: true}
	mockCloudClient := &MockClient{}
	plugin := Plugin{
		cloudClient:  mockCloudClient,
		dockerClient: dockerClient,
	}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	plugin.SetAPI(api)

	t.Run("update installation successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Update of installation")
	})

	t.Run("update installation successfully with name with caps to demonstrate case insensitivity of name", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"GabesInstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Update of installation")
	})

	t.Run("no version, license, or size", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify at least one option: version, license, image, size, env, clear-env")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("size", func(t *testing.T) {
		t.Run("incorrect size", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte(`[{"ID": "someid", "OwnerID": "gabeid", "Name": "gabesinstall", "Size": "1000users"}]`), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--size", "1000users"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Invalid size:")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("valid size", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte(`[{"ID": "someid", "OwnerID": "gabeid", "Name": "gabesinstall", "Size": "miniSingleton"}]`), nil)

			_, _, err := plugin.runUpdateCommand([]string{"gabesinstall", "--size", "miniSingleton"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
		})
	})

	t.Run("version only", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Update of installation")
	})

	t.Run("size only", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--size", "miniHA"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Update of installation")
	})

	t.Run("licenses", func(t *testing.T) {

		t.Run("invalid", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.13.1", "--license", "e30"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid license option")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("enterprise", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--license", licenseOptionE20}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

		t.Run("professional", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--license", licenseOptionProfessional}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

		t.Run("e20", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--license", licenseOptionE20}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

		t.Run("e10", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--license", licenseOptionE10}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

		t.Run("te", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--license", licenseOptionTE}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

	})

	t.Run("version is equal to current version", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\", \"Version\": \"5.31.1\"}]"), nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.31.1"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Update of installation")
	})

	t.Run("docker tag", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

		t.Run("invalid", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			dockerClient.tagExists = false
			defer func() { dockerClient.tagExists = true }()

			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall2", "--version", "5.13.1"}, &model.CommandArgs{UserId: "gabeid2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runUpdateCommand([]string{}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runUpdateCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation name")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("image", func(t *testing.T) {

		t.Run("invalid image", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--image", "mattermost/randomimage"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid image name")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
		t.Run("valid te-test image", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--image", "mattermostdevelopment/mm-te-test"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})

		t.Run("valid ee-test image", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--image", "mattermostdevelopment/mm-ee-test"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
		})
	})

	t.Run("env vars", func(t *testing.T) {

		t.Run("valid env vars", func(t *testing.T) {
			expectedEnv := cloud.EnvVarMap{"ENV1": cloud.EnvVar{Value: "test"}, "ENV2": cloud.EnvVar{Value: "test2"}}

			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--env", "ENV1=test,ENV2=test2"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
			assert.Equal(t, expectedEnv, mockCloudClient.patchRequest.PriorityEnv)
		})
		t.Run("clean env takes precedence", func(t *testing.T) {
			expectedEnv := cloud.EnvVarMap{"ENV1": cloud.EnvVar{}, "ENV2": cloud.EnvVar{Value: "test2"}}

			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			resp, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--env", "ENV1=test,ENV2=test2", "--clear-env", "ENV1"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Update of installation")
			assert.Equal(t, expectedEnv, mockCloudClient.patchRequest.PriorityEnv)
		})
		t.Run("invalid env vars", func(t *testing.T) {
			api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
			_, isUserError, err := plugin.runUpdateCommand([]string{"gabesinstall", "--version", "5.30.0", "--env", "ENV1:test,ENV2=test2"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.True(t, isUserError)
			assert.Contains(t, err.Error(), "ENV1:test is not in a valid env format")
		})
	})

}
