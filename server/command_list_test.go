package main

import (
	"encoding/json"
	"strings"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetUpdatedInstallsForUser(t *testing.T) {
	plugin := Plugin{
		cloudClient: &MockClient{
			overrideGetInstallationDTO: &cloud.InstallationDTO{Installation: &cloud.Installation{
				ID:    "id3",
				State: cloud.InstallationStateDeleted,
			}},
			mockedCloudInstallationsDTO: []*cloud.InstallationDTO{
				{
					Installation: &cloud.Installation{
						ID:    "id1",
						State: cloud.InstallationStateStable,
					},
				},
				{
					Installation: &cloud.Installation{
						ID:    "id2",
						State: cloud.InstallationStateStable,
					},
				},
				{
					Installation: &cloud.Installation{
						ID:    "id4",
						State: cloud.InstallationStateStable,
					},
				},
				{
					Installation: &cloud.Installation{
						ID:    "id5",
						State: cloud.InstallationStateStable,
					},
				},
			},
		},
		dockerClient: &MockedDockerClient{tagExists: true},
	}

	api := &plugintest.API{}

	plugin.SetAPI(api)

	t.Run("test deleted installations", func(t *testing.T) {
		pluginInstalls, installationBytes, err := getFakePluginInstallations()
		require.NoError(t, err)
		api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
		api.On("GetDirectChannel", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("LogWarn", mock.AnythingOfTypeArgument("string")).Return(nil)

		installations, err := plugin.getUpdatedInstallsForUser("owner 1", true)
		require.NoError(t, err)
		require.Equal(t, len(pluginInstalls), len(installations))
		assert.Equal(t, "id1", installations[0].ID)
		assert.Equal(t, "id2", installations[1].ID)
		assert.Equal(t, "id4", installations[3].ID)
		assert.Equal(t, "id5", installations[4].ID)
		assert.Contains(t, installations[2].Name, "installation-three")
		assert.Contains(t, installations[2].Name, "DELETED")
	})
}

func getFakePluginInstallations() ([]*Installation, []byte, error) {
	installations := []*Installation{
		{Name: "installation-one", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id1", OwnerID: "owner 1"}}},
		{Name: "installation-two", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id2", OwnerID: "owner 1"}}},
		{Name: "installation-three", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id3", OwnerID: "owner 1"}}},
		{Name: "installation-four", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id4", OwnerID: "owner 1"}}},
		{Name: "installation-five", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id5", OwnerID: "owner 1"}}},
	}
	b, err := json.Marshal(installations)

	return installations, b, err
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
