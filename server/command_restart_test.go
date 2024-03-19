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

func TestRestartCommand(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	plugin.SetAPI(api)

	t.Run("restart installation successfully", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "Installation gabesinstall restarting now."))
	})

	t.Run("restart installation successfully with caps in name to demonstrate name case insensitivity", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"GabesInstall\"}]"), nil)

		resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
		require.Nil(t, err)
		assert.False(t, isUserError)
		assert.True(t, strings.Contains(resp.Text, "Installation gabesinstall restarting now."))
	})

	t.Run("don't restart with wrong owner", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte("[{\"ID\": \"someid\", \"OwnerID\": \"gabeid\", \"Name\": \"gabesinstall\"}]"), nil)

		resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "no installation with the name gabesinstall found"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no installations", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

		resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid2"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "no installation with the name gabesinstall found"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("no name provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runRestartCommand([]string{}, &model.CommandArgs{UserId: "gabeid"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)

		resp, isUserError, err = plugin.runRestartCommand([]string{""}, &model.CommandArgs{UserId: "gabeid"})
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "must provide an installation name"))
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("shared installations", func(t *testing.T) {
		t.Run("not shared", func(t *testing.T) {
			installBytes, err := json.Marshal([]*Installation{{
				Name:               "gabesinstall",
				Shared:             false,
				AllowSharedUpdates: false,
				InstallationDTO:    cloud.InstallationDTO{Installation: &cloud.Installation{ID: cloud.NewID()}},
			}})
			require.NoError(t, err)
			api.On("KVGet").Unset()
			api.On("KVGet", mock.AnythingOfType("string")).Return(installBytes, nil)

			resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall", "--shared-installation"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no installation with the name gabesinstall found")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("shared, but not with updates allowed", func(t *testing.T) {
			installBytes, err := json.Marshal([]*Installation{{
				Name:               "gabesinstall",
				Shared:             true,
				AllowSharedUpdates: false,
				InstallationDTO:    cloud.InstallationDTO{Installation: &cloud.Installation{ID: cloud.NewID()}},
			}})
			require.NoError(t, err)
			api.On("KVGet").Unset()
			api.On("KVGet", mock.AnythingOfType("string")).Return(installBytes, nil)

			resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall", "--shared-installation"}, &model.CommandArgs{UserId: "gabeid"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no installation with the name gabesinstall found")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})

		t.Run("shared and updates allowed", func(t *testing.T) {
			installBytes, err := json.Marshal([]*Installation{{
				Name:               "gabesinstall",
				Shared:             true,
				AllowSharedUpdates: true,
				InstallationDTO:    cloud.InstallationDTO{Installation: &cloud.Installation{ID: cloud.NewID()}},
			}})
			require.NoError(t, err)
			api.On("KVGet").Unset()
			api.On("KVGet", mock.AnythingOfType("string")).Return(installBytes, nil)

			resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall", "--shared-installation"}, &model.CommandArgs{UserId: "gabeid"})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation gabesinstall restarting now.")
		})

		t.Run("missing --shared-installation flag", func(t *testing.T) {
			installBytes, err := json.Marshal([]*Installation{{
				Name:               "gabesinstall",
				Shared:             true,
				AllowSharedUpdates: true,
				InstallationDTO:    cloud.InstallationDTO{Installation: &cloud.Installation{ID: cloud.NewID()}},
			}})
			require.NoError(t, err)
			api.On("KVGet").Unset()
			api.On("KVGet", mock.AnythingOfType("string")).Return(installBytes, nil)

			resp, isUserError, err := plugin.runRestartCommand([]string{"gabesinstall"}, &model.CommandArgs{UserId: "gabeid"})
			assert.Contains(t, err.Error(), "no installation with the name gabesinstall found")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
	})
}
