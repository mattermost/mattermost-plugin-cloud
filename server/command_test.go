package main

import (
	"strings"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockClient struct {
	mockedCloudClustersDTO          []*cloud.ClusterDTO
	mockedCloudInstallationsDTO     []*cloud.InstallationDTO
	mockedCloudClusterInstallations []*cloud.ClusterInstallation

	overrideGetInstallationDTO *cloud.InstallationDTO

	err error
}

func (mc *MockClient) GetClusters(request *cloud.GetClustersRequest) ([]*cloud.ClusterDTO, error) {
	return mc.mockedCloudClustersDTO, mc.err
}

func (mc *MockClient) CreateInstallation(request *cloud.CreateInstallationRequest) (*cloud.InstallationDTO, error) {
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid"}}, nil
}

func (mc *MockClient) GetInstallation(installataionID string, request *cloud.GetInstallationRequest) (*cloud.InstallationDTO, error) {
	if mc.overrideGetInstallationDTO != nil {
		return mc.overrideGetInstallationDTO, nil
	}

	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) GetInstallationByDNS(DNS string, request *cloud.GetInstallationRequest) (*cloud.InstallationDTO, error) {
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid"}}, nil
}

func (mc *MockClient) GetInstallations(request *cloud.GetInstallationsRequest) ([]*cloud.InstallationDTO, error) {
	return mc.mockedCloudInstallationsDTO, mc.err
}

func (mc *MockClient) UpdateInstallation(installationID string, request *cloud.PatchInstallationRequest) (*cloud.InstallationDTO, error) {
	return &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "someid", OwnerID: "joramid"}}, nil
}

func (mc *MockClient) DeleteInstallation(installationID string) error {
	return nil
}

func (mc *MockClient) GetClusterInstallations(request *cloud.GetClusterInstallationsRequest) ([]*cloud.ClusterInstallation, error) {
	return mc.mockedCloudClusterInstallationsDTO, nil
}

func (mc *MockClient) RunMattermostCLICommandOnClusterInstallation(clusterInstallationID string, subcommand []string) ([]byte, error) {
	return []byte("mocked command output"), nil
}

func (mc *MockClient) GetGroup(groupID string) (*cloud.Group, error) {
	return &cloud.Group{ID: groupID, Name: "test-group"}, nil
}

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

func TestListCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)
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
		assert.Contains(t, err.Error(), "must specify at least one option: version, license, or size")
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
}

func TestMattermostCLICommand(t *testing.T) {
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

		resp, isUserError, err := plugin.runMattermostCLICommand([]string{"gabesinstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "mocked command output")
	})

	t.Run("run command successfully with caps in name to show name is case insensitive", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runMattermostCLICommand([]string{"GabesInstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "mocked command output")
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runMattermostCLICommand([]string{}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runMattermostCLICommand([]string{""}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no mattermost subcommand", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runMattermostCLICommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an mattermost CLI command")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runMattermostCLICommand([]string{"gabesinstall2", "version"}, &model.CommandArgs{UserId: "gabeid2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no cluster installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
		mockedCloudClient.mockedCloudClusterInstallations = []*cloud.ClusterInstallation{}

		resp, isUserError, err := plugin.runMattermostCLICommand([]string{"gabesinstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no cluster installations found for installation")
		assert.False(t, isUserError)
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

	t.Run("delete installation successfully with caps in name to demonstrate name case insensitivity", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Name\": \"JoramsInstall\"}]"), nil)

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

func TestStatusCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	t.Run("no clusters or installations", func(t *testing.T) {
		t.Run("show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{"--include-clusters=true"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
		})

		t.Run("don't show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.NotContains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
		})
	})

	t.Run("clusters and installations", func(t *testing.T) {
		cluster1 := &cloud.Cluster{
			ID:    cloud.NewID(),
			State: cloud.ClusterStateStable,
		}
		mockedCloudClient.mockedCloudClustersDTO = []*cloud.ClusterDTO{Cluster: cluster1}

		installation1 := &cloud.Installation{
			ID:      cloud.NewID(),
			DNS:     "https://greatawesome.com",
			Size:    "superextralarge",
			Version: "v7.1.44",
			State:   cloud.InstallationStateCreationDNS,
		}
		mockedCloudClient.mockedCloudInstallations = []*cloud.Installation{installation1}

		t.Run("show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{"--include-clusters=true"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
			assert.Contains(t, resp.Text, cluster1.ID)
			assert.Contains(t, resp.Text, cluster1.State)
			assert.Contains(t, resp.Text, installation1.ID)
			assert.Contains(t, resp.Text, installation1.DNS)
			assert.Contains(t, resp.Text, installation1.Size)
			assert.Contains(t, resp.Text, installation1.Version)
			assert.Contains(t, resp.Text, installation1.State)
		})

		t.Run("don't show clusters", func(t *testing.T) {
			resp, isUserError, err := plugin.runStatusCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.NotContains(t, resp.Text, clusterTableHeader)
			assert.Contains(t, resp.Text, installationTableHeader)
			assert.NotContains(t, resp.Text, cluster1.ID)
			assert.NotContains(t, resp.Text, cluster1.State)
			assert.Contains(t, resp.Text, installation1.ID)
			assert.Contains(t, resp.Text, installation1.DNS)
			assert.Contains(t, resp.Text, installation1.Size)
			assert.Contains(t, resp.Text, installation1.Version)
			assert.Contains(t, resp.Text, installation1.State)
		})
	})

	t.Run("error", func(t *testing.T) {
		mockedCloudClient.err = errors.New("an error was enountered")

		resp, isUserError, err := plugin.runStatusCommand([]string{"--include-clusters=true"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.False(t, isUserError)
		assert.Nil(t, resp)
	})
}

func TestInfoCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	t.Run("success", func(t *testing.T) {
		resp, isUserError, err := plugin.runInfoCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, manifest.Version)
	})
}

func TestValidInstallationName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"abc", true},
		{"abc123", true},
		{"abcABC123", true},
		{"123", true},
		{"A1", true},
		{"A1-", true},
		{"A1-abc", true},
		{"realllllllllllllllllylongname123123123123123", true},
		{"bad.", false},
		{"bad\\", false},
		{"bad/", false},
		{"bad,", false},
		{"bad:", false},
		{"bad;", false},
		{"bad_", false},
		{"123.,", false},
		{".", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.valid, validInstallationName(test.name))
		})
	}
}
