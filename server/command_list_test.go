package main

import (
	"encoding/json"
	"strings"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
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
						ID:            "id1",
						State:         cloud.InstallationStateStable,
						License:       "secret-license",
						MattermostEnv: cloud.EnvVarMap{"secret": cloud.EnvVar{Value: "supersecret"}},
						PriorityEnv:   cloud.EnvVarMap{"priority": cloud.EnvVar{Value: "prioritysecret"}},
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

	t.Run("test sensitivity", func(t *testing.T) {
		pluginInstalls, installationBytes, err := getFakePluginInstallations()
		require.NoError(t, err)
		api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
		api.On("GetDirectChannel", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)

		t.Run("with sensitive", func(t *testing.T) {
			installations, err := plugin.getUpdatedInstallsForUserWithSensitive("owner 1")
			require.NoError(t, err)
			require.Equal(t, len(pluginInstalls), len(installations))
			assert.Equal(t, "id1", installations[0].ID)
			assert.Equal(t, "secret-license", installations[0].License)
			assert.NotNil(t, installations[0].MattermostEnv)
			assert.NotNil(t, installations[0].PriorityEnv)
			t.Log(installations[0].State)
		})

		t.Run("without sensitive", func(t *testing.T) {
			installations, err := plugin.getUpdatedInstallsForUserWithoutSensitive("owner 1")
			require.NoError(t, err)
			require.Equal(t, len(pluginInstalls), len(installations))
			assert.Equal(t, "id1", installations[0].ID)
			assert.Equal(t, "hidden", installations[0].License)
			assert.Nil(t, installations[0].MattermostEnv)
			assert.Nil(t, installations[0].PriorityEnv)
		})
	})

	t.Run("test deleted installations", func(t *testing.T) {
		pluginInstalls, installationBytes, err := getFakePluginInstallations()
		require.NoError(t, err)
		api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
		api.On("GetDirectChannel", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)

		installations, err := plugin.getUpdatedInstallsForUserWithoutSensitive("owner 1")
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

func TestGetUpdatedInstallsForUserWithoutSensitiveDoesNotMutateRefreshSource(t *testing.T) {
	cloudInstall := &cloud.InstallationDTO{Installation: &cloud.Installation{
		ID:            "id1",
		OwnerID:       "owner 1",
		State:         cloud.InstallationStateStable,
		License:       "secret-license",
		MattermostEnv: cloud.EnvVarMap{"secret": cloud.EnvVar{Value: "supersecret"}},
		PriorityEnv:   cloud.EnvVarMap{"priority": cloud.EnvVar{Value: "prioritysecret"}},
	}}
	plugin := Plugin{
		cloudClient:  &MockClient{mockedCloudInstallationsDTO: []*cloud.InstallationDTO{cloudInstall}},
		dockerClient: &MockedDockerClient{tagExists: true},
	}
	api := &plugintest.API{}
	_, installationBytes, err := getFakePluginInstallations()
	require.NoError(t, err)
	api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
	plugin.SetAPI(api)

	withoutSensitive, err := plugin.getUpdatedInstallsForUserWithoutSensitive("owner 1")
	require.NoError(t, err)
	require.NotEmpty(t, withoutSensitive)
	assert.Equal(t, "hidden", withoutSensitive[0].License)
	assert.Nil(t, withoutSensitive[0].MattermostEnv)
	assert.Nil(t, withoutSensitive[0].PriorityEnv)

	withSensitive, err := plugin.getUpdatedInstallsForUserWithSensitive("owner 1")
	require.NoError(t, err)
	require.NotEmpty(t, withSensitive)
	assert.Equal(t, "secret-license", withSensitive[0].License)
	assert.Equal(t, "supersecret", withSensitive[0].MattermostEnv["secret"].Value)
	assert.Equal(t, "prioritysecret", withSensitive[0].PriorityEnv["priority"].Value)
}

func TestGetUpdatedSharedInstallationsWithoutSensitiveDoesNotMutateRefreshSource(t *testing.T) {
	plugin := Plugin{
		cloudClient: &MockClient{overrideGetInstallationDTO: &cloud.InstallationDTO{Installation: &cloud.Installation{
			ID:            "sharedid",
			OwnerID:       "owner 1",
			State:         cloud.InstallationStateStable,
			License:       "shared-secret-license",
			MattermostEnv: cloud.EnvVarMap{"sharedsecret": cloud.EnvVar{Value: "sharedsecretvalue"}},
			PriorityEnv:   cloud.EnvVarMap{"sharedpriority": cloud.EnvVar{Value: "sharedpriorityvalue"}},
		}}},
		dockerClient: &MockedDockerClient{tagExists: true},
	}
	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"sharedid\", \"OwnerID\": \"owner 1\", \"Name\": \"shared\", \"Shared\": true}]"), nil)
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
	plugin.SetAPI(api)

	withoutSensitive, err := plugin.getUpdatedSharedInstallations(true)
	require.NoError(t, err)
	require.Len(t, withoutSensitive, 1)
	assert.Equal(t, "hidden", withoutSensitive[0].License)
	assert.Nil(t, withoutSensitive[0].MattermostEnv)
	assert.Nil(t, withoutSensitive[0].PriorityEnv)

	withSensitive, err := plugin.getUpdatedSharedInstallations(false)
	require.NoError(t, err)
	require.Len(t, withSensitive, 1)
	assert.Equal(t, "shared-secret-license", withSensitive[0].License)
	assert.Equal(t, "sharedsecretvalue", withSensitive[0].MattermostEnv["sharedsecret"].Value)
	assert.Equal(t, "sharedpriorityvalue", withSensitive[0].PriorityEnv["sharedpriority"].Value)
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
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
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

	t.Run("no shared installations", func(t *testing.T) {
		api.On("KVGet").Unset()
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\"}]"), nil)

		resp, isUserError, err := plugin.runListCommand([]string{"--shared-installations"}, &model.CommandArgs{UserId: "gabeid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "No installations found."))
	})

	t.Run("shared installations", func(t *testing.T) {
		api.On("KVGet").Unset()
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"joramid\", \"Shared\": true}]"), nil)

		resp, isUserError, err := plugin.runListCommand([]string{"--shared-installations"}, &model.CommandArgs{UserId: "gabeid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "someid"))
	})

	t.Run("shared installations, hidden env", func(t *testing.T) {
		api.On("KVGet").Unset()
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"sharedid\", \"Shared\": true}]"), nil)
		plugin.cloudClient = &MockClient{overrideGetInstallationDTO: &cloud.InstallationDTO{
			Installation: &cloud.Installation{
				ID:      "someid",
				OwnerID: "sharedid",
				License: "license-secret",
				MattermostEnv: cloud.EnvVarMap{
					"testkey": cloud.EnvVar{Value: "testval"},
				},
				PriorityEnv: cloud.EnvVarMap{
					"prioritykey": cloud.EnvVar{Value: "priorityval"},
				},
			},
		}}

		resp, isUserError, err := plugin.runListCommand([]string{"--shared-installations"}, &model.CommandArgs{UserId: "gabeid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "someid"))
		assert.False(t, strings.Contains(resp.Text, "\"License\""))
		assert.False(t, strings.Contains(resp.Text, "license-secret"))
		assert.False(t, strings.Contains(resp.Text, "\"MattermostEnv\""))
		assert.False(t, strings.Contains(resp.Text, "testkey"))
		assert.False(t, strings.Contains(resp.Text, "testval"))
		assert.False(t, strings.Contains(resp.Text, "\"PriorityEnv\""))
		assert.False(t, strings.Contains(resp.Text, "prioritykey"))
		assert.False(t, strings.Contains(resp.Text, "priorityval"))
	})
}

func TestListCommandCleansUpDeletedInstallations(t *testing.T) {
	plugin := Plugin{
		cloudClient: &MockClient{
			overrideGetInstallationDTO: &cloud.InstallationDTO{Installation: &cloud.Installation{
				ID:      "deletedid",
				OwnerID: "ownerid",
				State:   cloud.InstallationStateDeleted,
			}},
		},
		dockerClient: &MockedDockerClient{tagExists: true},
	}
	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"deletedid\", \"OwnerID\": \"ownerid\", \"Name\": \"deletedinstall\"}]"), nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil).Once()
	api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
	plugin.SetAPI(api)

	resp, isUserError, err := plugin.runListCommand([]string{}, &model.CommandArgs{UserId: "ownerid"})
	require.NoError(t, err)
	assert.False(t, isUserError)
	assert.Contains(t, resp.Text, "deletedinstall [ DELETED ]")
	api.AssertExpectations(t)
}
