package main

import (
	"strings"
	"testing"

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
}
