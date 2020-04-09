package main

import (
	"encoding/json"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetUpdatedInstallsForUser(t *testing.T) {
	dockerClient := &MockedDockerClient{tagExists: true}
	plugin := Plugin{
		cloudClient: &MockClient{
			mockedCloudInstallations: []*cloud.Installation{
				&cloud.Installation{
					ID:      "id1",
					OwnerID: "owner 1",
				},
				&cloud.Installation{
					ID:      "id2",
					OwnerID: "owner 1",
				},
				&cloud.Installation{
					ID:       "id3",
					OwnerID:  "owner 1",
					DeleteAt: 99999,
				},
				&cloud.Installation{
					ID:      "id4",
					OwnerID: "owner 1",
					State:   cloud.ClusterInstallationStateCreationFailed,
				},
				&cloud.Installation{
					ID:      "id5",
					OwnerID: "owner 2",
				},
			},
		},
		dockerClient: dockerClient,
	}

	api := &plugintest.API{}

	plugin.SetAPI(api)

	t.Run("test deleted mismatched items", func(t *testing.T) {
		_, installationBytes, err := getFakeCloudInstallations()
		require.NoError(t, err)
		api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
		api.On("GetDirectChannel", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.Channel{}, nil)
		api.On("CreatePost", &model.Post{Message: "Cloud installation ID id5 has been removed from your Mattermost app."}).Return(&model.Post{}, nil)
		api.On("CreatePost", &model.Post{Message: "Cloud installation ID id2 has been removed from your Mattermost app."}).Return(&model.Post{}, nil)

		installations, err := plugin.getUpdatedInstallsForUser("owner 1")
		require.NoError(t, err)
		assert.Equal(t, 3, len(installations))
		assert.Equal(t, "id5", installations[0].ID)
		assert.Equal(t, "id1", installations[1].ID)
		assert.Equal(t, "id3", installations[2].ID)
	})

	t.Run("test updatePluginInstalls helper function", func(t *testing.T) {
		pluginInstalls := []*Installation{
			{Name: "one"},
			{Name: "two"},
			{Name: "three"},
			{Name: "four"},
		}

		pluginInstalls = updatePluginInstalls(3, pluginInstalls)
		require.Equal(t, 3, len(pluginInstalls))
		require.Equal(t, "one", pluginInstalls[0].Name)
		require.Equal(t, "two", pluginInstalls[1].Name)
		require.Equal(t, "three", pluginInstalls[2].Name)

		pluginInstalls = updatePluginInstalls(1, pluginInstalls)
		require.Equal(t, 2, len(pluginInstalls))
		require.Equal(t, "one", pluginInstalls[0].Name)
		require.Equal(t, "three", pluginInstalls[1].Name)

		pluginInstalls = updatePluginInstalls(0, pluginInstalls)
		require.Equal(t, 1, len(pluginInstalls))
		require.Equal(t, "three", pluginInstalls[0].Name)

		pluginInstalls = updatePluginInstalls(0, pluginInstalls)
		require.Equal(t, 0, len(pluginInstalls))

		pluginInstalls = updatePluginInstalls(0, pluginInstalls)
		require.Equal(t, 0, len(pluginInstalls))
	})
}

func getFakeCloudInstallations() ([]*Installation, []byte, error) {
	installations := []*Installation{
		&Installation{Name: "installation-one", Installation: cloud.Installation{ID: "id1", OwnerID: "owner 1"}},
		&Installation{Name: "installation-two", Installation: cloud.Installation{ID: "id2", OwnerID: "owner 1"}},
		&Installation{Name: "installation-three", Installation: cloud.Installation{ID: "id3", OwnerID: "owner 1"}},
		&Installation{Name: "installation-four", Installation: cloud.Installation{ID: "id4", OwnerID: "owner 1"}},
		&Installation{Name: "installation-five", Installation: cloud.Installation{ID: "id5", OwnerID: "owner 1"}},
	}
	b, err := json.Marshal(installations)

	return installations, b, err
}
