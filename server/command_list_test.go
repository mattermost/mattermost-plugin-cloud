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
	plugin := Plugin{
		cloudClient: &MockClient{
			overrideGetInstallation: &cloud.Installation{
				ID:    "id3",
				State: cloud.ClusterInstallationStateDeleted,
			},
			mockedCloudInstallations: []*cloud.Installation{
				{
					ID:    "id1",
					State: cloud.ClusterInstallationStateStable,
				}, {
					ID:    "id2",
					State: cloud.ClusterInstallationStateStable,
				}, {
					ID:    "id4",
					State: cloud.ClusterInstallationStateStable,
				}, {
					ID:    "id5",
					State: cloud.ClusterInstallationStateStable,
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

		installations, err := plugin.getUpdatedInstallsForUser("owner 1")
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
		{Name: "installation-one", Installation: cloud.Installation{ID: "id1", OwnerID: "owner 1"}},
		{Name: "installation-two", Installation: cloud.Installation{ID: "id2", OwnerID: "owner 1"}},
		{Name: "installation-three", Installation: cloud.Installation{ID: "id3", OwnerID: "owner 1"}},
		{Name: "installation-four", Installation: cloud.Installation{ID: "id4", OwnerID: "owner 1"}},
		{Name: "installation-five", Installation: cloud.Installation{ID: "id5", OwnerID: "owner 1"}},
	}
	b, err := json.Marshal(installations)

	return installations, b, err
}
