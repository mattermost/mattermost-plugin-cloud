package main

import (
	"strings"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetDebugPacketCommand(t *testing.T) {
	mockedCloudClient := &MockClient{}
	plugin := Plugin{cloudClient: mockedCloudClient}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("GetDirectChannel", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.Channel{}, nil)
	api.On("UploadFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.FileInfo{}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	plugin.SetAPI(api)

	ci1 := &cloud.ClusterInstallation{
		ID: cloud.NewID(),
	}
	mockedCloudClient.mockedCloudClusterInstallations = []*cloud.ClusterInstallation{ci1}

	t.Run("run command successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runGetDebugPacketCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Debug packet generated")
	})

	t.Run("run command successfully with caps in name to show name is case insensitive", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runGetDebugPacketCommand([]string{"GabesInstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Debug packet generated")
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runGetDebugPacketCommand([]string{}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runGetDebugPacketCommand([]string{"gabesinstall2"}, &model.CommandArgs{UserId: "gabeid2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation with the name gabesinstall2 found")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no cluster installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)
		mockedCloudClient.mockedCloudClusterInstallations = []*cloud.ClusterInstallation{}

		resp, isUserError, err := plugin.runGetDebugPacketCommand([]string{"gabesinstall", "version"}, &model.CommandArgs{UserId: "gabeid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no cluster installations found for installation")
		assert.False(t, isUserError)
		assert.Nil(t, resp)
	})
}

func TestExecGetDebugPacketErrors(t *testing.T) {
	clusterInstallation := &cloud.ClusterInstallation{
		ID: cloud.NewID(),
	}

	t.Run("returns error when debug data is nil", func(t *testing.T) {
		mockedCloudClient := &MockClient{
			mockedCloudClusterInstallations: []*cloud.ClusterInstallation{clusterInstallation},
			execDebugPacket: func(clusterInstallationID string) ([]byte, error) {
				return nil, nil
			},
		}
		plugin := Plugin{cloudClient: mockedCloudClient}

		err := plugin.execGetDebugPacket("installation-id", "user-id", "installation-name")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no debug data returned")
	})

	t.Run("returns direct channel app error", func(t *testing.T) {
		mockedCloudClient := &MockClient{
			mockedCloudClusterInstallations: []*cloud.ClusterInstallation{clusterInstallation},
		}
		plugin := Plugin{cloudClient: mockedCloudClient, BotUserID: "bot-id"}

		api := &plugintest.API{}
		api.On("GetDirectChannel", "user-id", "bot-id").Return(nil, model.NewAppError("test", "get_direct_channel_failed", nil, "direct channel failed", 500))
		plugin.SetAPI(api)

		err := plugin.execGetDebugPacket("installation-id", "user-id", "installation-name")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to get direct channel")
		assert.Contains(t, err.Error(), "direct channel failed")
	})

	t.Run("returns upload file app error", func(t *testing.T) {
		mockedCloudClient := &MockClient{
			mockedCloudClusterInstallations: []*cloud.ClusterInstallation{clusterInstallation},
		}
		plugin := Plugin{cloudClient: mockedCloudClient, BotUserID: "bot-id"}

		api := &plugintest.API{}
		api.On("GetDirectChannel", "user-id", "bot-id").Return(&model.Channel{Id: "dm-channel-id"}, nil)
		api.On("UploadFile", []byte("mocked debug packet output"), "dm-channel-id", mock.AnythingOfType("string")).Return(nil, model.NewAppError("test", "upload_file_failed", nil, "upload failed", 500))
		plugin.SetAPI(api)

		err := plugin.execGetDebugPacket("installation-id", "user-id", "installation-name")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to upload debug file")
		assert.Contains(t, err.Error(), "upload failed")
	})
}
