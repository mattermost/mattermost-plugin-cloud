package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetUpdatedInstallsForUser(t *testing.T) {
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
