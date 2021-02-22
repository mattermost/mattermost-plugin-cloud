package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	plugin.SetAPI(api)

	t.Run("get import successfully from valid DNS", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{"indu.dev.cloud.mattermost.com"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation imported")
	})

	t.Run("Invalid DNS display failed to parse", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{"a"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse DNS value")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("import installation successfully with capitalized DNS to show case insensitivity", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{"InDu.DEVLoud.mAtterMost.com"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation imported")
	})

	t.Run("no DNS provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation DNS")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("cloud installation not found based on DNS", func(t *testing.T) {
		plugin := Plugin{
			cloudClient: &MockClient{
				returnNilDNSInstalation: true,
			},
		}
		resp, isUserError, err := plugin.runImportCommand([]string{"name1.dev.cloud.mattermost.com"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no installation for the DNS provided")
		assert.True(t, isUserError)
		assert.Nil(t, resp)

	})

	t.Run("cloud installation not found based on DNS", func(t *testing.T) {
		plugin := Plugin{
			cloudClient: &MockClient{
				returnDNSErrorOverride: errors.New("it broke"),
			},
		}
		resp, isUserError, err := plugin.runImportCommand([]string{"name1.dev.cloud.mattermost.com"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get installation by DNS")
		assert.False(t, isUserError)
		assert.Nil(t, resp)

	})
}
