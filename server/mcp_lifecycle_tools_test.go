package main

import (
	"encoding/json"
	"errors"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallationMCP(t *testing.T) {
	t.Run("authorizes caller from trusted user context", func(t *testing.T) {
		plugin, _, api := newMCPToolsTestPlugin(t, nil)

		missingUserSession, missingUserCleanup := connectMCPToolsClient(t, plugin, "")
		defer missingUserCleanup()
		result, err := callMCPTool(t, missingUserSession, mcpCreateInstallationToolName, map[string]any{"name": "new"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no Mattermost user ID")

		plugin.configuration.AllowedEmailDomain = "mattermost.com"
		api.On("GetUser", "owner").Return(&model.User{Id: "owner", Email: "owner@example.com"}, nil).Once()

		deniedSession, deniedCleanup := connectMCPToolsClient(t, plugin, "owner")
		defer deniedCleanup()
		result, err = callMCPTool(t, deniedSession, mcpCreateInstallationToolName, map[string]any{"name": "new"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "not authorized")
	})

	t.Run("creates through provisioner before KV store and redacts output", func(t *testing.T) {
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, nil)
		plugin.dockerClient = &MockedDockerClient{tagExists: true, digest: "sha256:digest"}
		plugin.configuration.E20License = "raw-license-value"
		api.On("KVCompareAndSet").Unset()
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				require.NotNil(t, cloudClient.creationRequest)
				assert.Equal(t, "raw-license-value", cloudClient.creationRequest.License)
				assert.Equal(t, "env-secret-value", cloudClient.creationRequest.PriorityEnv["SECRET_ENV"].Value)
			}).
			Return(true, nil)

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpCreateInstallationToolName, map[string]any{
			"name":    "Created",
			"version": "9.5.0",
			"license": licenseOptionE20,
			"env": map[string]any{
				"SECRET_ENV": "env-secret-value",
			},
		})
		require.NoError(t, err)
		require.False(t, result.IsError)

		output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.Equal(t, "creation_requested", output.Result.Status)
		assert.Equal(t, "created", output.Result.Installation.Name)
		assert.Equal(t, "9.5.0", output.Result.Installation.VersionTag)
		assert.NotEmpty(t, output.Result.Message)
		assertMCPResultRedacted(t, result, "raw-license-value", "env-secret-value", "MattermostEnv", "PriorityEnv", "License")
	})

	t.Run("returns validation and provisioner errors as tool errors", func(t *testing.T) {
		tests := []struct {
			name        string
			args        map[string]any
			errContains string
			setup       func(*Plugin, *MockClient)
		}{
			{"duplicate name", map[string]any{"name": "taken"}, "already exists", nil},
			{"invalid license", map[string]any{"name": "new", "license": "e30"}, "invalid license option", nil},
			{"invalid image", map[string]any{"name": "new", "image": "mattermost/unknown"}, "invalid image name", nil},
			{"invalid size", map[string]any{"name": "new", "size": "huge"}, "Invalid size", nil},
			{"invalid database", map[string]any{"name": "new", "database": "sqlite"}, "invalid database option", nil},
			{"invalid filestore", map[string]any{"name": "new", "filestore": "local"}, "invalid filestore option", nil},
			{"bad docker tag", map[string]any{"name": "new", "version": "9.5.0"}, "is not a valid docker tag", func(p *Plugin, _ *MockClient) {
				p.dockerClient = &MockedDockerClient{tagExists: false}
			}},
			{"provisioner error", map[string]any{"name": "new"}, "failed to create installation", func(_ *Plugin, c *MockClient) {
				c.createErr = errors.New("provisioner unavailable")
			}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				plugin, cloudClient, _ := newMCPToolsTestPlugin(t, []*Installation{serviceTestInstall("taken-id", "taken", "owner")})
				if test.name != "duplicate name" {
					plugin, cloudClient, _ = newMCPToolsTestPlugin(t, nil)
				}
				if test.setup != nil {
					test.setup(plugin, cloudClient)
				}

				session, cleanup := connectMCPToolsClient(t, plugin, "owner")
				defer cleanup()
				result, err := callMCPTool(t, session, mcpCreateInstallationToolName, test.args)
				require.NoError(t, err)
				assert.True(t, result.IsError)
				assert.Contains(t, mcpToolText(t, result), test.errContains)
			})
		}
	})
}

func TestUpdateInstallationMCP(t *testing.T) {
	t.Run("updates owned install and redacts env and license material", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, _ := newMCPToolsTestPlugin(t, []*Installation{install})
		plugin.configuration.E20License = "raw-license-value"
		plugin.dockerClient = &MockedDockerClient{tagExists: true, digest: "sha256:new"}

		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{
			"name":      "install",
			"version":   "9.5.0",
			"license":   licenseOptionE20,
			"image":     imageTE,
			"size":      "miniHA",
			"set_env":   map[string]any{"SECRET_ENV": "secret-value"},
			"clear_env": []any{"OLD_ENV"},
		})
		require.NoError(t, err)
		require.False(t, result.IsError)

		output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.Equal(t, "update_requested", output.Result.Status)
		assert.ElementsMatch(t, []string{"env", "image", "license", "size", "version"}, output.Result.ChangedFields)
		assert.Equal(t, []string{"SECRET_ENV"}, output.Result.ChangedEnvKeys)
		assert.Equal(t, []string{"OLD_ENV"}, output.Result.ClearedEnvKeys)
		require.NotNil(t, cloudClient.patchRequest.License)
		assert.Equal(t, "raw-license-value", *cloudClient.patchRequest.License)
		assertMCPResultRedacted(t, result, "raw-license-value", "secret-value", "MattermostEnv", "PriorityEnv", "License")
	})

	t.Run("enforces shared update gates and rejects read-only scope", func(t *testing.T) {
		sharedRead := serviceTestInstall("shared-read-id", "SharedRead", "owner")
		sharedRead.Shared = true
		sharedUpdate := serviceTestInstall("shared-update-id", "SharedUpdate", "owner")
		sharedUpdate.Shared = true
		sharedUpdate.AllowSharedUpdates = true
		plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{sharedRead, sharedUpdate})

		session, cleanup := connectMCPToolsClient(t, plugin, "other")
		defer cleanup()
		result, err := callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{"name": "sharedread", "scope": "updatable", "size": "miniHA"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no installation with the name sharedread found")

		result, err = callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{"name": "sharedupdate", "scope": "shared", "size": "miniHA"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "shared scope is read-only")

		result, err = callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{"name": "sharedupdate", "scope": "updatable", "size": "miniHA"})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.Equal(t, "update_requested", output.Result.Status)
	})

	t.Run("validates refs empty updates not found and provisioner write ordering", func(t *testing.T) {
		install := serviceTestInstall("install-id", "Install", "owner")
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, []*Installation{install})
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		for _, args := range []map[string]any{
			{},
			{"installation_id": "install-id", "name": "install", "size": "miniHA"},
		} {
			result, err := callMCPTool(t, session, mcpUpdateInstallationToolName, args)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, mcpToolText(t, result), "must provide exactly one installation id or name")
		}

		result, err := callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{"name": "install"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "must specify at least one option")

		result, err = callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{"name": "missing", "size": "miniHA"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no installation with the name missing found")

		cloudClient.updateErr = errors.New("provisioner update failed")
		result, err = callMCPTool(t, session, mcpUpdateInstallationToolName, map[string]any{"name": "install", "size": "miniHA"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "failed to update installation")
		api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
	})
}

func TestRestartInstallationMCP(t *testing.T) {
	t.Run("restarts owned and updatable shared installs", func(t *testing.T) {
		owned := serviceTestInstall("owned-id", "Owned", "owner")
		sharedRead := serviceTestInstall("shared-read-id", "SharedRead", "owner")
		sharedRead.Shared = true
		sharedUpdate := serviceTestInstall("shared-update-id", "SharedUpdate", "owner")
		sharedUpdate.Shared = true
		sharedUpdate.AllowSharedUpdates = true
		plugin, cloudClient, _ := newMCPToolsTestPlugin(t, []*Installation{owned, sharedRead, sharedUpdate})

		ownerSession, ownerCleanup := connectMCPToolsClient(t, plugin, "owner")
		defer ownerCleanup()
		result, err := callMCPTool(t, ownerSession, mcpRestartInstallationToolName, map[string]any{"installation_id": "owned-id"})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.Equal(t, "restart_requested", output.Result.Status)
		assert.Equal(t, []string{"CLOUD_PLUGIN_RESTART"}, output.Result.ChangedEnvKeys)
		assert.Contains(t, cloudClient.patchRequest.MattermostEnv, "CLOUD_PLUGIN_RESTART")

		otherSession, otherCleanup := connectMCPToolsClient(t, plugin, "other")
		defer otherCleanup()
		result, err = callMCPTool(t, otherSession, mcpRestartInstallationToolName, map[string]any{"name": "sharedread", "scope": "updatable"})
		require.NoError(t, err)
		assert.True(t, result.IsError)

		result, err = callMCPTool(t, otherSession, mcpRestartInstallationToolName, map[string]any{"name": "sharedupdate", "scope": "updatable"})
		require.NoError(t, err)
		require.False(t, result.IsError)
		assert.Equal(t, "shared-update-id", cloudClient.patchInstallationID)
	})
}

func TestHibernateAndWakeInstallationMCP(t *testing.T) {
	stable := serviceTestInstall("stable-id", "Stable", "owner")
	hibernating := serviceTestInstall("hibernating-id", "Hibernating", "owner")
	hibernating.State = cloud.InstallationStateHibernating
	plugin, cloudClient, _ := newMCPToolsTestPlugin(t, []*Installation{stable, hibernating})
	cloudClient.mockedCloudInstallationsDTO = serviceDTOs(stable, hibernating)
	session, cleanup := connectMCPToolsClient(t, plugin, "owner")
	defer cleanup()

	result, err := callMCPTool(t, session, mcpHibernateInstallationToolName, map[string]any{"name": "stable"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
	assert.Equal(t, "hibernate_requested", output.Result.Status)
	assert.Equal(t, "stable-id", cloudClient.hibernatedInstallationID)

	result, err = callMCPTool(t, session, mcpHibernateInstallationToolName, map[string]any{"name": "hibernating"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, mcpToolText(t, result), "must be stable to hibernate")

	result, err = callMCPTool(t, session, mcpWakeInstallationToolName, map[string]any{"name": "hibernating"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	output = decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
	assert.Equal(t, "wake_requested", output.Result.Status)
	assert.Equal(t, "hibernating-id", cloudClient.wokenInstallationID)

	result, err = callMCPTool(t, session, mcpWakeInstallationToolName, map[string]any{"name": "stable"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, mcpToolText(t, result), "must be hibernating to wake up")
}

func TestSetInstallationSharingMCP(t *testing.T) {
	install := serviceTestInstall("install-id", "Install", "owner")
	plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{install})
	plugin.cloudClient.(*MockClient).mockedCloudInstallationsDTO = serviceDTOs(install)
	session, cleanup := connectMCPToolsClient(t, plugin, "owner")
	defer cleanup()

	result, err := callMCPTool(t, session, mcpSetInstallationSharingToolName, map[string]any{"name": "install", "shared": true, "allow_updates": true})
	require.NoError(t, err)
	require.False(t, result.IsError)
	output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
	assert.Equal(t, "sharing_updated", output.Result.Status)
	assert.True(t, output.Result.Installation.Shared)
	assert.True(t, output.Result.Installation.AllowSharedUpdates)

	result, err = callMCPTool(t, session, mcpSetInstallationSharingToolName, map[string]any{"name": "install", "shared": false, "allow_updates": true})
	require.NoError(t, err)
	require.False(t, result.IsError)
	output = decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
	assert.False(t, output.Result.Installation.Shared)
	assert.False(t, output.Result.Installation.AllowSharedUpdates)

	otherSession, otherCleanup := connectMCPToolsClient(t, plugin, "other")
	defer otherCleanup()
	result, err = callMCPTool(t, otherSession, mcpSetInstallationSharingToolName, map[string]any{"name": "install", "shared": true})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, mcpToolText(t, result), "no installation with the name install found")
}

func TestSetDeletionLockMCP(t *testing.T) {
	t.Run("locks unlocks enforces limits and idempotency", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		locked := serviceTestInstall("locked-id", "Locked", "owner")
		locked.DeletionLocked = true
		plugin, cloudClient, _ := newMCPToolsTestPlugin(t, []*Installation{target, locked})
		cloudClient.mockedCloudInstallationsDTO = serviceDTOs(target, locked)
		plugin.configuration.DeletionLockInstallationsAllowedPerPerson = "1"
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpSetDeletionLockToolName, map[string]any{"name": "target", "locked": true})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "at most 1 installations locked")

		result, err = callMCPTool(t, session, mcpSetDeletionLockToolName, map[string]any{"name": "locked", "locked": true})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.True(t, output.Result.Installation.DeletionLocked)
		assert.Equal(t, "locked-id", cloudClient.lockedInstallationID)

		result, err = callMCPTool(t, session, mcpSetDeletionLockToolName, map[string]any{"name": "locked", "locked": false})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output = decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.False(t, output.Result.Installation.DeletionLocked)
		assert.Equal(t, "locked-id", cloudClient.unlockedInstallationID)
	})

	t.Run("does not expose other users locks", func(t *testing.T) {
		target := serviceTestInstall("target-id", "Target", "owner")
		plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{target})
		plugin.cloudClient.(*MockClient).mockedCloudInstallationsDTO = serviceDTOs(target)
		session, cleanup := connectMCPToolsClient(t, plugin, "other")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpSetDeletionLockToolName, map[string]any{"name": "target", "locked": true})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no installations found")
	})
}

func TestMCPLifecycleProvisionerErrors(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		args      map[string]any
		installs  []*Installation
		setup     func(*MockClient)
		assertion func(*testing.T, *MockClient)
	}{
		{
			name:     "restart surfaces update error",
			toolName: mcpRestartInstallationToolName,
			args:     map[string]any{"name": "stable"},
			installs: []*Installation{serviceTestInstall("stable-id", "Stable", "owner")},
			setup: func(cloudClient *MockClient) {
				cloudClient.updateErr = errors.New("provisioner restart failed")
			},
			assertion: func(t *testing.T, cloudClient *MockClient) {
				assert.Equal(t, "stable-id", cloudClient.patchInstallationID)
			},
		},
		{
			name:     "hibernate surfaces hibernate error",
			toolName: mcpHibernateInstallationToolName,
			args:     map[string]any{"name": "stable"},
			installs: []*Installation{serviceTestInstall("stable-id", "Stable", "owner")},
			setup: func(cloudClient *MockClient) {
				cloudClient.hibernateErr = errors.New("provisioner hibernate failed")
			},
			assertion: func(t *testing.T, cloudClient *MockClient) {
				assert.Equal(t, "stable-id", cloudClient.hibernatedInstallationID)
			},
		},
		{
			name:     "wake surfaces wake error",
			toolName: mcpWakeInstallationToolName,
			args:     map[string]any{"name": "hibernating"},
			installs: []*Installation{func() *Installation {
				install := serviceTestInstall("hibernating-id", "Hibernating", "owner")
				install.State = cloud.InstallationStateHibernating
				return install
			}()},
			setup: func(cloudClient *MockClient) {
				cloudClient.wakeErr = errors.New("provisioner wake failed")
			},
			assertion: func(t *testing.T, cloudClient *MockClient) {
				assert.Equal(t, "hibernating-id", cloudClient.wokenInstallationID)
			},
		},
		{
			name:     "lock surfaces lock error",
			toolName: mcpSetDeletionLockToolName,
			args:     map[string]any{"name": "stable", "locked": true},
			installs: []*Installation{serviceTestInstall("stable-id", "Stable", "owner")},
			setup: func(cloudClient *MockClient) {
				cloudClient.lockErr = errors.New("provisioner lock failed")
			},
			assertion: func(t *testing.T, cloudClient *MockClient) {
				assert.Equal(t, "stable-id", cloudClient.lockedInstallationID)
			},
		},
		{
			name:     "unlock surfaces unlock error",
			toolName: mcpSetDeletionLockToolName,
			args:     map[string]any{"name": "locked", "locked": false},
			installs: []*Installation{func() *Installation {
				install := serviceTestInstall("locked-id", "Locked", "owner")
				install.DeletionLocked = true
				return install
			}()},
			setup: func(cloudClient *MockClient) {
				cloudClient.unlockErr = errors.New("provisioner unlock failed")
			},
			assertion: func(t *testing.T, cloudClient *MockClient) {
				assert.Equal(t, "locked-id", cloudClient.unlockedInstallationID)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plugin, cloudClient, api := newMCPToolsTestPlugin(t, test.installs)
			test.setup(cloudClient)
			api.On("KVCompareAndSet").Unset()

			session, cleanup := connectMCPToolsClient(t, plugin, "owner")
			defer cleanup()

			result, err := callMCPTool(t, session, test.toolName, test.args)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, mcpToolText(t, result), "provisioner")
			test.assertion(t, cloudClient)
			api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
		})
	}
}

func TestDeleteInstallationMCP(t *testing.T) {
	t.Run("requires confirmation and calls provisioner before local delete", func(t *testing.T) {
		target := serviceTestInstall("delete-id", "DeleteMe", "owner")
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, []*Installation{target})
		api.On("KVCompareAndSet").Unset()
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				assert.Equal(t, "delete-id", cloudClient.deletedInstallationID)
			}).
			Return(true, nil)
		session, cleanup := connectMCPToolsClient(t, plugin, "owner")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpDeleteInstallationToolName, map[string]any{"name": "deleteme"})
		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "missing properties")
		assert.Contains(t, err.Error(), "confirm_name")
		assert.Empty(t, cloudClient.deletedInstallationID)

		result, err = callMCPTool(t, session, mcpDeleteInstallationToolName, map[string]any{"name": "deleteme", "confirm_name": "wrong"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "does not match installation name")
		assert.Empty(t, cloudClient.deletedInstallationID)

		result, err = callMCPTool(t, session, mcpDeleteInstallationToolName, map[string]any{"name": "deleteme", "confirm_name": "DeleteMe"})
		require.NoError(t, err)
		require.False(t, result.IsError)
		output := decodeMCPStructuredOutput[InstallationActionMCPOutput](t, result)
		assert.Equal(t, "delete_requested", output.Result.Status)
		assert.Equal(t, "delete-id", cloudClient.deletedInstallationID)
	})

	t.Run("blocks wrong owner and preserves KV on provisioner failure", func(t *testing.T) {
		target := serviceTestInstall("delete-id", "DeleteMe", "owner")
		plugin, cloudClient, api := newMCPToolsTestPlugin(t, []*Installation{target})
		session, cleanup := connectMCPToolsClient(t, plugin, "other")
		defer cleanup()

		result, err := callMCPTool(t, session, mcpDeleteInstallationToolName, map[string]any{"name": "deleteme", "confirm_name": "DeleteMe"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "no installation with the name deleteme found")

		ownerSession, ownerCleanup := connectMCPToolsClient(t, plugin, "owner")
		defer ownerCleanup()
		cloudClient.deleteErr = errors.New("provisioner delete failed")
		result, err = callMCPTool(t, ownerSession, mcpDeleteInstallationToolName, map[string]any{"name": "deleteme", "confirm_name": "DeleteMe"})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, mcpToolText(t, result), "provisioner delete failed")
		api.AssertNotCalled(t, "KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything)
	})
}

func TestMCPLifecycleResultSensitivity(t *testing.T) {
	install := serviceTestInstall("secret-id", "Secret", "owner")
	install.License = "raw-license-value"
	install.MattermostEnv = cloud.EnvVarMap{"SECRET": cloud.EnvVar{Value: "mattermost-secret"}}
	install.PriorityEnv = cloud.EnvVarMap{"PRIORITY_SECRET": cloud.EnvVar{Value: "priority-secret"}}
	plugin, _, _ := newMCPToolsTestPlugin(t, []*Installation{install})
	session, cleanup := connectMCPToolsClient(t, plugin, "owner")
	defer cleanup()

	result, err := callMCPTool(t, session, mcpRestartInstallationToolName, map[string]any{"name": "secret"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	assertMCPResultRedacted(t, result, "raw-license-value", "mattermost-secret", "priority-secret", "MattermostEnv", "PriorityEnv", "License")

	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(data), "CLOUD_PLUGIN_RESTART")
}
