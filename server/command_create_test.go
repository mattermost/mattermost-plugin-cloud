package main

import (
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand(t *testing.T) {
	dockerClient := &MockedDockerClient{tagExists: true}
	plugin := Plugin{
		cloudClient:  &MockClient{},
		dockerClient: dockerClient,
	}

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

	t.Run("create installation successfully with capitalized name to show case insensitivity", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"jOrAmTeSt"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation being created.")
	})

	t.Run("block it try to install version below 5.12.0", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--version", "5.8.3"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("allow it try to install version greater than 5.12.0", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--version", "5.20.1"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation being created.")
	})

	t.Run("allow it try to install version called latest", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--version", "latest"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation being created.")
	})

	t.Run("docker tag", func(t *testing.T) {

		t.Run("valid", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--version", "totallyisreal"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run("invalid", func(t *testing.T) {
			dockerClient.tagExists = false
			defer func() { dockerClient.tagExists = true }()

			resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--version", "totallyisnotreal"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
	})

	t.Run("invalid license", func(t *testing.T) {
		resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--license", "e30"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid license option")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("affinity", func(t *testing.T) {
		t.Run("invalid", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--affinity", "banana"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid affinity option banana, must be isolated or multitenant")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("isolated", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--affinity", "isolated"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run("multitenant", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest", "--affinity", "multitenant"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})
	})

	t.Run("database", func(t *testing.T) {
		t.Run("invalid", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--database", "sqlite"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid database option sqlite")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run(cloud.InstallationDatabaseMysqlOperator, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--database", cloud.InstallationDatabaseMysqlOperator}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run(cloud.InstallationDatabaseSingleTenantRDSMySQL, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--database", cloud.InstallationDatabaseSingleTenantRDSMySQL}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run(cloud.InstallationDatabaseMultiTenantRDSMySQL, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--database", cloud.InstallationDatabaseMultiTenantRDSMySQL}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run(cloud.InstallationDatabaseSingleTenantRDSPostgres, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--database", cloud.InstallationDatabaseSingleTenantRDSPostgres}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run(cloud.InstallationDatabaseMultiTenantRDSPostgres, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--database", cloud.InstallationDatabaseMultiTenantRDSPostgres}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})
	})

	t.Run("filestore", func(t *testing.T) {
		t.Run("invalid", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--filestore", "usb-drive"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid filestore option usb-drive")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("invalid license option", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--filestore", cloud.InstallationFilestoreMultiTenantAwsS3, "--license", licenseOptionTE}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "filestore option aws-multitenant-s3 requires license option e20")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run(cloud.InstallationFilestoreMinioOperator, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--filestore", cloud.InstallationFilestoreMinioOperator}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run(cloud.InstallationFilestoreAwsS3, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--filestore", cloud.InstallationFilestoreAwsS3}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})

		t.Run(cloud.InstallationFilestoreMultiTenantAwsS3, func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--filestore", cloud.InstallationFilestoreMultiTenantAwsS3}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})
	})

	t.Run("image", func(t *testing.T) {
		t.Run("valid image name", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--image", "mattermost/mm-ee-test"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})
		t.Run("invalid image name", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"gabetest", "--image", "mattermost/randomimage"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid image name")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
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

	t.Run("groups", func(t *testing.T) {
		groupID := model.NewId()
		plugin.configuration = &configuration{
			GroupID: groupID,
		}

		t.Run("create installation successfully", func(t *testing.T) {
			resp, isUserError, err := plugin.runCreateCommand([]string{"joramtest"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation being created.")
		})
	})
}
